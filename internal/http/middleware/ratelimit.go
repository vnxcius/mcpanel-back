// ported from https://github.com/bernardinorafael/go-boilerplate/tree/main/internal/infra/http/middleware/ratelimit.go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	// rateLimit is the number of requests per second
	rateLimit = 1
	// burst is the maximum number of requests that can be made in a single burst
	burst = 4
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit() gin.HandlerFunc {
	var mu sync.Mutex
	var clients = make(map[string]*client)

	// Background routine to remove expired clients
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > time.Minute*3 {
					delete(clients, ip)
				}
			}
			// IMPORTANT: Unlock the mutext when the cleanup is done
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		mu.Lock()

		ip := c.ClientIP()
		if _, ok := clients[ip]; !ok {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rateLimit, burst),
			}
		}
		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Rate limit exceeded",
			})
			return
		}

		mu.Unlock()
		c.Next()
	}
}
