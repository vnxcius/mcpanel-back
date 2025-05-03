package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/internal/http/events"
)

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func Status(c *gin.Context) {
	status := events.ServerStatusManager.GetStatus()
	if status == "" {
		status = "Cannot determine status"
	}
	slog.Info("Sending current server status", "status", status)
	c.JSON(http.StatusOK, gin.H{"message": status})
}

func StatusStream(c *gin.Context) {
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

func StartServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus == events.Online || currentStatus == events.Starting {
		slog.Info("Received request to start server, but server is already online or starting")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está ligado ou iniciando",
		})
		return
	}

	// events.ServerStatusManager.SimulateStart()
	events.ServerStatusManager.StartServer()

	slog.Info("Server is starting...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está iniciando..."})
}

func StopServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus == events.Offline || currentStatus == events.Stopping {
		slog.Info("Received request to stop server, but server is already offline or stopping")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está desligado ou parando",
		})
		return
	}

	// events.ServerStatusManager.SimulateStop()
	events.ServerStatusManager.StopServer()

	slog.Info("Server stopping...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está parando..."})
}

func RestartServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus != events.Online && currentStatus != events.Offline {
		slog.Info("Received request to restart server, but server is currently changing state")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor está ocupado em outra operação",
		})
		return
	}

	// events.ServerStatusManager.SimulateRestart()
	events.ServerStatusManager.RestartServer()

	slog.Info("Server restarting...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está reiniciando..."})
}
