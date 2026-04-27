package chats

import (
	"backend/auth-service"
	"backend/config"
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var dbURL = config.GetEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/auth_db?sslmode=disable")

var db *pgxpool.Pool

func connectDB() *pgx.Conn {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to connect:", err)
	}
	return conn
}

func newPool() *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to create pool:", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping db:", err)
	}
	return pool
}

func initDB(conn *pgx.Conn) error {
	ctx := context.Background()

		_, err := conn.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS rooms (
				id SERIAL PRIMARY KEY,
				room_name TEXT UNIQUE NOT NULL,
				room_type TEXT NOT NULL,
				access_type TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT NOW()
			)
		`)
	if err != nil {
		return err
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
		return err
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
		return err
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
		return err
	}

	return nil
}

func repostitoryCreateDB(RoomName string, RoomType RoomType, AccessType AccessType, Username string) error {
	ctx := context.Background()

	query := "INSERT INTO rooms(room_name, room_type, access_type) VALUES ($1, $2, $3)"
	query2 := "INSERT INTO members(username, role, status, room_name) VALUES ($1, $2, $3, $4)"

	_, err := db.Exec(ctx, query, RoomName, RoomType, AccessType)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, query2, Username, "owner", "in", RoomName)
	if err != nil {
		return err
	}

	return nil
}

func manageRoomDB(req SignRoomRequest) (error, bool) {
	ctx := context.Background()

	if req.Move == "sign" {
		insertQuery := "INSERT INTO members(username, role, status, room_name) VALUES($1, $2, $3, $4)"
		accessTypeQuery := "SELECT access_type FROM rooms WHERE room_name = $1"

		var accessType string
		err := db.QueryRow(ctx, accessTypeQuery, req.RoomName).Scan(&accessType)
		if err != nil {
			return err, false
		}

		if accessType == string(AccessPrivate) {
			tokenQuery := "SELECT token FROM tokens WHERE room_name = $1"

			var token string
			err := db.QueryRow(ctx, tokenQuery, req.RoomName).Scan(&token)
			if err != nil {
				return err, false
			}

			if token != req.Token {
				return nil, false
			}
		}

		par, err := auth.ParseJWT(req.JWT)
		if err != nil {
			return err, false
		}

		memberCheckQuery := "SELECT 1 FROM members WHERE username = $1 AND room_name = $2"
		var exists int
		err = db.QueryRow(ctx, memberCheckQuery, par.Username, req.RoomName).Scan(&exists)
		if err == nil {
			return nil, false
		}
		if err != pgx.ErrNoRows {
			return err, false
		}

		_, err = db.Exec(ctx, insertQuery, par.Username, "default", "in", req.RoomName)
		if err != nil {
			return err, false
		}

		return nil, true
	}

	if req.Move == "leave" {
		parsed, err := auth.ParseJWT(req.JWT)
		if err != nil {
			return err, false
		}

		query := "DELETE FROM members WHERE username = $1 AND room_name = $2"
		_, err = db.Exec(ctx, query, parsed.Username, req.RoomName)
		if err != nil {
			return err, false
		}

		return nil, true
	}

	return nil, false
}

func sendDB(req SendReq) error {
	ctx := context.Background()
	parsed, err := auth.ParseJWT(req.JWT)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO messages(text, username, room_name)
		SELECT $1, username, room_name
		FROM members
		WHERE username = $2
		  AND room_name = $3
	`
	tag, err := db.Exec(ctx, query, req.Text, parsed.Username, req.RoomName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user is not a member of the room")
	}

	return nil
}
func InitChatDB() {
	conn := connectDB()
	defer conn.Close(context.Background())

	if err := initDB(conn); err != nil {
		log.Fatal("failed to init chat db:", err)
	}

	db = newPool()
}
func selectMessages(jwt string, last_time time.Time) (error, []Sync) {
	parsed, err := auth.ParseJWT(jwt)
	if err != nil {
		return err, nil
	}

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
	rows, err := db.Query(context.Background(), query, parsed.Username, last_time)
	if err != nil {
		return err, nil
	}
	defer rows.Close()
	var msgs []Sync
	for rows.Next() {
		var msg Sync
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
