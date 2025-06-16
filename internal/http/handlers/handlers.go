package handlers

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func UpdateModlist(c *gin.Context) {
	entries, err := os.ReadDir(modsPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]Mod, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".jar") {
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			out = append(out, Mod{Name: name})
		}
	}

	c.JSON(http.StatusOK, ModList{Mods: out})
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
			continue // silently skip non‑jar
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
	c.JSON(http.StatusCreated, gin.H{"mods": uploaded})
}

func DeleteMod(c *gin.Context) {
	var modsDir = `C:\Users\simon\curseforge\minecraft\Instances\MMFC-PLUS\mods`
	modName := c.Param("name") // e.g. ad_astra-forge-1.20.1-1.15.20

	// rudimentary sanitisation
	if strings.Contains(modName, "..") || strings.ContainsAny(modName, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mod name"})
		return
	}

	target := filepath.Join(modsDir, modName+".jar")
	// extra safety: ensure we're still inside modsDir after Join/Clean
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(modsDir)) {
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

	c.Status(http.StatusNoContent) // 204
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
