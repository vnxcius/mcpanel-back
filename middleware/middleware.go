package middleware

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	memory "github.com/ulule/limiter/v3/drivers/store/memory"
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
