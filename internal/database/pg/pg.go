package pg

import (
	"log"
	"os"
	"time"

	"github.com/vnxcius/sss-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewConnection(cfg *config.Config) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if cfg.DBLogMode {
		logLevel = logger.Info
	} else {
		logLevel = logger.Silent
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logLevel,
			Colorful:      true,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Printf("Failed to connect to database! HOST: %s", cfg.DBHost)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get underlying sql.DB: %v", err)
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Failed to ping database: %v", err)
		_ = sqlDB.Close() // Attempt to close the connection
		return nil, err
	}

	log.Println("Database connection established successfully")
	return db, nil
}
