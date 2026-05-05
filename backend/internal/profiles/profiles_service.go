package profiles

import (
	"backend/internal/auth"
	"backend/pkg/database"
	"fmt"
)

// ProfileUpdateService обрабатывает запрос на обновление профиля
// Парсит JWT, определяет действие и вызывает соответствующий метод БД напрямую
func ProfileUpdateService(req ProfileUpdateRequest) error {
	// Парсим JWT для получения username
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	user := &database.User{Username: parsed.Username}

	// Определяем действие и вызываем нужный метод БД
	switch req.Action {
	case ActionChangeNickname:
		if req.NewNickname == "" {
			return fmt.Errorf("new_nickname is required for change_nickname action")
		}
		err = user.ChangeNickcname(req.NewNickname)
		if err != nil {
			return fmt.Errorf("failed to change nickname: %w", err)
		}

	case ActionChangeFamilyStatus:
		if req.FamilyStatus == "" {
			return fmt.Errorf("family_status is required for change_family_status action")
		}
		err = user.ChangeFamilyStatus(req.FamilyStatus)
		if err != nil {
			return fmt.Errorf("failed to change family status: %w", err)
		}

	case ActionChangeLinks:
		links := database.Links{
			Discord:  req.Links.Discord,
			Telegram: req.Links.Telegram,
			Other:    req.Links.Other,
		}
		err = user.ChangeLinks(links)
		if err != nil {
			return fmt.Errorf("failed to change links: %w", err)
		}

	default:
		return fmt.Errorf("unknown action: %s", req.Action)
	}

	return nil
}
