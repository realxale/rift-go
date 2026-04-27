## Rift TUI

Простой терминальный клиент для локального backend.

Запуск:

```bash
go run ./cmd/tui
```

Если API работает не на `http://localhost:8080`, задайте:

```bash
RIFT_BASE_URL=http://localhost:8080 go run ./cmd/tui
```
