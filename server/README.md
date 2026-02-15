# server

## Stack
- Go + Gin + GORM
- sqlite (local) / pgsql (production)
- local in-memory cache (local) / redis (production)

## Run
```bash
go mod tidy
go run ./cmd/server
```

## Core APIs
- `GET /healthz` health check
- `GET /api/v1/members` list members
- `POST /api/v1/members` create member
- `GET /api/v1/orders` list orders
- `POST /api/v1/orders` create order
- `GET /api/v1/summary` merchant KPI summary

All `/api/v1/*` endpoints return:
```json
{
  "code": 200,
  "msg": "ok",
  "data": {}
}
```

## Smoke Test
```bash
go test ./...
```
