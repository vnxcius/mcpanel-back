package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func TokenAuthMiddleware() gin.HandlerFunc {
	validToken := os.Getenv("TOKEN")
	return func(c *gin.Context) {
		token, err := c.Cookie("sss-token")
		if err != nil || token != validToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "invalid or missing token",
			})
			return
		}

		// If valid, continue to the next handler
		c.Next()
	}
}
