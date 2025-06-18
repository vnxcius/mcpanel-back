package handlers

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/http/events"
)

var (
	modsPath string
	logsPath string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	modsPath = os.Getenv("MODS_PATH")
	logsPath = os.Getenv("LOGS_PATH")
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

	events.Manager.AddClient(conn)

	slog.Info("WebSocket client connected", "ip", c.ClientIP())
}

func UpdateModlist(c *gin.Context) {
	err := events.Manager.UpdateModlist()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nil)
}

func GetLatestLogs(c *gin.Context) {
	// Read file
	data, err := os.ReadFile(filepath.Clean(logsPath))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read log file"})
		return
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

func UploadMods(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
		return
	}
	files := form.File["files"] // <input name="files" multiple>

	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files"})
		return
	}

	var uploaded []string
	for _, fh := range files {
		if !strings.EqualFold(filepath.Ext(fh.Filename), ".jar") {
			continue
		}

		dst := filepath.Join(modsPath, filepath.Base(fh.Filename))
		// extra safety – ensure dst is inside modsDir
		if !strings.HasPrefix(filepath.Clean(dst), filepath.Clean(modsPath)) {
			continue
		}
		if err := c.SaveUploadedFile(fh, dst); err == nil {
			uploaded = append(uploaded, fh.Filename)
		}
	}

	if len(uploaded) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no .jar files uploaded"})
		return
	}

	events.Manager.UpdateModlist()
	c.JSON(http.StatusCreated, gin.H{"mods": uploaded})
}

func DeleteMod(c *gin.Context) {
	modName := c.Param("name") // e.g. ad_astra-forge-1.20.1-1.15.20

	// rudimentary sanitisation
	if strings.Contains(modName, "..") || strings.ContainsAny(modName, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mod name"})
		return
	}

	target := filepath.Join(modsPath, modName+".jar")
	// extra safety: ensure we're still inside modsDir after Join/Clean
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(modsPath)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "mod not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
