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
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already online or starting"})
		return
	}

	// events.ServerStatusManager.SimulateStart()
	events.ServerStatusManager.StartServer()

	c.JSON(http.StatusOK, gin.H{"message": "Server starting..."})
}

func StopServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus == events.Offline || currentStatus == events.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already offline or stopping"})
		return
	}

	// events.ServerStatusManager.SimulateStop()
	events.ServerStatusManager.StopServer()
	c.JSON(http.StatusOK, gin.H{"message": "Server stopping..."})
}

func RestartServer(c *gin.Context) {
	currentStatus := events.ServerStatusManager.GetStatus()
	if currentStatus ==
		events.Restarting ||
		currentStatus == events.Starting ||
		currentStatus == events.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is currently changing state"})
		return
	}

	if currentStatus != events.Online && currentStatus != events.Offline {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Server cannot be restarted from current state",
		})
		return
	}

	// events.ServerStatusManager.SimulateRestart()
	events.ServerStatusManager.RestartServer()
	c.JSON(http.StatusOK, gin.H{"message": "Server restarting..."})
}
