package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/config"
)

func TokenAuth() gin.HandlerFunc {
	cfg := config.GetConfig()

	validToken := cfg.Token
	if validToken == "" {
		log.Fatal("Server configuration error: Missing validation token (TOKEN env variable)")
	}

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
