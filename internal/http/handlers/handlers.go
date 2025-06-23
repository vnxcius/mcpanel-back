package handlers

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/helpers"
	"github.com/vnxcius/mcpanel-back/internal/http/events"
)

var (
	modsDir string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	modsDir = os.Getenv("MODS_PATH")
}

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func ServeWebSocket(c *gin.Context) {
	conn, err := events.WebsocketUpgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		slog.Error("Error upgrading connection to websocket", "error", err)
		return
	}

	ip := c.ClientIP()
	events.Manager.AddClient(conn, ip)

	slog.Info("WebSocket client connected", "ip", ip)
}

func UpdateModlist(c *gin.Context) {
	err := events.Manager.UpdateModlist()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nil)
}

func UploadMods(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files"})
		return
	}

	uploaded, skipped, err := helpers.UploadModsToDir(files, modsDir, c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events.Manager.UpdateModlist()
	c.JSON(http.StatusCreated, gin.H{
		"mods":    uploaded,
		"skipped": skipped,
	})
}

func UpdateMod(c *gin.Context) {
	oldModBase := c.Param("name")
	form, err := c.MultipartForm()
	if err != nil || len(form.File["file"]) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
		return
	}
	file := form.File["file"][0]

	if strings.Contains(oldModBase, "..") || strings.ContainsAny(oldModBase, `\/`) ||
		strings.Contains(file.Filename, "..") || strings.ContainsAny(file.Filename, `\/`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid name"})
		return
	}

	if err := helpers.UpdateModFromDir(file, modsDir, oldModBase, c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events.Manager.UpdateModlist()
	c.Status(http.StatusNoContent) // 204
}

func DeleteMod(c *gin.Context) {
	modName := c.Param("name")

	// rudimentary sanitisation
	if strings.Contains(modName, "..") || strings.ContainsAny(modName, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mod name"})
		return
	}

	if err := helpers.DeleteModFromDir(modsDir, modName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events.Manager.UpdateModlist()
	c.Status(http.StatusNoContent) // 204
}

func StartServer(c *gin.Context) {
	currentStatus := events.Manager.GetStatus()
	if currentStatus == events.Online || currentStatus == events.Starting {
		slog.Info("Received request to start server, but server is already online or starting")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está ligado ou iniciando",
		})
		return
	}

	slog.Info("Server is starting...")
	events.Manager.StartServer()

	c.JSON(http.StatusOK, gin.H{"message": "O servidor está iniciando..."})
}

func StopServer(c *gin.Context) {
	currentStatus := events.Manager.GetStatus()
	if currentStatus == events.Offline || currentStatus == events.Stopping {
		slog.Info("Received request to stop server, but server is already offline or stopping")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está desligado ou parando",
		})
		return
	}

	events.Manager.StopServer()

	slog.Info("Server stopping...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está parando..."})
}

func RestartServer(c *gin.Context) {
	currentStatus := events.Manager.GetStatus()
	if currentStatus != events.Online && currentStatus != events.Offline {
		slog.Info("Received request to restart server, but server is currently changing state")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor está ocupado em outra operação",
		})
		return
	}

	events.Manager.RestartServer()

	slog.Info("Server restarting...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está reiniciando..."})
}
