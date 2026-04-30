package database

import (
	"backend/config"
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbURL = config.GetEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/auth_db?sslmode=disable")

// Глобальный пул соединений
var Pool *pgxpool.Pool

// ConnectDB создаёт одно подключение
func ConnectDB() *pgx.Conn {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to connect:", err)
	}
	return conn
}

// NewPool создаёт пул соединений
func NewPool() *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to create pool:", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping db:", err)
	}

	Pool = pool
	return pool
}

// InitAllTables создаёт все таблицы в БД
func InitAllTables() {
	conn := ConnectDB()
	defer conn.Close(context.Background())

	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("failed to create users table:", err)
	}

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rooms (
			id SERIAL PRIMARY KEY,
			room_name TEXT UNIQUE NOT NULL,
			room_type TEXT NOT NULL,
			access_type TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("failed to create rooms table:", err)
	}

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS members (
			room_name TEXT REFERENCES rooms(room_name),
			username TEXT REFERENCES users(username),
			status TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE (room_name, username)
		)
	`)
	if err != nil {
		log.Fatal("failed to create members table:", err)
	}

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tokens (
			id INT REFERENCES rooms(id),
			status TEXT NOT NULL,
			role TEXT NOT NULL,
			token BIGINT UNIQUE,
			room_name TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("failed to create tokens table:", err)
	}

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			text TEXT NOT NULL,
			username TEXT REFERENCES users(username),
			room_name TEXT REFERENCES rooms(room_name),
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("failed to create messages table:", err)
	}

	// Создаём пул после инициализации таблиц
	NewPool()
}