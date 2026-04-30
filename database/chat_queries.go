package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// MessageRow представляет строку сообщения для синхронизации
type MessageRow struct {
	Text      string
	Username  string
	RoomName  string
	CreatedAt time.Time
}

// CreateRoomDB создаёт комнату и добавляет владельца
func CreateRoomDB(roomName, roomType, accessType, username string) error {
	ctx := context.Background()

	query := "INSERT INTO rooms(room_name, room_type, access_type) VALUES ($1, $2, $3)"
	query2 := "INSERT INTO members(username, role, status, room_name) VALUES ($1, $2, $3, $4)"

	_, err := Pool.Exec(ctx, query, roomName, roomType, accessType)
	if err != nil {
		return err
	}

	_, err = Pool.Exec(ctx, query2, username, "owner", "in", roomName)
	if err != nil {
		return err
	}

	return nil
}

// SignRoomDB подписывает пользователя на комнату
// Если комната приватная, проверяет токен
// Возвращает (error, ok)
func SignRoomDB(roomName, token, username string) (error, bool) {
	ctx := context.Background()

	// Получаем тип доступа к комнате
	accessTypeQuery := "SELECT access_type FROM rooms WHERE room_name = $1"
	var accessType string
	err := Pool.QueryRow(ctx, accessTypeQuery, roomName).Scan(&accessType)
	if err != nil {
		return err, false
	}

	// Если комната приватная, проверяем токен
	if accessType == "private" {
		tokenQuery := "SELECT token FROM tokens WHERE room_name = $1"
		var dbToken string
		err := Pool.QueryRow(ctx, tokenQuery, roomName).Scan(&dbToken)
		if err != nil {
			return err, false
		}

		if dbToken != token {
			return nil, false
		}
	}

	// Проверяем, не состоит ли пользователь уже в комнате
	memberCheckQuery := "SELECT 1 FROM members WHERE username = $1 AND room_name = $2"
	var exists int
	err = Pool.QueryRow(ctx, memberCheckQuery, username, roomName).Scan(&exists)
	if err == nil {
		return nil, false // уже в комнате
	}
	if err != pgx.ErrNoRows {
		return err, false
	}

	// Добавляем пользователя в комнату
	insertQuery := "INSERT INTO members(username, role, status, room_name) VALUES($1, $2, $3, $4)"
	_, err = Pool.Exec(ctx, insertQuery, username, "default", "in", roomName)
	if err != nil {
		return err, false
	}

	return nil, true
}

// LeaveRoomDB удаляет пользователя из комнаты
func LeaveRoomDB(roomName, username string) error {
	ctx := context.Background()
	query := "DELETE FROM members WHERE username = $1 AND room_name = $2"
	_, err := Pool.Exec(ctx, query, username, roomName)
	return err
}

// SendMessageDB отправляет сообщение в комнату от имени участника
func SendMessageDB(roomName, text, username string) error {
	ctx := context.Background()

	query := `
		INSERT INTO messages(text, username, room_name)
		SELECT $1, username, room_name
		FROM members
		WHERE username = $2
		  AND room_name = $3
	`
	tag, err := Pool.Exec(ctx, query, text, username, roomName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user is not a member of the room")
	}

	return nil
}

// SelectMessages получает сообщения из комнат пользователя с момента lastTime
func SelectMessages(username string, lastTime time.Time) (error, []MessageRow) {
	query := `
		SELECT m.text, m.username, m.room_name, m.created_at
		FROM messages m
		WHERE m.room_name IN (
			SELECT DISTINCT room_name
			FROM members
			WHERE username = $1
		)
		AND m.created_at > $2
		ORDER BY m.created_at
	`
	rows, err := Pool.Query(context.Background(), query, username, lastTime)
	if err != nil {
		return err, nil
	}
	defer rows.Close()

	var msgs []MessageRow
	for rows.Next() {
		var msg MessageRow
		if err := rows.Scan(&msg.Text, &msg.Username, &msg.RoomName, &msg.CreatedAt); err != nil {
			return err, nil
		}
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return err, nil
	}

	return nil, msgs
}

// GetMemberRole получает роль пользователя в комнате
func GetMemberRole(roomName, username string) (string, error) {
	query := "SELECT role FROM members WHERE room_name = $1 AND username = $2"
	var role string
	err := Pool.QueryRow(context.Background(), query, roomName, username).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

// UpdateMemberStatus обновляет статус участника (ban, mute)
func UpdateMemberStatus(roomName, username, status string) error {
	query := "UPDATE members SET status = $1 WHERE room_name = $2 AND username = $3"
	_, err := Pool.Exec(context.Background(), query, status, roomName, username)
	return err
}

// DeleteMessage удаляет сообщение из комнаты
func DeleteMessage(text string) error {
	query := "DELETE FROM messages WHERE text = $1"
	_, err := Pool.Exec(context.Background(), query, text)
	return err
}

// DeleteRoom удаляет комнату и всех её участников
func DeleteRoom(roomName string) error {
	ctx := context.Background()

	_, err := Pool.Exec(ctx, "DELETE FROM members WHERE room_name = $1", roomName)
	if err != nil {
		return err
	}

	_, err = Pool.Exec(ctx, "DELETE FROM tokens WHERE room_name = $1", roomName)
	if err != nil {
		return err
	}

	_, err = Pool.Exec(ctx, "DELETE FROM messages WHERE room_name = $1", roomName)
	if err != nil {
		return err
	}

	_, err = Pool.Exec(ctx, "DELETE FROM rooms WHERE room_name = $1", roomName)
	if err != nil {
		return err
	}

	return nil
}

// CheckMemberExists проверяет, является ли пользователь участником комнаты
func CheckMemberExists(roomName, username string) (bool, error) {
	query := "SELECT 1 FROM members WHERE room_name = $1 AND username = $2"
	var exists int
	err := Pool.QueryRow(context.Background(), query, roomName, username).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}