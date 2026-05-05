package profiles

// ProfileAction тип действия при обновлении профиля
type ProfileAction string

const (
	ActionChangeNickname    ProfileAction = "change_nickname"      // смена никнейма
	ActionChangeFamilyStatus ProfileAction = "change_family_status" // смена семейного статуса
	ActionChangeLinks       ProfileAction = "change_links"         // смена ссылок
)

// ProfileLinks структура для ссылок профиля
type ProfileLinks struct {
	Discord  string `json:"discord,omitempty"`  // Discord ссылка
	Telegram string `json:"telegram,omitempty"` // Telegram ссылка
	Other    string `json:"other,omitempty"`    // Другая ссылка
}

// ProfileUpdateRequest запрос на обновление профиля
type ProfileUpdateRequest struct {
	JWT          string        `json:"jwt" binding:"required"`              // JWT токен авторизации
	Action       ProfileAction `json:"action" binding:"required"`           // действие
	NewNickname  string        `json:"new_nickname,omitempty"`              // новый никнейм
	FamilyStatus string        `json:"family_status,omitempty"`             // новый семейный статус
	Links        ProfileLinks  `json:"links,omitempty"`                     // новые ссылки
}