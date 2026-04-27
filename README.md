# Rift

> Realtime chat backend for rooms, membership, message delivery and WebSocket sync.

`Rift` is a Go backend project with JWT authentication, PostgreSQL storage and a WebSocket transport for realtime chat.

## Overview

The backend currently consists of:

- `auth-service` for registration, login and JWT validation
- `chat-service` for rooms, membership, messages and WebSocket sync
- `config` for environment loading
- PostgreSQL as the shared storage layer

## Stack

- Go `1.26`
- Gin
- Gorilla WebSocket
- pgx / pgxpool
- PostgreSQL 16
- JWT (`golang-jwt/jwt/v5`)
- Docker / Docker Compose

## Service Map

```mermaid
flowchart LR
    Client["Client App"] -->|HTTP| API["Gin API :8080"]
    Client -->|WebSocket| WS["/connect"]
    API --> AUTH["auth-service"]
    API --> CHAT["chat-service"]
    WS --> CHAT
    CHAT --> DB[(PostgreSQL)]
    AUTH --> DB
```

## Repository Layout

```text
backend/
├── main.go
├── go.mod
├── auth-service/
├── chat-service/
├── config/
└── docker/
```

## Request Flow

### HTTP flow

```mermaid
sequenceDiagram
    participant C as Client
    participant G as Gin Handler
    participant A as auth-service
    participant S as chat-service
    participant D as PostgreSQL

    C->>G: POST /chats/room_create
    G->>A: ParseJWT(jwt)
    A-->>G: username
    G->>S: create room
    S->>D: insert rooms + members
    D-->>S: ok
    S-->>G: success
    G-->>C: 200 JSON
```

### WebSocket flow

```mermaid
sequenceDiagram
    participant C as Client
    participant W as /connect
    participant R as Reader
    participant M as manager()
    participant D as PostgreSQL
    participant X as Writer

    C->>W: GET /connect + JWT header
    W-->>C: WebSocket upgrade
    C->>R: {"type":"send","payload":...}
    R->>M: decode message
    M->>D: validate and persist
    D-->>M: result
    M->>X: response payload
    X-->>C: JSON response
```

## API

### Auth endpoints

- `POST /auth/reg`
- `POST /auth/auth`

### Chat endpoints

- `POST /chats/room_create`
- `POST /chats/room_sign`
- `POST /chats/manage`
- `POST /chats/send`
- `GET /connect`

## HTTP Endpoints

### `POST /auth/reg`

Registers a user.

```json
{
  "username": "alice1",
  "password": "secret1"
}
```

### `POST /auth/auth`

Authenticates a user and returns JWT.

```json
{
  "username": "alice1",
  "password": "secret1"
}
```

Success response:

```json
{
  "token": "<JWT>"
}
```

### `POST /chats/room_create`

Creates a room and automatically adds the creator as `owner`.

```json
{
  "jwt": "user-jwt",
  "room_name": "team-room",
  "room_type": "group",
  "access_type": "public"
}
```

### `POST /chats/room_sign`

Joins a room.

```json
{
  "jwt": "user-jwt",
  "room_name": "team-room",
  "token": "123456",
  "move": "sign"
}
```

### `POST /chats/manage`

Leaves a room.

```json
{
  "jwt": "user-jwt",
  "room_name": "team-room",
  "move": "leave"
}
```

### `POST /chats/send`

Sends a message to a room if the sender is a member.

```json
{
  "jwt": "user-jwt",
  "room_name": "team-room",
  "text": "hello"
}
```

## WebSocket Protocol

### Handshake

```text
GET /connect
```

JWT must be passed in the `JWT` header.

### Message envelope

```json
{
  "type": "send",
  "payload": {}
}
```

### Supported message types

- `room_create`
- `send`
- `manage_room`
- `sync`

### Example: sync

```json
{
  "type": "sync",
  "payload": {
    "jwt": "user-jwt",
    "last_time": "2026-04-27T10:00:00Z"
  }
}
```

Example response:

```json
[
  {
    "text": "hello",
    "username": "alice",
    "room_name": "team-room",
    "created_at": "2026-04-27T10:10:00Z"
  }
]
```

## Domain Model

### Room types

- `channel`
- `1v1`
- `group`

### Access types

- `public`
- `private`

### Member roles

- `owner`
- `admin`
- `default`

### Member moves

- `sign`
- `leave`

## Database

The backend uses one shared PostgreSQL database for both `auth-service` and `chat-service`.

### Entity relationship diagram

