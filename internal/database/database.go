package database

import (
	"log"

	"gallery-be/config"
	"gallery-be/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Enable WAL mode & performance pragmas for SQLite
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			log.Printf("Warning: failed to execute pragma %s: %v", pragma, err)
		}
	}

	sqlDB.SetMaxOpenConns(1) // SQLite works best with 1 writer connection in WAL mode

	// Auto-migration
	if err := db.AutoMigrate(&model.Photo{}); err != nil {
		return nil, err
	}

	log.Println("SQLite database initialized successfully with WAL mode.")
	return db, nil
}
