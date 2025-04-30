package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func TokenAuth() gin.HandlerFunc {
	validToken := os.Getenv("TOKEN")

	const bearerTokenPrefix = "Bearer "

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token não encontrado",
			})
			return
		}

		if !strings.HasPrefix(authHeader, bearerTokenPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization scheme, 'Bearer' prefix required",
			})
			return
		}
		token := strings.TrimPrefix(authHeader, bearerTokenPrefix)
		if token != validToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token inválido ou não encontrado",
			})
			return
		}

		c.Next()
	}
}
