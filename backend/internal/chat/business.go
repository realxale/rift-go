package chats

import (
	"backend/internal/auth"
	"backend/internal/moderation"
	"backend/internal/profiles"
	"backend/pkg/database"
	"encoding/json"
	"time"
	"errors"
)
// ManageRoomService обрабатывает запросы на управление комнатой (вход/выход/подпись)
// Принимает запрос на подпись или выход из комнаты
// Возвращает ошибку, если операция не удалась
func ManageRoomService(req SignRoomRequest) error {
	// Парсим JWT
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return err
	}

	// Выполняем операцию в базе данных
	err, res := manageRoomDB(req.RoomName, req.Token, parsed.Username, string(req.Move))
	if err != nil {
		return err
	}
	// Если операция успешна, но результат false - возвращаем nil
	if !res {
		return nil
	}
	return nil
}

// sendService обрабатывает отправку сообщения в комнату
// Проверяет, что текст сообщения не пустой, и вызывает функцию записи в БД
// Возвращает ошибку и флаг успеха (true - сообщение отправлено, false - пропущено)
func sendService(req SendReq) (error, bool) {
	// Проверяем, что текст сообщения не пустой
	if req.Text == "" {
		return nil, false
	}

	// Парсим JWT
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return err, false
	}

	// Отправляем сообщение в базу данных
	err = sendDB(req.RoomName, req.Text, parsed.Username)
	if err != nil {
		return err, false
	}
	return nil, true
}

// RoomInfo структура для списка комнат
type RoomInfo struct {
	RoomName   string `json:"room_name"`
	RoomType   string `json:"room_type"`
	AccessType string `json:"access_type"`
	Role       string `json:"role"`
}

// roomsListService возвращает список комнат пользователя
func roomsListService(jwt string) ([]RoomInfo, error) {
	parsed, err := auth.ParseJWT(jwt)
	if err != nil {
		return nil, err
	}
	rows, err := database.GetUserRoomsDB(parsed.Username)
	if err != nil {
		return nil, err
	}
	rooms := make([]RoomInfo, 0, len(rows))
	for _, r := range rows {
		rooms = append(rooms, RoomInfo{
			RoomName:   r.RoomName,
			RoomType:   r.RoomType,
			AccessType: r.AccessType,
			Role:       r.Role,
		})
	}
	return rooms, nil
}

// Sync структура для синхронизации сообщений
// Представляет одно сообщение для отправки клиенту при синхронизации
type Sync struct {
	Text      string    `json:"text"`       // Текст сообщения
	Username  string    `json:"username"`   // Имя пользователя, отправившего сообщение
	RoomName  string    `json:"room_name"`  // Название комнаты
	CreatedAt time.Time `json:"created_at"` // Время создания сообщения
}

// syncService выполняет синхронизацию сообщений для клиента
// Принимает запрос с JWT токеном и временем последней синхронизации
// Возвращает ошибку и список новых сообщений
func syncService(req SyncRequest) (error, []Sync) {
	// Парсим JWT
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return err, nil
	}

	// Парсим время из строки RFC3339
	lastTime, err := time.Parse(time.RFC3339, req.LastTime)
	if err != nil {
		return err, nil
	}

	// Получаем новые сообщения из базы данных (начиная с lastTime)
	err, msg := selectMessages(parsed.Username, lastTime)
	if err != nil {
		return err, nil
	}
	return nil, msg
}

// manager главный обработчик WebSocket сообщений
// Принимает сообщение типа WSM и канал для отправки ответов клиенту
// Декодирует Payload в соответствующий тип запроса и вызывает нужную логику
func manager(c *WSM, Send chan any) error {
	// Определяем тип входящего сообщения
	switch c.Type {
	// Обработка создания новой комнаты
	case "room_create":
		var req CreateRoomRequest
		// Декодируем Payload в структуру CreateRoomRequest
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		// Парсим JWT токен для получения имени пользователя
		parsed, err := auth.ParseJWT(req.JWT)
		if err != nil {
			return err
		}

		// Создаем комнату в базе данных
			err = repostitoryCreateDB(req.RoomName, string(req.RoomType), string(req.AccessType), parsed.Username)
			if err != nil {
				return err
			}
		// Отправляем клиенту код успеха 101
		Send <- 101
		return nil

	// Обработка отправки сообщения
	case "send":
		var req SendReq
		// Декодируем Payload в структуру SendReq
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		// Вызываем сервис отправки сообщения
		err, ok := sendService(req)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("defected message")
		}
		// Отправляем клиенту код успеха 101
		Send <- 101
		return nil

	// Обработка управления комнатой (вход/выход/подпись)
	case "manage_room":
		var req SignRoomRequest
		// Декодируем Payload в структуру SignRoomRequest
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		// Выполняем операцию в базе данных
		err = ManageRoomService(req)
		if err != nil {
			return err
		}
		// Отправляем клиенту код успеха 101
		Send <- 101
		return nil

	case "moderation":
		var req moderation.ModerationRequest
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		msg, err := moderation.HandleModeration(req)
		if err != nil {
			return err
		}
		Send <- msg
		return nil

	// Обработка синхронизации сообщений
	case "sync":
		var req SyncRequest
		// Декодируем Payload в структуру SyncRequest
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		// Получаем новые сообщения
		err, msgs := syncService(req)
		if err != nil {
			return err
		}
		// Отправляем клиенту список новых сообщений
		Send <- msgs
		return nil

	// Обработка изменения профиля
	case "profile_change":
		var req profiles.ProfileUpdateRequest
		err := json.Unmarshal(c.Payload, &req)
		if err != nil {
			return err
		}

		err = profiles.ProfileUpdateService(req)
		if err != nil {
			return err
		}
		Send <- 101
		return nil
	}
	// Если тип сообщения не распознан, просто выходим без ошибки
	return nil
}
