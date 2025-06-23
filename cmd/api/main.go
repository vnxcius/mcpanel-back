package main

import (
	"database/sql"
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/http/events"
	"github.com/vnxcius/mcpanel-back/internal/http/router"
	"github.com/vnxcius/mcpanel-back/internal/logging"

	_ "github.com/lib/pq"
)

func init() {
	const logsDir = "./logs"
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logging.SetupLogger(logsDir + "/system.log")
	logging.SetupModlistChangelog(logsDir + "/modlist-changelog")
	slog.Debug("Initialized loggers")
}

func main() {
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
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

	events.InitializeManager()
	router.NewRouter(db)
}
