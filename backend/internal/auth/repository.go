package auth

import (
	"backend/pkg/database"
)

// RegUserDB прокси к database.RegUserDB
func RegUserDB(username, password string) error {
	return database.RegUserDB(username, password)
}

// AuthUserDB прокси к database.AuthUserDB
func AuthUserDB(username, password string) (error, bool) {
	return database.AuthUserDB(username, password)
}

// InitAuthDB инициализирует БД для auth-сервиса
func InitAuthDB() {
	database.InitAllTables()
}