package moderation

import (
	"backend/auth-service"
	"backend/database"
	"errors"
)

// ActionInf содержит информацию о действии
type ActionInf struct {
	Target string // text msg, room_name, username
	Action string // ban, mute, kick, deleteroom, deletemsg
}

// Access определяет права доступа
type Access struct {
	Ban        bool // ban access
	Kick       bool // kick access
	MuteAccess bool // mute access
	DeleteRoom bool // delete room access
	DeleteMsg  bool // delete message access
}

// Actor представляет модератора, выполняющего действие
type Actor struct {
	JWT         string
	RoomName    string
	Permissions Access
	ActionInf   ActionInf
}

// Ban банит пользователя в комнате
func Ban(actorJWT, roomName, target string) (string, error) {
	return performAction(actorJWT, roomName, target, "ban")
}

// Kick кикает пользователя из комнаты
func Kick(actorJWT, roomName, target string) (string, error) {
	return performAction(actorJWT, roomName, target, "kick")
}

// Mute мутит пользователя в комнате
func Mute(actorJWT, roomName, target string) (string, error) {
	return performAction(actorJWT, roomName, target, "mute")
}

// DeleteRoom удаляет комнату
func DeleteRoom(actorJWT, roomName string) (string, error) {
	return performAction(actorJWT, roomName, "", "deleteroom")
}

// DeleteMsg удаляет сообщение
func DeleteMsg(actorJWT, roomName, message string) (string, error) {
	return performAction(actorJWT, roomName, message, "deletemsg")
}

// performAction выполняет действие модерации
func performAction(actorJWT, roomName, target, action string) (string, error) {
	// Парсим JWT модератора
	claims, err := auth.ParseJWT(actorJWT)
	if err != nil {
		return "", errors.New("invalid JWT: " + err.Error())
	}

	// Проверяем права модератора
	role, err := database.GetMemberRole(roomName, claims.Username)
	if err != nil {
		return "", errors.New("user is not a member of the room: " + err.Error())
	}

	// Только owner и admin могут выполнять действия модерации
	if role != "owner" && role != "admin" {
		return "", errors.New("insufficient permissions")
	}

	switch action {
	case "ban":
		if target == "" {
			return "", errors.New("target username is required for ban")
		}
		err := database.UpdateMemberStatus(roomName, target, "banned")
		if err != nil {
			return "", errors.New("failed to ban user: " + err.Error())
		}
		return "user banned successfully", nil

	case "kick":
		if target == "" {
			return "", errors.New("target username is required for kick")
		}
		err := database.LeaveRoomDB(roomName, target)
		if err != nil {
			return "", errors.New("failed to kick user: " + err.Error())
		}
		return "user kicked successfully", nil

	case "mute":
		if target == "" {
			return "", errors.New("target username is required for mute")
		}
		err := database.UpdateMemberStatus(roomName, target, "muted")
		if err != nil {
			return "", errors.New("failed to mute user: " + err.Error())
		}
		return "user muted successfully", nil

	case "deleteroom":
		if roomName == "" {
			return "", errors.New("room name is required to delete room")
		}
		err := database.DeleteRoom(roomName)
		if err != nil {
			return "", errors.New("failed to delete room: " + err.Error())
		}
		return "room deleted successfully", nil

	case "deletemsg":
		if target == "" {
			return "", errors.New("message text is required to delete message")
		}
		err := database.DeleteMessage(target)
		if err != nil {
			return "", errors.New("failed to delete message: " + err.Error())
		}
		return "message deleted successfully", nil

	default:
		return "", errors.New("unknown action: " + action)
	}
}