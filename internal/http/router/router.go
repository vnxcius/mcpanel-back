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
	"github.com/vnxcius/mcpanel-back/internal/http/handlers"
	"github.com/vnxcius/mcpanel-back/internal/http/middleware"
	"github.com/vnxcius/mcpanel-back/internal/http/templates"
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
		AllowMethods: []string{"GET", "POST", "OPTIONS", "DELETE"},
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

	{
		v2 := r.Group("/api/v2").Use(middleware.RateLimit())
		v2.GET("/ping", handlers.Ping)
		v2.GET("/logs/latest", handlers.GetLatestLogs)

		v2.GET("/bot/terms-of-service", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "tos", nil)
		})
		v2.GET("/bot/privacy-policy", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "privacy-policy", nil)
		})
		v2.GET("/ws", handlers.ServeWebSocket)
		// v2.GET("/server-status", handlers.Status)
	}

	{
		protected := r.Group("/api/v2/signed")
		protected.Use(middleware.WithDB(db))
		protected.Use(middleware.RateLimit())
		protected.Use(middleware.TokenAuth())

		protected.POST("/server/start", handlers.StartServer)
		protected.POST("/server/stop", handlers.StopServer)
		protected.POST("/server/restart", handlers.RestartServer)

		protected.GET("/modlist", handlers.UpdateModlist)
		protected.POST("/modlist/upload", handlers.UploadMods)
		protected.DELETE("/modlist/delete/:name", handlers.DeleteMod)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Not Found: " + c.Request.URL.Path})
	})

	slog.Info("Starting server on port " + os.Getenv("PORT"))
	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
