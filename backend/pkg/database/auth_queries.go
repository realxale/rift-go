package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// RegUserDB регистрирует нового пользователя
func RegUserDB(username, password string) error {
	if Pool == nil {
		return errors.New("database pool is not initialized")
	}

	query := "INSERT INTO users (username, password_hash) VALUES ($1, $2)"
	_, err := Pool.Exec(context.Background(), query, username, password)
	if err != nil {
		return err
	}

	return nil
}

// AuthUserDB аутентифицирует пользователя
func AuthUserDB(username, password string) (error, bool) {
	if Pool == nil {
		return errors.New("database pool is not initialized"), false
	}

	var passwordHash string
	query := "SELECT password_hash FROM users WHERE username = $1"

	err := Pool.QueryRow(context.Background(), query, username).Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return err, false
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, false
		}
		return err, false
	}

	return nil, true
}