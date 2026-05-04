package fileload

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/pkg/config"
	"backend/pkg/database"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================================
// JWT
// ============================================================

// Claims — структура JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte(config.GetEnv("JWT_SECRET", "dasdasdwefafdsaefafdsaf"))

// ParseJWT парсит и валидирует JWT токен
func ParseJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// ============================================================
// Бизнес-логика загрузки
// ============================================================

// BusinessStartUpload проверяет параметры и создаёт сессию загрузки
func BusinessStartUpload(payload UploadStartPayload, username string) (*UploadSession, error) {
	if payload.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	if payload.FileSize <= 0 {
		return nil, fmt.Errorf("file_size must be > 0")
	}
	maxSize := int64(config.GetEnvInt("MAX_FILE_SIZE", 100*1024*1024))
	if payload.FileSize > maxSize {
		return nil, fmt.Errorf("file too large, max %d bytes", maxSize)
	}
	if payload.RoomName == "" {
		return nil, fmt.Errorf("room_name is required")
	}

	uploadDir := config.GetEnv("UPLOAD_DIR", "./uploads")
	chunkSize := config.GetEnvInt("CHUNK_SIZE", 1024*1024)

	// Безопасное имя файла
	safeName := SanitizeFileName(payload.FileName)
	roomDir := filepath.Join(uploadDir, SanitizeFileName(payload.RoomName))
	if err := os.MkdirAll(roomDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create room dir: %w", err)
	}

	// Уникальное имя файла
	finalPath := filepath.Join(roomDir, safeName)
	if _, err := os.Stat(finalPath); err == nil {
		ext := filepath.Ext(safeName)
		base := strings.TrimSuffix(safeName, ext)
		finalPath = filepath.Join(roomDir, fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext))
	}

	buf := NewFileBuffer(finalPath)

	session := &UploadSession{
		FileName:  filepath.Base(finalPath),
		FileSize:  payload.FileSize,
		RoomName:  payload.RoomName,
		Username:  username,
		TempFile:  buf,
		ChunkSize: int64(chunkSize),
		StartedAt: time.Now(),
	}

	sessionsMu.Lock()
	sessions[username] = session
	sessionsMu.Unlock()

	return session, nil
}

// BusinessProcessChunk обрабатывает чанк файла
func BusinessProcessChunk(session *UploadSession, payload ChunkPayload) (float64, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	decoded, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return 0, fmt.Errorf("invalid base64 data: %w", err)
	}

	if int64(len(decoded)) > session.ChunkSize {
		return 0, fmt.Errorf("chunk too large, max %d bytes", session.ChunkSize)
	}

	n, err := session.TempFile.Write(decoded)
	if err != nil {
		return 0, fmt.Errorf("write error: %w", err)
	}
	session.Received += int64(n)

	progress := float64(session.Received) / float64(session.FileSize) * 100
	return progress, nil
}

// BusinessFinishUpload завершает загрузку и сохраняет в БД
func BusinessFinishUpload(session *UploadSession) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Received != session.FileSize {
		return "", fmt.Errorf("file size mismatch: got %d, expected %d", session.Received, session.FileSize)
	}

	// Сохраняем буфер на диск
	if err := os.WriteFile(session.TempFile.Path, session.TempFile.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Определяем расширение и тип
	ext := filepath.Ext(session.FileName)
	fileType := DetectFileType(ext)

	// Сохраняем в БД
	if err := database.InsertFile(session.FileName, ext, fileType, session.TempFile.Path, session.Username, session.RoomName, session.FileSize); err != nil {
		log.Printf("failed to save file metadata to DB: %v", err)
		// Не фатально — файл уже сохранён
	}

	return session.TempFile.Path, nil
}

// BusinessCancelUpload отменяет загрузку
func BusinessCancelUpload(session *UploadSession) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.TempFile.Close()
}

// ============================================================
// Вспомогательные функции
// ============================================================

// SanitizeFileName очищает имя файла от path traversal
func SanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	return name
}

// DetectFileType определяет тип файла по расширению
func DetectFileType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".mp3", ".wav", ".ogg", ".flac", ".aac", ".wma":
		return "audio"
	case ".mp4", ".avi", ".mkv", ".mov", ".wmv", ".webm":
		return "video"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp":
		return "image"
	case ".pdf":
		return "pdf"
	case ".doc", ".docx":
		return "document"
	case ".zip", ".rar", ".7z", ".tar", ".gz":
		return "archive"
	case ".txt", ".md":
		return "text"
	default:
		return "other"
	}
}

// CleanupSession удаляет сессию из глобальной мапы
func CleanupSession(session *UploadSession) {
	if session == nil {
		return
	}
	sessionsMu.Lock()
	delete(sessions, session.Username)
	sessionsMu.Unlock()
}