package middleware

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func WithDB(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}
