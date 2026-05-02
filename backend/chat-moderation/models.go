package moderation

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

// ModerationRequest запрос на модерацию из WebSocket/HTTP
type ModerationRequest struct {
	JWT      string `json:"jwt" binding:"required"`       // JWT токен авторизации
	RoomName string `json:"room_name" binding:"required"` // имя комнаты
	Target   string `json:"target"`                        // username, текст сообщения, или пусто (для удаления комнаты)
	Action   string `json:"action"`                        // ban, mute, kick, deleteroom, deletemsg
}