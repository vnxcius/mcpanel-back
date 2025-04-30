package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/sss-backend/internal/config"
	"github.com/vnxcius/sss-backend/internal/database/model"
	"github.com/vnxcius/sss-backend/internal/database/pg"
	"github.com/vnxcius/sss-backend/internal/http/events"
	"github.com/vnxcius/sss-backend/internal/http/handlers"
	"github.com/vnxcius/sss-backend/internal/http/router"
	"github.com/vnxcius/sss-backend/internal/util"
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

	// initialize database connection
	db, err := pg.NewConnection()
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	db.AutoMigrate(
		&model.User{},
		&model.Session{},
	)

	var count int64
	db.Model(&model.User{}).Count(&count)

	if count == 0 {
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			log.Fatal("ADMIN_PASSWORD is not set")
		}
		hashedPassword := util.HashPassword(password)
		user := &model.User{
			Password: hashedPassword,
		}

		if err := db.Create(&user).Error; err != nil {
			log.Fatal("Failed to create admin user: ", err)
		}

		slog.Info("Admin user created")
	}

	events.InitializeStatusManager()
	hdl := handlers.NewHandlers(os.Getenv("SECRET_KEY"))
	router.NewRouter(hdl)
}
