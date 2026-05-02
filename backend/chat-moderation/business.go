package moderation

import (
	"backend/auth-service"
	"backend/database"
	"errors"
)

// HandleModeration выполняет действие модерации
// Принимает запрос от WebSocket/HTTP, парсит JWT, проверяет права, выполняет действие
func HandleModeration(req ModerationRequest) (string, error) {
	// Парсим JWT модератора
	claims, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return "", errors.New("invalid JWT: " + err.Error())
	}

	// Проверяем права модератора
	role, err := database.GetMemberRole(req.RoomName, claims.Username)
	if err != nil {
		return "", errors.New("user is not a member of the room: " + err.Error())
	}

	// Только owner и admin могут выполнять действия модерации
	if role != "owner" && role != "admin" {
		return "", errors.New("insufficient permissions")
	}

	switch req.Action {
	case "ban":
		return banUser(req.RoomName, req.Target)
	case "kick":
		return kickUser(req.RoomName, req.Target)
	case "mute":
		return muteUser(req.RoomName, req.Target)
	case "deleteroom":
		return deleteRoom(req.RoomName)
	case "deletemsg":
		return deleteMessage(req.Target)
	default:
		return "", errors.New("unknown action: " + req.Action)
	}
}

// banUser банит пользователя в комнате
func banUser(roomName, target string) (string, error) {
	if target == "" {
		return "", errors.New("target username is required for ban")
	}
	err := database.UpdateMemberStatus(roomName, target, "banned")
	if err != nil {
		return "", errors.New("failed to ban user: " + err.Error())
	}
	return "user banned successfully", nil
}

// kickUser кикает пользователя из комнаты
func kickUser(roomName, target string) (string, error) {
	if target == "" {
		return "", errors.New("target username is required for kick")
	}
	err := database.LeaveRoomDB(roomName, target)
	if err != nil {
		return "", errors.New("failed to kick user: " + err.Error())
	}
	return "user kicked successfully", nil
}

// muteUser мутит пользователя в комнате
func muteUser(roomName, target string) (string, error) {
	if target == "" {
		return "", errors.New("target username is required for mute")
	}
	err := database.UpdateMemberStatus(roomName, target, "muted")
	if err != nil {
		return "", errors.New("failed to mute user: " + err.Error())
	}
	return "user muted successfully", nil
}

// deleteRoom удаляет комнату
func deleteRoom(roomName string) (string, error) {
	if roomName == "" {
		return "", errors.New("room name is required to delete room")
	}
	err := database.DeleteRoom(roomName)
	if err != nil {
		return "", errors.New("failed to delete room: " + err.Error())
	}
	return "room deleted successfully", nil
}

// deleteMessage удаляет сообщение
func deleteMessage(text string) (string, error) {
	if text == "" {
		return "", errors.New("message text is required to delete message")
	}
	err := database.DeleteMessage(text)
	if err != nil {
		return "", errors.New("failed to delete message: " + err.Error())
	}
	return "message deleted successfully", nil
}