```mermaid
erDiagram
    USERS {
        int id PK
        text username UK
        text password_hash
        timestamp created_at
    }

    ROOMS {
        int id PK
        text room_name UK
        text room_type
        text access_type
        timestamp created_at
    }

    MEMBERS {
        text room_name FK
        text username FK
        text status
        text role
        timestamp created_at
    }

    TOKENS {
        int id FK
        text status
        text role
        bigint token UK
        text room_name
        timestamp created_at
    }

    MESSAGES {
        text text
        text username FK
        text room_name FK
        timestamp created_at
    }

    USERS ||--o{ MEMBERS : joins
    USERS ||--o{ MESSAGES : sends
    ROOMS ||--o{ MEMBERS : contains
    ROOMS ||--o{ MESSAGES : stores
    ROOMS ||--o| TOKENS : protects
```

### Tables

#### `users`

Owned by `auth-service`.

| Column | Type | Notes |
|---|---|---|
| `id` | `SERIAL` | primary key |
| `username` | `TEXT` | unique |
| `password_hash` | `TEXT` | bcrypt hash |
| `created_at` | `TIMESTAMP` | default `NOW()` |

#### `rooms`

| Column | Type | Notes |
|---|---|---|
| `id` | `SERIAL` | primary key |
| `room_name` | `TEXT` | unique |
| `room_type` | `TEXT` | `channel`, `1v1`, `group` |
| `access_type` | `TEXT` | `public`, `private` |
| `created_at` | `TIMESTAMP` | default `NOW()` |

#### `members`

| Column | Type | Notes |
|---|---|---|
| `room_name` | `TEXT` | FK -> `rooms.room_name` |
| `username` | `TEXT` | FK -> `users.username` |
| `status` | `TEXT` | current member state |
| `role` | `TEXT` | `owner`, `admin`, `default` |
| `created_at` | `TIMESTAMP` | default `NOW()` |

Constraint:

- unique pair: `room_name`, `username`

#### `tokens`

| Column | Type | Notes |
|---|---|---|
| `id` | `INT` | FK -> `rooms.id` |
| `status` | `TEXT` | token state |
| `role` | `TEXT` | role attached to token |
| `token` | `BIGINT` | unique |
| `room_name` | `TEXT` | room reference |
| `created_at` | `TIMESTAMP` | default `NOW()` |

#### `messages`

| Column | Type | Notes |
|---|---|---|
| `text` | `TEXT` | message body |
| `username` | `TEXT` | FK -> `users.username` |
| `room_name` | `TEXT` | FK -> `rooms.room_name` |
| `created_at` | `TIMESTAMP` | default `NOW()` |

## Startup Order

`main.go` initializes the backend in this order:

1. `auth.InitAuthDB()`
2. `chats.InitChatDB()`
3. `r.Run(":8080")`

This is important because chat tables reference `users(username)`.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:password@localhost:5432/auth_db?sslmode=disable` | PostgreSQL connection |
| `JWT_SECRET` | `dasdasdwefafdsaefafdsaf` | JWT signing and validation |
| `CHAT_WORKERS_SOCKETS` | `200` | max concurrent websocket workers |

## Run

### With Go

From `backend/`:

```bash
go run ./main.go
```

### With Docker Compose

From `backend/docker/`:

```bash
docker compose up --build
```

Default ports:

- app: `8080`
- postgres: `5432`

## Example Usage

### Register

```bash
curl -X POST http://localhost:8080/auth/reg \
  -H "Content-Type: application/json" \
  -d '{"username":"alice1","password":"secret1"}'
```

### Authenticate

```bash
curl -X POST http://localhost:8080/auth/auth \
  -H "Content-Type: application/json" \
  -d '{"username":"alice1","password":"secret1"}'
```

### Create room

```bash
curl -X POST http://localhost:8080/chats/room_create \
  -H "Content-Type: application/json" \
  -d '{"jwt":"<TOKEN>","room_name":"team-room","room_type":"group","access_type":"public"}'
```

### Send message

```bash
curl -X POST http://localhost:8080/chats/send \
  -H "Content-Type: application/json" \
  -d '{"jwt":"<TOKEN>","room_name":"team-room","text":"hello"}'
```

## Current Limitations

- WebSocket `CheckOrigin` currently allows every origin
- JWT transport is inconsistent between HTTP body and WebSocket header
- there is no dedicated REST endpoint for history reads
- private room token generation is not implemented yet
- WebSocket success responses are still low-level

## Key Files

- [main.go](/home/void/project/magicv2/backend/main.go)
- [auth-service/models.go](/home/void/project/magicv2/backend/auth-service/models.go)
- [auth-service/repository.go](/home/void/project/magicv2/backend/auth-service/repository.go)
- [chat-service/handlers.go](/home/void/project/magicv2/backend/chat-service/handlers.go)
- [chat-service/business.go](/home/void/project/magicv2/backend/chat-service/business.go)
- [chat-service/repository.go](/home/void/project/magicv2/backend/chat-service/repository.go)
- [chat-service/models.go](/home/void/project/magicv2/backend/chat-service/models.go)
