package router

import (
	"database/sql"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/http/handlers"
	"github.com/vnxcius/sss-backend/internal/http/middleware"
	"github.com/vnxcius/sss-backend/internal/http/templates"
)

func NewRouter(db *sql.DB) {
	r := gin.New()
	r.Use(middleware.SlogLoggerMiddleware())
	r.Use(gin.Recovery())

	// since we're using Cloudflare Tunnel to reverse proxy the API
	// we should trust only localhost
	err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	if err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := strings.Split(allowedOriginsEnv, ",")

	slog.Info("Allowing origins", "origins", allowedOrigins)
	r.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
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

	tmpl := template.Must(template.ParseFS(templates.TemplatesFS, "*.html"))
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
		r.GET("/v2/server-status-stream", handlers.StatusStream)
	}

	r.Use(middleware.WithDB(db))
	protected := r.Group("/api/v2")
	protected.Use(middleware.TokenAuth())
	{
		protected.POST("/start", handlers.StartServer)
		protected.POST("/stop", handlers.StopServer)
		protected.POST("/restart", handlers.RestartServer)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Not Found: " + c.Request.URL.Path})
	})

	slog.Info("Starting server on port " + os.Getenv("PORT"))
	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
