package events

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/exec"

	"github.com/vnxcius/mcpanel-back/internal/helpers"
)

func runServerScript(action string) error {
	cmd := exec.Command("sudo", "/opt/mcpanel-back/cmd/api/minecraft-server.sh", action)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Command '%s' failed: %v\nOutput:\n%s", action, err, string(output))
	}
	return err
}

// Returns the current server status stored in the Manager
func (m *WSManager) GetStatus() ServerStatus {
	m.RLock()
	defer m.RUnlock()
	return m.currentStatus
}

func (m *WSManager) SetStatus(status ServerStatus) {
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

func (m *WSManager) UpdateModlist() error {
	payload, err := helpers.GetMods()
	if err != nil {
		slog.Error("Error marshalling message", "error", err)
		return err
	}

	m.broadcast(Event{
		Type:    EventModlistUpdate,
		Payload: payload,
	})

	return nil
}

func (m *WSManager) StartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateStart()
			return
		}

		m.SetStatus(Starting)
		if err := runServerScript("start"); err != nil {
			slog.Error("Failed to start server", "error", err)
			m.SetStatus(Offline)
			return
		}

		isOnline := helpers.WaitMinecraftServer("online")
		if !isOnline {
			m.SetStatus(Offline)
			return
		}

		m.SetStatus(Online)
	}()
}

func (m *WSManager) StopServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateStop()
			return
		}

		m.SetStatus(Stopping)
		if err := runServerScript("stop"); err != nil {
			slog.Error("Failed to stop server:", "error", err)
			m.SetStatus(Online)
			return
		}

		isOffline := helpers.WaitMinecraftServer("offline")
		if isOffline {
			m.SetStatus(Offline)
			return
		}

		m.SetStatus(Online)
	}()
}

func (m *WSManager) RestartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") != "production" {
			simulateRestart()
			return
		}

		m.SetStatus(Restarting)
		if err := runServerScript("restart"); err != nil {
			slog.Error("Failed to restart server:", "error", err)
			m.SetStatus(Online)
			return
		}

		isOnline := helpers.WaitMinecraftServer("online")
		if !isOnline {
			m.SetStatus(Offline)
			return
		}

		m.SetStatus(Online)
	}()
}
