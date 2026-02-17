# data-channel-service

Обмен данными в реальном времени (Go): чат, аннотации, файлы. WebSocket `/ws/data/:session_id/:user_id`, история и загрузка файлов.

## Стек

- Go 1.21+, **Gin**, **gorilla/websocket**, **GORM**, PostgreSQL, **golang-migrate**
- Конфиг только `.env`

## API

- `GET /health`, `GET /ready`
- `GET /ws/data/:session_id/:user_id` — WebSocket (ретрансляция в сессию + запись в БД)
- `GET /data/:session_id/history` — история (query `limit`, по умолчанию 100)
- `POST /data/file` — multipart: `session_id`, `user_id`, `file`

## Запуск

```bash
cp .env.example .env
go run ./cmd/data-channel-service api
```

Порт **8093**. PostgreSQL должен быть запущен; миграции применяются при старте. Docker: `cd deployments && docker compose up -d`.
