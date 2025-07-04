package middleware

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/mcpanel-back/internal/db"
)

type Session struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func TokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Debug("No authorization header found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token not found",
			})
			return
		}

		const bearerTokenPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerTokenPrefix) {
			slog.Debug("Invalid authorization scheme")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization scheme, 'Bearer' prefix required",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, bearerTokenPrefix)

		if token == os.Getenv("DISCORD_BOT_TOKEN") {
			slog.Info("Discord bot request received, skipping session token validation")
			c.Next()
			return
		}

		var retrievedID string
		err := db.DBConn.QueryRow(`SELECT id FROM "Session" WHERE id = $1`, token).Scan(&retrievedID)

		if err != nil {
			if err == sql.ErrNoRows {
				slog.Info("Session token not found", "token_attempted", token)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"message": "Invalid or expired token.",
				})
				return
			}

			slog.Error("Database error during token validation query", "error", err, "token_attempted", token)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Internal server error during token validation.",
			})
			return
		}

		slog.Info("Session token successfully validated")
		c.Next()
	}
}
