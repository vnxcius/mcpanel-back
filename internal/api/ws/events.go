package ws

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/vnxcius/mcpanel-back/internal/utils"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type StatusUpdateEvent struct {
	Status string `json:"status"`
}

type EventHandler func(event Event, c *Client) error

const (
	EventStatusUpdate     = "status_update"
	EventModAdded         = "mod_added"
	EventModDeleted       = "mod_deleted"
	EventModUpdated       = "mod_updated"
	EventModlist          = "modlist"
	EventModlistChangelog = "modlist_changelog"
	EventLogAppend        = "log_append"
	EventLogSnapshot      = "log_snapshot"
)

func simulateOperation(startStatus, endStatus string, delay time.Duration) {
	Manager.SetStatus(startStatus)
	time.Sleep(delay)
	Manager.SetStatus(endStatus)
}

// Start -> Online in 2s
func simulateStart() {
	slog.Info("Simulating server start...")
	go simulateOperation("starting", "online", 2*time.Second)
}

// Stop -> Offline in 2s
func simulateStop() {
	slog.Info("Simulating server stop...")
	go simulateOperation("stopping", "offline", 2*time.Second)
}

// Restart -> Online in 2s
func simulateRestart() {
	slog.Info("Simulating server restart...")
	go simulateOperation("restarting", "online", 2*time.Second)
}

func runServerScript(action string) error {
	cmd := exec.Command("sudo", "/opt/mcpanel-back/cmd/api/minecraft-server.sh", action)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Command '%s' failed: %v\nOutput:\n%s", action, err, string(output))
	}
	return err
}

// Returns the current server status stored in the Manager
func (m *WSManager) GetStatus() string {
	m.RLock()
	defer m.RUnlock()
	return m.currentStatus
}

func (m *WSManager) SetStatus(status string) {
	m.Lock()
	m.currentStatus = status
	m.Unlock()

	payload, err := json.Marshal(StatusUpdateEvent{Status: status})
	if err != nil {
		slog.Error("Error marshalling message", "error", err)
		return
	}
	m.broadcast(Event{
		Type:    EventStatusUpdate,
		Payload: payload,
	})

	slog.Info("Server status updated", "status", status)
}

func (m *WSManager) UpdateModlist(eventType string, payload json.RawMessage) {
	m.broadcast(Event{
		Type:    eventType,
		Payload: payload,
	})

	// Update modlist changelog
	modlistChangelog, err := utils.GetModlistChangelog()
	if err == nil {
		payload, _ := json.Marshal(modlistChangelog)
		m.broadcast(Event{
			Type:    EventModlistChangelog,
			Payload: payload,
		})
	}
}

func (m *WSManager) StartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateStart()
			return
		}

		m.SetStatus("starting")
		if err := runServerScript("start"); err != nil {
			slog.Error("Failed to start server", "error", err)
			m.SetStatus("offline")
			return
		}

		isOnline := utils.WaitMinecraftServer("online")
		if !isOnline {
			m.SetStatus("offline")
			return
		}

		m.SetStatus("online")
	}()
}

func (m *WSManager) StopServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateStop()
			return
		}

		m.SetStatus("stopping")
		if err := runServerScript("stop"); err != nil {
			slog.Error("Failed to stop server:", "error", err)
			m.SetStatus("online")
			return
		}

		isOffline := utils.WaitMinecraftServer("offline")
		if isOffline {
			m.SetStatus("offline")
			return
		}

		m.SetStatus("online")
	}()
}

func (m *WSManager) RestartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateRestart()
			return
		}

		m.SetStatus("restarting")
		if err := runServerScript("restart"); err != nil {
			slog.Error("Failed to restart server:", "error", err)
			m.SetStatus("online")
			return
		}

		isOnline := utils.WaitMinecraftServer("online")
		if !isOnline {
			m.SetStatus("offline")
			return
		}

		m.SetStatus("online")
	}()
}
