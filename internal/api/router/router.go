package router

import (
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vnxcius/mcpanel-back/internal/api/handlers"
	"github.com/vnxcius/mcpanel-back/internal/api/middleware"
	"github.com/vnxcius/mcpanel-back/web"
)

func NewRouter() {
	r := gin.New()
	r.Use(middleware.SloggerMiddleware())
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 512 << 20

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

	tmpl := template.Must(template.ParseFS(web.TemplatesFS, "templates/*.html"))
	r.SetHTMLTemplate(tmpl)

	staticFS, _ := fs.Sub(web.TemplatesFS, "static")
	r.StaticFS("/static", http.FS(staticFS))

	{
		v2 := r.Group("/api/v2").Use(middleware.RateLimit())
		v2.GET("/ping", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		v2.GET("/bot/terms-of-service", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "tos", nil)
		})
		v2.GET("/bot/privacy-policy", func(ctx *gin.Context) {
			ctx.HTML(http.StatusOK, "privacy-policy", nil)
		})
		v2.GET("/ws", handlers.ServeWebSocket)
		v2.GET("/server-status", handlers.GetServerStatus)
		v2.GET("/modlist", handlers.GetModlist)
	}

	{
		protected := r.Group("/api/v2/signed")
		protected.Use(middleware.RateLimit())
		protected.Use(middleware.TokenAuth())

		protected.POST("/server/start", handlers.StartServer)
		protected.POST("/server/stop", handlers.StopServer)
		protected.POST("/server/restart", handlers.RestartServer)

		protected.POST("/mod/upload", handlers.UploadMods)
		protected.GET("/mod/download/:name", handlers.DownloadMod)
		protected.POST("/mod/update/:name", handlers.UpdateMod)
		protected.DELETE("/mod/delete/:name", handlers.DeleteMod)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "This Page Was Not Found: " + c.Request.URL.Path})
	})

	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	slog.Info("Starting server in " + gin.Mode())
	slog.Info("Starting server on port " + os.Getenv("PORT"))
	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
