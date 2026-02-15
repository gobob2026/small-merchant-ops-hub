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
