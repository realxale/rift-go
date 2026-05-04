package fileload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"backend/pkg/config"
	"backend/pkg/database"

	"github.com/gorilla/websocket"
)

// ============================================================
// WebSocket хендлер — загрузка файла
// ============================================================

// HandleUpload — WebSocket endpoint для загрузки файлов
// Подключается по ws://host/upload?jwt=TOKEN
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	// JWT из заголовка или query
	token := r.Header.Get("JWT")
	if token == "" {
		token = r.URL.Query().Get("jwt")
	}
	if token == "" {
		http.Error(w, "missing jwt", http.StatusUnauthorized)
		return
	}

	claims, err := ParseJWT(token)
	if err != nil {
		http.Error(w, "invalid jwt: "+err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Printf("user %s connected to file upload", claims.Username)

	var session *UploadSession

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			CleanupSession(session)
			return
		}

		switch msg.Type {
		case "upload_start":
			var payload UploadStartPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sendError(conn, "invalid payload: "+err.Error())
				continue
			}

			session, err = BusinessStartUpload(payload, claims.Username)
			if err != nil {
				sendError(conn, err.Error())
				session = nil
				continue
			}

			sendJSON(conn, WSResponse{
				Type:    "upload_started",
				Message: fmt.Sprintf("uploading %s to room %s", payload.FileName, payload.RoomName),
			})

		case "chunk":
			if session == nil {
				sendError(conn, "no active upload, send upload_start first")
				continue
			}

			var payload ChunkPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sendError(conn, "invalid chunk payload")
				continue
			}

			progress, err := BusinessProcessChunk(session, payload)
			if err != nil {
				sendError(conn, err.Error())
				return
			}

			log.Printf("upload %s: %.2f%% (%d/%d bytes)", session.FileName, progress, session.Received, session.FileSize)

			sendJSON(conn, WSResponse{
				Type:    "chunk_ack",
				Message: fmt.Sprintf("chunk %d received, %.2f%% complete", payload.Index, progress),
			})

		case "upload_finish":
			if session == nil {
				sendError(conn, "no active upload")
				continue
			}

			finalPath, err := BusinessFinishUpload(session)
			if err != nil {
				sendError(conn, err.Error())
				CleanupSession(session)
				session = nil
				continue
			}

			sendJSON(conn, WSResponse{
				Type:    "upload_complete",
				Message: fmt.Sprintf("file %s uploaded successfully", session.FileName),
				Path:    finalPath,
			})

			log.Printf("file %s uploaded by %s to room %s (%d bytes)", finalPath, claims.Username, session.RoomName, session.FileSize)
			CleanupSession(session)
			session = nil

		case "upload_cancel":
			if session != nil {
				BusinessCancelUpload(session)
				CleanupSession(session)
				session = nil
				sendJSON(conn, WSResponse{Type: "upload_cancelled", Message: "upload cancelled"})
			}

		default:
			sendError(conn, "unknown message type: "+msg.Type)
		}
	}
}

// ============================================================
// HTTP хендлер — скачивание файла
// ============================================================

// HandleDownload — HTTP endpoint для скачивания файла по пути
// GET /download?path=/uploads/room/file.txt
func HandleDownload(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	uploadDir := config.GetEnv("UPLOAD_DIR", "./uploads")

	// Проверяем, что файл внутри uploadDir
	absUpload, _ := filepath.Abs(uploadDir)
	absFile, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFile, absUpload) {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filePath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}

// ============================================================
// HTTP хендлер — список файлов в комнате
// ============================================================

// HandleRoomFiles — HTTP endpoint для получения списка файлов комнаты
// POST /room_files с { "room_name": "..." }
func HandleRoomFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomName string `json:"room_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.RoomName == "" {
		http.Error(w, "room_name is required", http.StatusBadRequest)
		return
	}

	files, err := database.GetRoomFiles(req.RoomName)
	if err != nil {
		http.Error(w, "failed to get files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
	})
}

// ============================================================
// Вспомогательные функции
// ============================================================

func sendJSON(conn *websocket.Conn, resp WSResponse) {
	conn.WriteJSON(resp)
}

func sendError(conn *websocket.Conn, errMsg string) {
	sendJSON(conn, WSResponse{Type: "error", Error: errMsg})
}