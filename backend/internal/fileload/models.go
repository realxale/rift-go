package fileload

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================
// WebSocket сообщения
// ============================================================

// WSMessage — входящее сообщение от клиента
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// UploadStartPayload — начало загрузки файла
type UploadStartPayload struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	RoomName string `json:"room_name"`
}

// ChunkPayload — чанк файла (base64)
type ChunkPayload struct {
	Index int    `json:"index"`
	Data  string `json:"data"`
}

// WSResponse — ответ сервера клиенту
type WSResponse struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Path    string `json:"path,omitempty"`
}

// ============================================================
// Состояние сессии загрузки
// ============================================================

// UploadSession — состояние активной загрузки файла
type UploadSession struct {
	mu        sync.Mutex
	FileName  string
	FileSize  int64
	RoomName  string
	Username  string
	TempFile  *FileBuffer
	Received  int64
	ChunkSize int64
	StartedAt time.Time
}

// FileBuffer — буфер для временного хранения данных файла
type FileBuffer struct {
	Data   []byte
	Path   string
	Closed bool
}

// NewFileBuffer создаёт новый буфер файла
func NewFileBuffer(path string) *FileBuffer {
	return &FileBuffer{
		Data:   make([]byte, 0),
		Path:   path,
		Closed: false,
	}
}

// Write записывает данные в буфер
func (fb *FileBuffer) Write(p []byte) (int, error) {
	if fb.Closed {
		return 0, nil
	}
	fb.Data = append(fb.Data, p...)
	return len(p), nil
}

// Close закрывает буфер
func (fb *FileBuffer) Close() error {
	fb.Closed = true
	return nil
}

// ============================================================
// Глобальное состояние
// ============================================================

var (
	sessions   = make(map[string]*UploadSession)
	sessionsMu sync.Mutex
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)