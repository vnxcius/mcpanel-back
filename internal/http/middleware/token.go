package middleware

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Session struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func TokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the DB from context
		dbAny, exists := c.Get("db")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Database connection not found",
			})
			return
		}

		db := dbAny.(*sql.DB)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token não encontrado",
			})
			return
		}

		const bearerTokenPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerTokenPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization scheme, 'Bearer' prefix required",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, bearerTokenPrefix)

		slog.Info("Validating session token")
		if token == os.Getenv("DISCORD_BOT_TOKEN") {
			slog.Info("Discord bot request received, skipping session token validation")
			c.Next()
			return
		}

		err := db.QueryRow(`SELECT id FROM public."Session" WHERE id = $1`, token).Scan()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid session token",
			})
			return
		}
		slog.Info("Found session token in database")

		c.Next()
	}
}
