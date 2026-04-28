# Rift Backend

Go-бэкенд для realtime-чата с JWT-аутентификацией, PostgreSQL и WebSocket-подключением.

## Что есть в проекте

- `auth-service` — регистрация, логин, проверка JWT
- `chat-service` — комнаты, участники, сообщения, WebSocket
- `cmd/tui` — простой терминальный клиент
- `docker/` — Dockerfile и `docker-compose.yml` для локального запуска

## Стек

- Go `1.26`
- Gin
- Gorilla WebSocket
- PostgreSQL
- pgx / pgxpool
- JWT

## Быстрый запуск

### Через Docker Compose

```bash
docker compose -f docker/docker-compose.yml up --build
```

API будет доступен на `http://localhost:8080`.

### Локально

Нужны PostgreSQL и переменные окружения:

- `DATABASE_URL`
- `JWT_SECRET`
- `PORT` — по умолчанию `8080`
- `LIMIT` — лимит WebSocket-подключений, если используется

Запуск:

```bash
go run .
```

## Основные эндпоинты

- `POST /auth/reg`
- `POST /auth/auth`
- `POST /chats/room_create`
- `POST /chats/room_sign`
- `POST /chats/manage`
- `POST /chats/send`
- `GET /connect`

## TUI-клиент

```bash
go run ./cmd/tui
```

По умолчанию клиент использует `http://localhost:8080`. Для другого адреса можно задать `RIFT_BASE_URL`.

## Документация

Подробная старая документация перенесена в [docs/README.legacy.md](/home/void/project/magicv2/backend/docs/README.legacy.md).
