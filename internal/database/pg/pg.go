package pg

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func NewConnection() (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if os.Getenv("ENVIRONMENT") == "development" {
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

	db, err := gorm.Open(postgres.Open(os.Getenv("POSTGRES_DSN")), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Printf("Failed to connect to database! HOST: %s", os.Getenv("DB_HOST"))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get underlying sql.DB: %v", err)
		return nil, err
	}

	DBMaxIdleConns, _ := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS"))
	DBMaxOpenConns, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	DBConnMaxLifetime, _ := time.ParseDuration(os.Getenv("DB_CONN_MAX_LIFETIME"))
	sqlDB.SetMaxIdleConns(DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(DBConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		slog.Warn("Failed to ping database", "error", err)
		_ = sqlDB.Close()
		return nil, err
	}

	slog.Info("Database connection established successfully")

	DB = db
	return db, nil
}
