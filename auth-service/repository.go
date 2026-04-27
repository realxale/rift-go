package auth

import (
	"backend/config"
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var dbURL = config.GetEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/auth_db?sslmode=disable")

// Глобальный пул соединений
var db *pgxpool.Pool

func ConnectDB() *pgx.Conn {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to connect:", err)
	}
	return conn
}

func NewPool() *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to create pool:", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping db:", err)
	}

	db = pool
	return pool
}

func InitDB(conn *pgx.Conn) error {
	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

func RegUserDB(username, password string) error {
	if db == nil {
		return errors.New("database pool is not initialized")
	}

	query := "INSERT INTO users (username, password_hash) VALUES ($1, $2)"
	_, err := db.Exec(context.Background(), query, username, password)
	if err != nil {
		return err
	}

	return nil
}

func AuthUserDB(req *RegRequest) (error, bool) {
	if db == nil {
		return errors.New("database pool is not initialized"), false
	}

	var passwordHash string
	query := "SELECT password_hash FROM users WHERE username = $1"

	err := db.QueryRow(context.Background(), query, req.Username).Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return err, false
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.Password),
	)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, false
		}
		return err, false
	}

	return nil, true
}
func InitAuthDB() {
	conn := ConnectDB()
	defer conn.Close(context.Background())

	if err := InitDB(conn); err != nil {
		log.Fatal("failed to init auth db:", err)
	}

	NewPool() // создаёт пул и кладёт в db
}
