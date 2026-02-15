package config

import (
	"errors"
	"os"
)

type Config struct {
	Env             string
	Port            string
	SQLitePath      string
	PGDSN           string
	RedisURL        string
	CacheMode       string
	CORSAllowOrigin string
}

func LoadFromEnv() Config {
	env := getenv("APP_ENV", "local")
	corsDefault := "*"
	if env != "local" {
		corsDefault = ""
	}
	cfg := Config{
		Env:             env,
		Port:            getenv("PORT", "8080"),
		SQLitePath:      getenv("SQLITE_PATH", "./data/app.db"),
		PGDSN:           getenv("PG_DSN", ""),
		RedisURL:        getenv("REDIS_URL", "redis://127.0.0.1:6379/0"),
		CacheMode:       getenv("CACHE_MODE", ""),
		CORSAllowOrigin: getenv("CORS_ALLOW_ORIGIN", corsDefault),
	}

	if cfg.CacheMode == "" {
		if cfg.IsLocal() {
			cfg.CacheMode = "local"
		} else {
			cfg.CacheMode = "redis"
		}
	}

	return cfg
}

func (c Config) IsLocal() bool {
	return c.Env == "local"
}

func (c Config) DatabaseDriver() string {
	if c.IsLocal() {
		return "sqlite"
	}
	return "pgsql"
}

func (c Config) Validate() error {
	if !c.IsLocal() && c.PGDSN == "" {
		return errors.New("PG_DSN is required when APP_ENV is not local")
	}
	if !c.IsLocal() && c.CORSAllowOrigin == "" {
		return errors.New("CORS_ALLOW_ORIGIN is required when APP_ENV is not local")
	}
	if !c.IsLocal() && c.CORSAllowOrigin == "*" {
		return errors.New("CORS_ALLOW_ORIGIN cannot be * when APP_ENV is not local")
	}
	if c.CacheMode != "local" && c.CacheMode != "redis" {
		return errors.New("CACHE_MODE must be local or redis")
	}
	if c.CacheMode == "redis" && c.RedisURL == "" {
		return errors.New("REDIS_URL is required when CACHE_MODE=redis")
	}
	return nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
