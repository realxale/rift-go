package chats

import (
	"backend/config"
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Максимальное количество одновременных воркеров (соединений)
var WORKERS_SOCKETS = configuredSocketWorkers()

// Мьютекс для потокобезопасного изменения счетчика WORKERS_SOCKETS
var mu sync.Mutex

func configuredSocketWorkers() int {
	workers, err := strconv.Atoi(config.GetEnv("CHAT_WORKERS_SOCKETS", "200"))
	if err != nil || workers <= 0 {
		log.Println("invalid CHAT_WORKERS_SOCKETS, using default 200")
		return 200
	}
	return workers
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
	// Безопасно уменьшаем счетчик активных воркеров
	defer conn.Close()
	mu.Lock()
	if WORKERS_SOCKETS <= 0 {
		mu.Unlock()
		// Превышен лимит воркеров - отправляем ошибку и закрываем соединение
		errMsg := map[string]string{"error": "too many workers"}
		if err := conn.WriteJSON(errMsg); err != nil {
			log.Println("failed to write error:", err)
		}
		return
	}
	WORKERS_SOCKETS = WORKERS_SOCKETS - 1
	mu.Unlock()
	defer func() {
		mu.Lock()
		WORKERS_SOCKETS = WORKERS_SOCKETS + 1
		mu.Unlock()
	}()

	// Устанавливаем таймаут на чтение (60 секунд)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Обработчик Pong-сообщений для поддержания соединения живым
	conn.SetPongHandler(func(string) error {
		// При получении pong обновляем таймаут
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
	// Таймер для периодической отправки Ping сообщений (каждые 30 секунд)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

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

		// Отправляем Ping для поддержания соединения
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("ping error:", err)
				return
			}

		// Получен сигнал остановки
		case <-ctx.Done():
			log.Println("writer context done")
			return
		}
	}
}
