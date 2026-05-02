# MagicV2 — Архитектура проекта

## Общая структура

```
magicv2/
├── backend/          # Go-бэкенд (Rift)
│   ├── main.go       # Точка входа, роутинг Gin
│   ├── auth-service/ # Регистрация, логин, JWT
│   ├── chat-service/ # Комнаты, сообщения, WebSocket
│   ├── chat-moderation/ # Модерация (бан, мут, кик, удаление)
│   ├── database/     # PostgreSQL + Redis подключение и запросы
│   ├── config/       # Загрузка .env
│   ├── cmd/tui/      # Терминальный клиент
│   └── docker/       # Docker Compose + Dockerfile
├── frontend/         # Веб-клиент (HTML/CSS/JS)
└── meta/             # Документация для разработчиков
```

## Бэкенд: архитектура слоёв

Каждый сервис (auth-service, chat-service, chat-moderation) следует единому паттерну:

```
handlers.go  →  business.go  →  database/ (напрямую)
```

- **handlers.go** — HTTP-обработчики (Gin) или WebSocket manager
- **business.go** — бизнес-логика (JWT, проверка прав, вызов database)
- **models.go** — типы данных, константы

**Важно:** Репозиторий не используется — бизнес-слой вызывает `database` напрямую.

## База данных

PostgreSQL, таблицы создаются в `database/connection.go` → `InitAllTables()`:

- `users` — пользователи (username, password_hash)
- `rooms` — комнаты (room_name, room_type, access_type)
- `members` — участники (username, role, status, room_name)
- `tokens` — токены для приватных комнат
- `messages` — сообщения (text, username, room_name, created_at)

## Эндпоинты API

### HTTP

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/auth/reg` | Регистрация |
| POST | `/auth/auth` | Авторизация (возвращает JWT) |
| POST | `/chats/room_create` | Создать комнату |
| POST | `/chats/room_sign` | Войти в комнату |
| POST | `/chats/manage` | Выйти из комнаты |
| POST | `/chats/send` | Отправить сообщение |
| GET  | `/connect` | WebSocket upgrade |

### WebSocket (ws://host/connect)

Формат сообщения:
```json
{
  "type": "send|room_create|manage_room|sync|moderation",
  "payload": { ... }
}
```

Типы:
- `send` — отправка сообщения
- `room_create` — создание комнаты
- `manage_room` — вход/выход (sign/leave)
- `sync` — синхронизация сообщений
- `moderation` — действия модерации (ban, mute, kick, deleteroom, deletemsg)

## Модерация

Пакет `chat-moderation`:
- `HandleModeration(req)` — основная функция
- Проверяет JWT, права (owner/admin), выполняет действие
- Действия: ban, mute, kick, deleteroom, deletemsg

## Запуск

```bash
# Docker
cd backend/docker && docker compose up --build

# Локально
cd backend && go run .
```

## Переменные окружения (.env)

```
DATABASE_URL=postgres://postgres:password@localhost:5432/auth_db?sslmode=disable
JWT_SECRET=dasdasdwefafdsaefafdsaf
REDIS_PASSWORD=
REDIS_ADRESS=localhost:6379
LIMIT=400
PORT=8080