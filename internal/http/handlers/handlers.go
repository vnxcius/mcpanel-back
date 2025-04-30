package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/database/controllers"
	"github.com/vnxcius/sss-backend/internal/database/model"
	"github.com/vnxcius/sss-backend/internal/http/events"
	"github.com/vnxcius/sss-backend/internal/token"
	"github.com/vnxcius/sss-backend/internal/util"
)

type Handlers struct {
	c          *gin.Context
	TokenMaker *token.JWTMaker
}

func NewHandlers(secretKey string) *Handlers {
	return &Handlers{
		TokenMaker: token.NewJWTMaker(secretKey),
	}
}

func (h *Handlers) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func (h *Handlers) StatusStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	clientChan := make(chan events.ServerStatus, 1)
	events.ServerStatusManager.AddClient(clientChan)
	defer events.ServerStatusManager.RemoveClient(clientChan)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Streaming unsupported"})
		return
	}

	slog.Info("SSE client connected", "ip", c.ClientIP())
	for {
		select {
		case <-ctx.Done():
			slog.Info("SSE client disconnected", "ip", c.ClientIP())
			return
		case statusUpdate := <-clientChan:
			_, err := fmt.Fprintf(c.Writer, "data: {\"status\": \"%s\"}\n\n", statusUpdate)
			if err != nil {
				slog.Error("Error writing to SSE client", "error", err)
				return
			}
			flusher.Flush()
		case <-ticker.C:
			_, err := fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			if err != nil {
				slog.Error("Error writing heartbeat to SSE client", "error", err, "ip", c.ClientIP())
				return
			}
			slog.Info("Heartbeat sent", "ip", c.ClientIP())
		}
		flusher.Flush()
	}
}

func (h *Handlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid request",
			"error":   true,
		})
		return
	}

	user, _ := controllers.GetUser()
	err := util.CheckPasswordHash(req.Password, user.Password)

	if err != nil {
		slog.Info("Login failed", "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Senha incorreta",
			"error":   true,
		})
		return
	}

	accessToken, accessClaims, err := h.TokenMaker.CreateToken(user.ID, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create access token",
			"error":   true,
		})
		return
	}
	refreshToken, refreshClaims, err := h.TokenMaker.CreateToken(user.ID, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create access token",
			"error":   true,
		})
		return
	}

	err = controllers.CreateSession(&model.Session{
		ID:           refreshClaims.RegisteredClaims.ID,
		UserID:       user.ID,
		IsRevoked:    false,
		RefreshToken: refreshToken,
		ExpiresAt:    refreshClaims.ExpiresAt.Time,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create session",
			"error":   true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Login successful",
		"error":                 false,
		"token":                 accessToken,
		"refresh_token":         refreshToken,
		"RefreshTokenExpiresAt": refreshClaims.ExpiresAt.Time,
		"AccessTokenExpiresAt":  accessClaims.ExpiresAt.Time,
		"user": gin.H{
			"id": user.ID,
		},
	})
}

func (h *Handlers) Logout(c *gin.Context) {
	id := c.Param("id")
	controllers.DeleteSession(id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
		"error":   false,
	})
}

func (h *Handlers) RenewAccessToken(c *gin.Context) {
	var req RenewAccessTokenRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid request",
			"error":   true,
		})
		return
	}

	refreshClaims, err := h.TokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid refresh token",
			"error":   true,
		})
		return
	}

	session, err := controllers.GetSession(refreshClaims.RegisteredClaims.ID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "No session found",
			"error":   true,
		})
		return
	}

	if session.IsRevoked {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Session is revoked",
			"error":   true,
		})
		return
	}

	if session.UserID != refreshClaims.ID {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid session",
			"error":   true,
		})
		return
	}

	accessToken, accessClaims, err := h.TokenMaker.CreateToken(refreshClaims.ID, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create access token",
			"error":   true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Renew access token successful",
		"error":                 false,
		"token":                 accessToken,
		"AccessTokenExpiresAt":  accessClaims.ExpiresAt.Time,
		"RefreshTokenExpiresAt": refreshClaims.ExpiresAt.Time,
	})
}

func (h *Handlers) RevokeSession(c *gin.Context) {
	id := c.Param("id")
	err := controllers.RevokeSession(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to revoke session",
			"error":   true,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Session revoked",
		"error":   false,
	})
}

func (h *Handlers) StartServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus == events.Online || currentStatus == events.Starting {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already online or starting"})
		return
	}

	// Trigger the simulation (which updates status via SetStatus)
	events.ServerStatusManager.StartServer()

	c.JSON(http.StatusOK, gin.H{"message": "Server starting..."})
}

func (h *Handlers) StopServer(c *gin.Context) {
	// Add checks: Is it already offline or stopping?
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus == events.Offline || currentStatus == events.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already offline or stopping"})
		return
	}

	events.ServerStatusManager.StopServer()
	c.JSON(http.StatusOK, gin.H{"message": "Server stopping..."})
}

func (h *Handlers) RestartServer(c *gin.Context) {
	// Add checks: avoid restarting if already restarting, starting, stopping?
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus ==
		events.Restarting ||
		currentStatus == events.Starting ||
		currentStatus == events.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is currently changing state"})
		return
	}
	// Allow restarting from Online or Offline state maybe? Adjust logic as needed.
	if currentStatus != events.Online && currentStatus != events.Offline {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Server cannot be restarted from current state",
		})
		return
	}

	events.ServerStatusManager.RestartServer()
	c.JSON(http.StatusOK, gin.H{"message": "Server restarting..."})
}
