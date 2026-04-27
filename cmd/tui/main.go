package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultBaseURL = "http://localhost:8080"
	maxMessages    = 20
)

type app struct {
	baseURL string
	token   string
	room    string
	user    string

	status      string
	allMessages []syncMessage
	lastSync    time.Time

	httpClient *http.Client
	ws         *websocket.Conn
	wsMu       sync.Mutex
	syncMu     sync.Mutex
	msgMu      sync.Mutex
	cfgPath    string

	done chan struct{}
}

type persistedConfig struct {
	BaseURL string `json:"base_url"`
	User    string `json:"user"`
	Token   string `json:"token"`
	Room    string `json:"room"`
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type apiResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type createRoomRequest struct {
	JWT        string `json:"jwt"`
	RoomName   string `json:"room_name"`
	RoomType   string `json:"room_type"`
	AccessType string `json:"access_type"`
}

type manageRoomRequest struct {
	JWT      string `json:"jwt"`
	RoomName string `json:"room_name"`
	Token    string `json:"token,omitempty"`
	Move     string `json:"move"`
}

type sendRequest struct {
	JWT      string `json:"jwt"`
	RoomName string `json:"room_name"`
	Text     string `json:"text"`
}

type wsEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type syncRequest struct {
	JWT      string    `json:"jwt"`
	LastTime time.Time `json:"last_time"`
}

type syncMessage struct {
	Text      string    `json:"text"`
	Username  string    `json:"username"`
	RoomName  string    `json:"room_name"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	baseURL := strings.TrimRight(os.Getenv("RIFT_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	a := &app{
		baseURL: baseURL,
		status:  "Введите /help для списка команд.",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		done: make(chan struct{}),
	}
	a.cfgPath = defaultConfigPath()
	a.loadConfig()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		a.shutdown()
		os.Exit(0)
	}()

	go a.syncLoop()
	a.render()
	a.readLoop()
}

func (a *app) readLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			a.shutdown()
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			a.render()
			continue
		}

		if err := a.handleCommand(line); err != nil {
			a.setStatus("Ошибка: " + err.Error())
		}
		a.render()
	}
}

func (a *app) handleCommand(line string) error {
	parts := strings.Fields(line)
	switch parts[0] {
	case "/help":
		a.setStatus(helpText())
		return nil
	case "/save":
		return a.saveConfig()
	case "/logout":
		return a.logout()
	case "/quit", "/exit":
		a.shutdown()
		os.Exit(0)
		return nil
	case "/reg":
		if len(parts) != 3 {
			return errors.New("использование: /reg <username> <password>")
		}
		return a.register(parts[1], parts[2])
	case "/login":
		if len(parts) != 3 {
			return errors.New("использование: /login <username> <password>")
		}
		return a.login(parts[1], parts[2])
	case "/create":
		if len(parts) < 2 || len(parts) > 4 {
			return errors.New("использование: /create <room> [public|private] [group|channel|1v1]")
		}
		access := "public"
		roomType := "group"
		if len(parts) >= 3 {
			access = parts[2]
		}
		if len(parts) == 4 {
			roomType = parts[3]
		}
		return a.createRoom(parts[1], access, roomType)
	case "/join":
		if len(parts) < 2 || len(parts) > 3 {
			return errors.New("использование: /join <room> [token]")
		}
		token := ""
		if len(parts) == 3 {
			token = parts[2]
		}
		return a.joinRoom(parts[1], token)
	case "/leave":
		room := a.room
		if len(parts) == 2 {
			room = parts[1]
		}
		if room == "" {
			return errors.New("комната не выбрана")
		}
		return a.leaveRoom(room)
	case "/use":
		if len(parts) != 2 {
			return errors.New("использование: /use <room>")
		}
		a.room = parts[1]
		if err := a.saveConfig(); err != nil {
			return err
		}
		a.setStatus("Активная комната: " + a.room)
		return nil
	case "/send":
		if a.room == "" {
			return errors.New("сначала выберите комнату через /use или /join")
		}
		text := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))
		text = strings.TrimSpace(text)
		if text == "" {
			return errors.New("использование: /send <text>")
		}
		return a.sendMessage(text)
	case "/sync":
		return a.syncOnce()
	default:
		if a.room == "" {
			return errors.New("неизвестная команда; используйте /help")
		}
		return a.sendMessage(line)
	}
}

func (a *app) register(username, password string) error {
	req := credentials{Username: username, Password: password}
	if err := a.postJSON("/auth/reg", req, nil); err != nil {
		return err
	}
	a.setStatus("Пользователь создан: " + username)
	return nil
}

func (a *app) login(username, password string) error {
	req := credentials{Username: username, Password: password}
	var resp tokenResponse
	if err := a.postJSON("/auth/auth", req, &resp); err != nil {
		return err
	}
	if resp.Token == "" {
		return errors.New("сервер не вернул JWT")
	}
	a.user = username
	a.token = resp.Token
	a.lastSync = time.Time{}
	a.msgMu.Lock()
	a.allMessages = nil
	a.msgMu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	if err := a.connectWS(); err != nil {
		return err
	}
	a.setStatus("Авторизация выполнена: " + username)
	return a.syncOnce()
}

func (a *app) createRoom(room, access, roomType string) error {
	if a.token == "" {
		return errors.New("сначала выполните /login")
	}
	req := createRoomRequest{
		JWT:        a.token,
		RoomName:   room,
		RoomType:   roomType,
		AccessType: access,
	}
	if err := a.postJSON("/chats/room_create", req, nil); err != nil {
		return err
	}
	a.room = room
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.setStatus("Комната создана: " + room)
	return nil
}

func (a *app) joinRoom(room, joinToken string) error {
	if a.token == "" {
		return errors.New("сначала выполните /login")
	}
	req := manageRoomRequest{
		JWT:      a.token,
		RoomName: room,
		Token:    joinToken,
		Move:     "sign",
	}
	if err := a.postJSON("/chats/room_sign", req, nil); err != nil {
		return err
	}
	a.room = room
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.setStatus("Вошли в комнату: " + room)
	return a.syncOnce()
}

func (a *app) leaveRoom(room string) error {
	if a.token == "" {
		return errors.New("сначала выполните /login")
	}
	req := manageRoomRequest{
		JWT:      a.token,
		RoomName: room,
		Move:     "leave",
	}
	if err := a.postJSON("/chats/manage", req, nil); err != nil {
		return err
	}
	if a.room == room {
		a.room = ""
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	a.setStatus("Вышли из комнаты: " + room)
	return nil
}

func (a *app) sendMessage(text string) error {
	if a.token == "" {
		return errors.New("сначала выполните /login")
	}
	req := sendRequest{
		JWT:      a.token,
		RoomName: a.room,
		Text:     text,
	}
	if err := a.postJSON("/chats/send", req, nil); err != nil {
		return err
	}
	a.setStatus("Сообщение отправлено в " + a.room)
	return a.syncOnce()
}

func (a *app) postJSON(path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var apiErr apiResponse
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			return errors.New(apiErr.Error)
		}
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (a *app) connectWS() error {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()

	if a.ws != nil {
		_ = a.ws.Close()
		a.ws = nil
	}

	wsURL, err := toWSURL(a.baseURL)
	if err != nil {
		return err
	}

	header := http.Header{}
	header.Set("JWT", a.token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/connect", header)
	if err != nil {
		return err
	}
	a.ws = conn
	return nil
}

func (a *app) syncLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.token != "" {
				if err := a.syncOnce(); err != nil {
					a.setStatus("Ошибка sync: " + err.Error())
					a.render()
				}
			}
		case <-a.done:
			return
		}
	}
}

func (a *app) syncOnce() error {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()

	a.wsMu.Lock()
	conn := a.ws
	a.wsMu.Unlock()

	if conn == nil || a.token == "" {
		return nil
	}

	req := wsEnvelope{
		Type: "sync",
		Payload: syncRequest{
			JWT:      a.token,
			LastTime: a.lastSync,
		},
	}
	if err := conn.WriteJSON(req); err != nil {
		if reconnectErr := a.connectWS(); reconnectErr != nil {
			return reconnectErr
		}
		a.wsMu.Lock()
		conn = a.ws
		a.wsMu.Unlock()
		if conn == nil {
			return errors.New("ws reconnect failed")
		}
		if err := conn.WriteJSON(req); err != nil {
			return err
		}
	}

	var msgs []syncMessage
	if err := conn.ReadJSON(&msgs); err != nil {
		return err
	}
	a.msgMu.Lock()
	updated := false
	for _, msg := range msgs {
		if !a.hasMessageLocked(msg) {
			a.allMessages = append(a.allMessages, msg)
			updated = true
		}
		if msg.CreatedAt.After(a.lastSync) {
			a.lastSync = msg.CreatedAt
		}
	}
	if len(a.allMessages) > maxMessages*8 {
		a.allMessages = a.allMessages[len(a.allMessages)-(maxMessages*8):]
	}
	a.msgMu.Unlock()
	if updated {
		a.setStatus(fmt.Sprintf("Получено %d новых сообщений", len(msgs)))
		a.render()
	}
	return nil
}

func (a *app) render() {
	a.msgMu.Lock()
	var messages []syncMessage
	for _, msg := range a.allMessages {
		if a.room == "" || msg.RoomName == a.room {
			messages = append(messages, msg)
		}
	}
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}
	a.msgMu.Unlock()

	fmt.Print("\033[2J\033[H")
	fmt.Println("Rift TUI")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf("Server : %s\n", a.baseURL)
	fmt.Printf("User   : %s\n", emptyDash(a.user))
	fmt.Printf("Room   : %s\n", emptyDash(a.room))
	fmt.Printf("Status : %s\n", a.status)
	fmt.Println(strings.Repeat("-", 72))
	fmt.Println("Сообщения:")
	if len(messages) == 0 {
		fmt.Println("  пока пусто")
	} else {
		for _, msg := range messages {
			ts := msg.CreatedAt.Format("15:04:05")
			fmt.Printf("  [%s] %s@%s: %s\n", ts, msg.Username, msg.RoomName, msg.Text)
		}
	}
	fmt.Println(strings.Repeat("-", 72))
	fmt.Println(helpText())
}

func (a *app) setStatus(status string) {
	a.status = status
}

func (a *app) shutdown() {
	select {
	case <-a.done:
	default:
		close(a.done)
	}
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.ws != nil {
		_ = a.ws.Close()
		a.ws = nil
	}
}

func helpText() string {
	return strings.Join([]string{
		"/reg <user> <pass>",
		"/login <user> <pass>",
		"/logout",
		"/create <room> [public|private] [group|channel|1v1]",
		"/join <room> [token]",
		"/leave [room]",
		"/use <room>",
		"/save",
		"/send <text>",
		"/sync",
		"/quit",
		"Любой текст без команды отправляется в активную комнату.",
	}, "\n")
}

func emptyDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func (a *app) hasMessageLocked(target syncMessage) bool {
	for _, msg := range a.allMessages {
		if msg.CreatedAt.Equal(target.CreatedAt) &&
			msg.Username == target.Username &&
			msg.RoomName == target.RoomName &&
			msg.Text == target.Text {
			return true
		}
	}
	return false
}

func toWSURL(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported base url scheme: %s", u.Scheme)
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func defaultConfigPath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil || cfgDir == "" {
		return ".rift-tui.json"
	}
	return filepath.Join(cfgDir, "rift", "tui.json")
}

func (a *app) loadConfig() {
	data, err := os.ReadFile(a.cfgPath)
	if err != nil {
		return
	}

	var cfg persistedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		a.setStatus("Конфиг поврежден: " + err.Error())
		return
	}

	if cfg.BaseURL != "" && os.Getenv("RIFT_BASE_URL") == "" {
		a.baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	a.user = cfg.User
	a.token = cfg.Token
	a.room = cfg.Room
	if a.token != "" {
		if err := a.connectWS(); err != nil {
			a.setStatus("Автоподключение из конфига не удалось: " + err.Error())
			return
		}
		a.setStatus("Загружен сохраненный пользователь: " + emptyDash(a.user))
	}
}

func (a *app) saveConfig() error {
	cfg := persistedConfig{
		BaseURL: a.baseURL,
		User:    a.user,
		Token:   a.token,
		Room:    a.room,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(a.cfgPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(a.cfgPath, data, 0o600)
}

func (a *app) logout() error {
	a.shutdown()
	a.done = make(chan struct{})
	a.user = ""
	a.token = ""
	a.room = ""
	a.lastSync = time.Time{}
	a.msgMu.Lock()
	a.allMessages = nil
	a.msgMu.Unlock()
	if err := a.saveConfig(); err != nil {
		return err
	}
	go a.syncLoop()
	a.setStatus("Сессия очищена")
	return nil
}
