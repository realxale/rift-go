package database

import (
	"context"
	"sync"
	"time"
)

// Role представляет роль пользователя в комнате
type Role string

const (
	RoleOwner     Role = "owner"
	RoleMember    Role = "member"
	RoleModerator Role = "moderator"
)

type UserInfo interface {
	WhoIm(username string)
	DeleteDeadLine(username string)
}

// Actions определяет интерфейс действий модерации
type Actions interface {
	ChangeRole(username, target, roomName string, finalRole Role)
	Ban(username, target, roomName string)
	Kick(username, target, roomName string)
}

// User представляет информацию о пользователе в комнате
type User struct {
	Username string
	ChatRole Role
	RoomName string
}

// UsersInfo содержит всех активных пользователей
var (
	UsersInfo []User
	usersMu   sync.RWMutex
)

// GetUsersInfo возвращает копию списка активных пользователей (thread-safe)
// WhoIm получает все комнаты и роли пользователя (статус = active, не забанен)
func (u *User) WhoIm() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := "SELECT username, role, room_name FROM members WHERE username = $1 AND status = 'in'"
	rows, err := Pool.Query(ctx, query, u.Username)
	if err != nil {
		return err
	}
	defer rows.Close()

	usersMu.Lock()
	UsersInfo = UsersInfo[:0]
	for rows.Next() {
		var x User
		if err := rows.Scan(&x.Username, &x.ChatRole, &x.RoomName); err != nil {
			usersMu.Unlock()
			return err
		}

		UsersInfo = append(UsersInfo, x)
	}
	usersMu.Unlock()

	return rows.Err()
}
func (u *User) DeleteDeadLine() {

}
