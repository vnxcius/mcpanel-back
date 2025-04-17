package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	memory "github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/vnxcius/sss-backend/internal/config"
)

func TokenAuthMiddleware() gin.HandlerFunc {
	cfg := config.GetConfig()

	validToken := cfg.Token
	if validToken == "" {
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Server configuration error: Missing validation token",
			})
		}
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

func RateLimitMiddleware() gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted("100-H")
	if err != nil {
		panic(err)
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	return func(c *gin.Context) {
		ctx, err := instance.Get(c, c.ClientIP())

		if err != nil {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "internal server error",
			})
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", ctx.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", ctx.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", ctx.Reset))

		if ctx.Reached {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Rate limit exceeded. Try again later.",
			})
			return
		}

		c.Next()
	}
}
