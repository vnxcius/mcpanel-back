package main

import (
	"database/sql"
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/sss-backend/internal/config"
	"github.com/vnxcius/sss-backend/internal/http/events"
	"github.com/vnxcius/sss-backend/internal/http/router"

	_ "github.com/lib/pq"
)

const logFilePath = "./logs/system.log"

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	config.SetupLogger(logFilePath)
	slog.Info("Initialized logger")

	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	slog.Info("Gin mode set to: " + gin.Mode())

	db, err := sql.Open("postgres", os.Getenv("POSTGRES_DSN"))
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}
	slog.Info("Connected to database")

	events.InitializeStatusManager()
	router.NewRouter(db)
}
