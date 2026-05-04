package chats

import (
	"backend/pkg/database"
	"time"
)

// repostitoryCreateDB прокси к database.CreateRoomDB
func repostitoryCreateDB(roomName, roomType, accessType, username string) error {
	return database.CreateRoomDB(roomName, roomType, accessType, username)
}

// manageRoomDB обрабатывает вход/выход из комнаты
// Возвращает (error, ok)
func manageRoomDB(roomName, token, username, move string) (error, bool) {
	if move == "sign" {
		return database.SignRoomDB(roomName, token, username)
	}
	if move == "leave" {
		err := database.LeaveRoomDB(roomName, username)
		return err, err == nil
	}
	return nil, false
}

// sendDB прокси к database.SendMessageDB
func sendDB(roomName, text, username string) error {
	return database.SendMessageDB(roomName, text, username)
}

// selectMessages прокси к database.SelectMessages
// Возвращает слайс Sync (структура бизнес-слоя)
func selectMessages(username string, lastTime time.Time) (error, []Sync) {
	err, rows := database.SelectMessages(username, lastTime)
	if err != nil {
		return err, nil
	}

	msgs := make([]Sync, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, Sync{
			Text:      r.Text,
			Username:  r.Username,
			RoomName:  r.RoomName,
			CreatedAt: r.CreatedAt,
		})
	}
	return nil, msgs
}

// InitChatDB инициализирует БД для chat-сервиса
func InitChatDB() {
	database.InitAllTables()
}