package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"small-merchant-ops-hub-server/internal/config"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	var (
		database *gorm.DB
		err      error
	)

	if cfg.IsLocal() {
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}
		database, err = gorm.Open(sqlite.Open(cfg.SQLitePath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	} else {
		database, err = gorm.Open(postgres.Open(cfg.PGDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.AutoMigrate(&KeyValue{}, &Member{}, &Order{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return database, nil
}
