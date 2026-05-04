package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRow — запись о файле в БД
type FileRow struct {
	ID            int
	FileName      string
	FileExtension string
	FileType      string
	FileSize      int64
	FilePath      string
	Username      string
	RoomName      string
	CreatedAt     time.Time
}

// InsertFile сохраняет информацию о загруженном файле
func InsertFile(fileName, fileExtension, fileType, filePath, username, roomName string, fileSize int64) error {
	query := `
		INSERT INTO files (file_name, file_extension, file_type, file_size, file_path, username, room_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := Pool.Exec(context.Background(), query, fileName, fileExtension, fileType, fileSize, filePath, username, roomName)
	return err
}

// GetRoomFiles получает список файлов в комнате
func GetRoomFiles(roomName string) ([]FileRow, error) {
	query := `
		SELECT id, file_name, file_extension, file_type, file_size, file_path, username, room_name, created_at
		FROM files
		WHERE room_name = $1
		ORDER BY created_at DESC
		LIMIT 100
	`
	rows, err := Pool.Query(context.Background(), query, roomName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRow
	for rows.Next() {
		var f FileRow
		if err := rows.Scan(&f.ID, &f.FileName, &f.FileExtension, &f.FileType, &f.FileSize, &f.FilePath, &f.Username, &f.RoomName, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// GetPool возвращает глобальный пул соединений (для использования в fileload)
func GetPool() *pgxpool.Pool {
	return Pool
}