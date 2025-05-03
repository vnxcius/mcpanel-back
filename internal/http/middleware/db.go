// middleware/db.go
package middleware

import (
	"database/sql"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func WithDB(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		slog.Info("Set DB on context")
		c.Next()
	}
}
