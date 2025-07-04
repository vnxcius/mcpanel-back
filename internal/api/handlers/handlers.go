package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/api/ws"
	"github.com/vnxcius/mcpanel-back/internal/utils"
	"github.com/vnxcius/mcpanel-back/internal/logging"
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

func ServeWebSocket(c *gin.Context) {
	conn, err := ws.WebsocketUpgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		slog.Error("Error upgrading connection to websocket", "error", err)
		return
	}

	ip := c.ClientIP()
	ws.Manager.AddClient(conn, ip)
	slog.Info("WebSocket client connected", "ip", ip)
}

func GetModlist(c *gin.Context) {
	data, err := utils.GetMods()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var parsed struct {
		Mods []any `json:"mods"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid mod list format"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mods": parsed.Mods})
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

	uploaded, skipped, err := utils.UploadModsToDir(files, modsDir, c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, uploadedMod := range uploaded {
		change := logging.LogModChange(uploadedMod, logging.ModAdded)

		payload, err := json.Marshal(change)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ws.Manager.UpdateModlist(ws.EventModAdded, payload)
	}

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

	if err := utils.UpdateModFromDir(file, modsDir, oldModBase, c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	change := logging.LogModChange(
		fmt.Sprintf("%s → %s", oldModBase, file.Filename),
		logging.ModUpdated,
	)

	payload, err := json.Marshal(change)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ws.Manager.UpdateModlist(ws.EventModUpdated, payload)
	c.Status(http.StatusNoContent) // 204
}

func DeleteMod(c *gin.Context) {
	modName := c.Param("name")

	// rudimentary sanitisation
	if strings.Contains(modName, "..") || strings.ContainsAny(modName, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mod name"})
		return
	}

	if err := utils.DeleteModFromDir(modsDir, modName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	change := logging.LogModChange(modName, logging.ModDeleted)

	payload, err := json.Marshal(change)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ws.Manager.UpdateModlist(ws.EventModDeleted, payload)
	c.Status(http.StatusNoContent) // 204
}

func DownloadMod(c *gin.Context) {
	name := c.Param("name")
	path := filepath.Join(modsDir, name)

	f, err := os.Open(path)
	if err != nil {
		c.String(http.StatusNotFound, "mod not found")
		return
	}
	defer f.Close()

	c.Header("Content-Disposition", "attachment; filename="+name)
	c.Header("Content-Type", "application/java-archive")
	c.File(path)
}

func GetModsChangelog(c *gin.Context) {
	const logDir = "./logs/modlist-changelog"

	files, err := os.ReadDir(logDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read changelog dir"})
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	var allChanges []map[string]any

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		path := filepath.Join(logDir, file.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var entry map[string]any
			if err := json.Unmarshal([]byte(scanner.Text()), &entry); err == nil {
				allChanges = append(allChanges, entry)
			}
		}
		f.Close()
	}

	c.JSON(http.StatusOK, gin.H{"changes": allChanges})
}

func GetServerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": ws.Manager.GetStatus()})
}

func StartServer(c *gin.Context) {
	currentStatus := ws.Manager.GetStatus()
	if currentStatus == "online" || currentStatus == "starting" {
		slog.Info("Received request to start server, but server is already online or starting")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está ligado ou iniciando",
		})
		return
	}

	slog.Info("Server is starting...")
	ws.Manager.StartServer()

	c.JSON(http.StatusOK, gin.H{"message": "O servidor está iniciando..."})
}

func StopServer(c *gin.Context) {
	currentStatus := ws.Manager.GetStatus()
	if currentStatus == "offline" || currentStatus == "stopping" {
		slog.Info("Received request to stop server, but server is already offline or stopping")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor já está desligado ou parando",
		})
		return
	}

	ws.Manager.StopServer()

	slog.Info("Server stopping...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está parando..."})
}

func RestartServer(c *gin.Context) {
	currentStatus := ws.Manager.GetStatus()
	if currentStatus != "online" && currentStatus != "offline" {
		slog.Info("Received request to restart server, but server is currently changing state")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "O servidor está ocupado em outra operação",
		})
		return
	}

	ws.Manager.RestartServer()

	slog.Info("Server restarting...")
	c.JSON(http.StatusOK, gin.H{"message": "O servidor está reiniciando..."})
}
