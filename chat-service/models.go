package chats

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"
	"github.com/gorilla/websocket"
)
//create semaphor channel for limits in reader/writer
var connsem chan struct{}

func InitLimit() {
    limit := os.Getenv("LIMIT")
    if limit == "" {
        limit = "100"
    }   
    limitInt, err := strconv.Atoi(limit)
    if err != nil {
        log.Printf("Ошибка конвертации LIMIT='%s', using default 100: %v", limit, err)
        limitInt = 100
    }
    
    connsem = make(chan struct{}, limitInt)
    log.Printf("Connection semaphore initialized with limit: %d", limitInt)
}
// Move определяет тип действия в комнате
type Move string

const (
	Leave Move = "leave" // покинуть комнату
	Sign  Move = "sign"  // подписать документ
)

// SignRoomRequest запрос на подпись/выход из комнаты
type SignRoomRequest struct {
	JWT      string `json:"jwt" binding:"required"`       // JWT токен авторизации
	RoomName string `json:"room_name" binding:"required"` // имя комнаты
	Token    string `json:"token"`                        // опциональный токен
	Move     Move   `json:"move" binding:"required"`      // действие (leave/sign)
}

// CreateRoomRequest запрос на создание новой комнаты
type CreateRoomRequest struct {
	JWT        string     `json:"jwt" binding:"required"`         // JWT токен авторизации
	RoomName   string     `json:"room_name" binding:"required"`   // имя новой комнаты
	RoomType   RoomType   `json:"room_type" binding:"required"`   // тип комнаты
	AccessType AccessType `json:"access_type" binding:"required"` // тип доступа
	Move       Move       `json:"move"`                           // опциональное действие
}

// SendReq запрос на отправку сообщения в комнату
type SendReq struct {
	JWT      string `json:"jwt" binding:"required"`       // JWT токен авторизации
	RoomName string `json:"room_name" binding:"required"` // имя комнаты
	Text     string `json:"text" binding:"required"`      // текст сообщения
}

// RoomType тип комнаты
type RoomType string

const (
	RoomChannel RoomType = "channel" // канал (публичная комната)
	Room1v1     RoomType = "1v1"     // приватный чат 1 на 1
	RoomGroup   RoomType = "group"   // групповая комната
)

// AccessType тип доступа к комнате
type AccessType string

const (
	AccessPublic  AccessType = "public"  // публичный доступ
	AccessPrivate AccessType = "private" // приватный доступ
)

// Role роль пользователя в комнате
type Role string

const (
	Owner   Role = "owner"   // владелец комнаты
	Admin   Role = "admin"   // администратор
	Default Role = "default" // обычный участник
)

// WSM структура WebSocket сообщения
type WSM struct {
	Type    string          `json:"type"`    // тип сообщения
	Payload json.RawMessage `json:"payload"` // полезная нагрузка в формате JSON
}

// Client представляет подключенного клиента
type Client struct {
	Conn *websocket.Conn // WebSocket соединение
	Code int16           // код статуса/ошибки
}

// SyncRequest запрос на синхронизацию данных
type SyncRequest struct {
	JWT      string    `json:"jwt" binding:"required"`       // JWT токен
	LastTime time.Time `json:"last_time" binding:"required"` // время последней синхронизации
}

// ChannelCommand команда для управления каналом
type ChannelCommand struct {
	Code    int   // код команды
	Err     error // ошибка (если есть)
	Command int   // тип команды: 1 - закрыть, 2 - пинг, 3 - другое
}

// Reader главная горутина для чтения сообщений от клиента
// Обрабатывает входящие WebSocket сообщения и управляет соединением
func Reader(conn *websocket.Conn) {
	defer conn.Close() // гарантия смерти соединения и writer

	// Пытаемся захватить слот
	select {
	case connsem <- struct{}{}:
		// Слот захвачен - обрабатываем соединение
		defer func() { <-connsem }() // освободим слот при выходе
	default:
		// Слоты закончились - отправляем ошибку
		errMsg := map[string]string{"error": "too many workers"}
		if err := conn.WriteJSON(errMsg); err != nil {
			log.Println("failed to write error:", err)
		}
		return
	}

	// Устанавливаем таймаут на чтение (60 секунд)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Обработчик Pong-сообщений для поддержания соединения живым
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Буферизированный канал для отправки сообщений клиенту
	send := make(chan interface{}, 256)

	// Создаем контекст с возможностью отмены для управления горутинами
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Отменяем контекст при выходе из функции

	// Запускаем горутину писателя
	go Writer(conn, send, ctx)

	// Основной цикл чтения сообщений
	for {
		var msg WSM

		// Читаем JSON сообщение из WebSocket
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			cancel() // Сигналим писателю об остановке
			return
		}

		// Передаем сообщение в менеджер для обработки
		if err := manager(&msg, send); err != nil {
			log.Println("manager error:", err)
			cancel() // При ошибке останавливаем писателя
			return
		}
	}
}

// Writer горутина для отправки сообщений клиенту
// conn - WebSocket соединение
// send - канал для получения сообщений на отправку
// ctx - контекст для graceful остановки
func Writer(conn *websocket.Conn, send <-chan interface{}, ctx context.Context) {
	for {
		select {
		// Получаем сообщение из канала на отправку
		case msg, ok := <-send:
			if !ok {
				// Канал закрыт - выходим из горутины
				return
			}
			// Отправляем сообщение клиенту
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("write error:", err)
				return
			}

		// Получен сигнал остановки
		case <-ctx.Done():
			log.Println("writer context done")
			return
		}
	}
}