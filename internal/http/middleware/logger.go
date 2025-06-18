package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func SlogLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // restore body
		}
		requestBody := string(bodyBytes)

		end := time.Now()
		latency := end.Sub(start).String()
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errors := c.Errors.ByType(gin.ErrorTypePrivate).String()

		attrs := []slog.Attr{
			slog.Int("status", statusCode),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", clientIP),
			slog.String("latency", latency),
			slog.String("body", requestBody),
		}

		if errors != "" {
			attrs = append(attrs, slog.String("errors", errors))
		}

		level := slog.LevelInfo
		if statusCode >= 500 {
			level = slog.LevelError
		} else if statusCode >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c, level, "REQUEST RECEIVED", attrs...)
	}
}
