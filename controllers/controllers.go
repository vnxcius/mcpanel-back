package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/vnxcius/sss-backend/status"
)

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

var ServerStatusMgr *status.StatusManager

func InitializeStatusManager() {
	ServerStatusMgr = status.NewStatusManager()
}

// StatusStream handles SSE connections
func StatusStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	clientChan := make(chan status.ServerStatus, 1)
	ServerStatusMgr.AddClient(clientChan)
	defer ServerStatusMgr.RemoveClient(clientChan)

	// Use context to detect client disconnect
	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// Handle error: streaming not supported
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Streaming unsupported"})
		return
	}

	// Send initial status (already handled by AddClient, but explicit send can also work)
	// initialStatus := ServerStatusMgr.GetStatus()
	// fmt.Fprintf(c.Writer, "data: {\"status\": \"%s\"}\n\n", initialStatus)
	// flusher.Flush()

	for {
		select {
		case <-ctx.Done(): // Client disconnected
			fmt.Println("SSE client disconnected")
			return
		case statusUpdate := <-clientChan: // Received status update from broadcaster
			// Format message according to SSE spec: "data: json_payload\n\n"
			_, err := fmt.Fprintf(c.Writer, "data: {\"status\": \"%s\"}\n\n", statusUpdate)
			if err != nil {
				// Handle error writing to client (client likely disconnected)
				fmt.Printf("Error writing to SSE client: %v\n", err)
				return
			}
			flusher.Flush() // Send data immediately
		}
	}
}

func VerifyToken(c *gin.Context) {
	validToken := os.Getenv("TOKEN")
	var req VerifyTokenRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if validToken == req.Token {
		c.SetCookie(
			"sss-token",
			validToken,
			30*24*60*60, // 30 days
			"/",
			"",
			false,
			true,
		)

		c.JSON(http.StatusOK, gin.H{"message": "Token verificado com sucesso"})

		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"message": "Token inválido"})
}

func StartServer(c *gin.Context) {
	// Add checks: Is it already online or starting?
	currentStatus := ServerStatusMgr.GetStatus()
	if currentStatus == status.Online || currentStatus == status.Starting {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already online or starting"})
		return
	}

	// Trigger the simulation (which updates status via SetStatus)
	ServerStatusMgr.SimulateStart()

	c.JSON(http.StatusOK, gin.H{"message": "Server starting..."})
}

func StopServer(c *gin.Context) {
	// Add checks: Is it already offline or stopping?
	currentStatus := ServerStatusMgr.GetStatus()
	if currentStatus == status.Offline || currentStatus == status.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is already offline or stopping"})
		return
	}

	ServerStatusMgr.SimulateStop()
	c.JSON(http.StatusOK, gin.H{"message": "Server stopping..."})
}

func RestartServer(c *gin.Context) {
	// Add checks: avoid restarting if already restarting, starting, stopping?
	currentStatus := ServerStatusMgr.GetStatus()
	if currentStatus == status.Restarting || currentStatus == status.Starting || currentStatus == status.Stopping {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server is currently changing state"})
		return
	}
	// Allow restarting from Online or Offline state maybe? Adjust logic as needed.
	if currentStatus != status.Online && currentStatus != status.Offline {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Server cannot be restarted from current state"})
		return
	}

	ServerStatusMgr.SimulateRestart()
	c.JSON(http.StatusOK, gin.H{"message": "Server restarting..."})
}
