package router

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/config"
	"github.com/vnxcius/sss-backend/internal/http/handlers"
	"github.com/vnxcius/sss-backend/internal/http/middleware"
	"github.com/vnxcius/sss-backend/internal/http/templates"
)

func NewRouter() {
	r := gin.Default()

	// since we're using Cloudflare Tunnel to reverse proxy the API
	// we should trust only localhost
	err := r.SetTrustedProxies([]string{"127.0.0.1", "::1", "sss-api.vncius.dev"})
	if err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

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

	tmpl := template.Must(template.ParseFS(templates.TemplatesFS, "*.html")) // Adjust glob pattern if needed
	r.SetHTMLTemplate(tmpl)

	r.Use(middleware.RateLimit())
	{
		r.GET("/ping", handlers.Ping)
		r.GET("/bot/terms-of-service", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "tos", nil)
		})
		r.GET("/bot/privacy-policy", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "privacy-policy", nil)
		})
		r.POST("/v1/verify-token", handlers.VerifyToken)
		r.GET("/v1/sse", handlers.StatusStream)
	}

	protected := r.Group("/api/v1")
	protected.Use(middleware.TokenAuth())
	{
		protected.POST("/start", handlers.StartServer)
		protected.POST("/stop", handlers.StopServer)
		protected.POST("/restart", handlers.RestartServer)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Not Found: " + c.Request.URL.Path})
	})

	p := config.GetConfig().Port
	log.Println("Server is running on port", p)
	r.Run(p)
	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
