package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/controllers"
	"github.com/vnxcius/sss-backend/middleware"
)

func RoutesHandler() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           30 * 24 * 60 * 60, // 30 days
	}))

	r.POST("/v1/verify-token", controllers.VerifyToken)

	protected := r.Group("/api/v1")
	protected.Use(middleware.TokenAuthMiddleware())

	protected.POST("/start", controllers.StartServer)

	r.Run(":4000")
}
