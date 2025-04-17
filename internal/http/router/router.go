package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/http/handlers"
	"github.com/vnxcius/sss-backend/internal/http/middleware"
)

func NewRouter() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://sss.vncius.dev",
			"https://sss-api.vncius.dev",
		},
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
	{
		r.GET("/ping", handlers.Ping)
		r.POST("/v1/verify-token", handlers.VerifyToken)
		r.GET("/v1/sse", handlers.StatusStream)
	}

	protected := r.Group("/api/v1")
	protected.Use(middleware.TokenAuthMiddleware())
	{
		protected.POST("/start", handlers.StartServer)
		protected.POST("/stop", handlers.StopServer)
		protected.POST("/restart", handlers.RestartServer)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Not Found: " + c.Request.URL.Path})
	})
}
