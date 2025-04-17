package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/config"
	"github.com/vnxcius/sss-backend/internal/database/pg"
	"github.com/vnxcius/sss-backend/internal/http/events"
	"github.com/vnxcius/sss-backend/internal/http/router"
)

func main() {
	// load environment variables
	cfg := config.GetConfig()
	log.Println("Loaded environment", cfg.Environment)

	switch cfg.Environment {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "production":
		gin.SetMode(gin.ReleaseMode)
	}
	log.Printf("Gin mode set to: %s", gin.Mode())

	// initialize database connection
	db, err := pg.NewConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	events.NewStatusManager()
	router.NewRouter()
}
