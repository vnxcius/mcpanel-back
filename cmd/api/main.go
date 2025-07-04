package main

import (
	"database/sql"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/api/router"
	"github.com/vnxcius/mcpanel-back/internal/api/ws"
	"github.com/vnxcius/mcpanel-back/internal/db"
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
	var err error
	db.DBConn, err = sql.Open("postgres", os.Getenv("POSTGRES_DSN"))
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
	}
	defer db.DBConn.Close()

	if err := db.DBConn.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
	}
	slog.Info("Connected to database")

	ws.InitializeManager()
	router.NewRouter()
}
