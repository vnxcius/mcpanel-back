package routes

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/controllers"
	"github.com/vnxcius/sss-backend/middleware"
)

func RoutesHandler() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{
			"Content-Length",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	r.Use(middleware.RateLimitMiddleware())
	r.POST("/v1/verify-token", controllers.VerifyToken)

	protected := r.Group("/api/v1")
	protected.Use(middleware.TokenAuthMiddleware())

	protected.POST("/start", controllers.StartServer)

	if err := r.Run(":4000"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
