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
- `POST /api/auth/login` admin login (`Super/Admin/User`, password `123456`; `User` is read-only operations role)
- `GET /api/user/info` current user profile + roles/buttons (requires `Authorization` token)
- `GET /api/user/list` system user list (requires `Authorization` token)
- `GET /api/role/list` system role list (requires `Authorization` token)
- `GET /api/v3/system/menus` backend-mode menu list (requires `Authorization` token)
- Auth token session is in-memory with default 24h TTL
- `GET /api/v1/members` list members
- `POST /api/v1/members` create member
- `GET /api/v1/orders` list orders
- `POST /api/v1/orders` create order
- `GET /api/v1/campaigns` list campaigns
- `POST /api/v1/campaigns` create campaign
- `GET /api/v1/followups` list repurchase follow-up members
- `GET /api/v1/reports/campaign-attribution` campaign attribution report
- `GET /api/v1/reports/campaign-attribution/export` export attribution CSV
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
