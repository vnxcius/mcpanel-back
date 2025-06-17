package events

import (
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"time"
)

func runServerScript(action string) error {
	cmd := exec.Command("sudo", "/opt/mcpanel-back/cmd/api/minecraft-server.sh", action)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Command '%s' failed: %v\nOutput:\n%s", action, err, string(output))
	}
	return err
}

func (m *WSManager) GetStatus() ServerStatus {
	m.RLock()
	defer m.RUnlock()
	return m.currentStatus
}

func (m *WSManager) SetStatus(status ServerStatus) {
	// 1) persist the new status
	m.Lock()
	m.currentStatus = status
	m.Unlock()

	// 2) build & broadcast the event
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

func (m *WSManager) StartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") == "development" {
			m.SimulateStart()
			return
		}

		m.SetStatus(Starting)
		if err := runServerScript("start"); err != nil {
			log.Println("Failed to start server:", err)
			m.SetStatus(Offline)
			return
		}

		waitUntilOnline(m)
	}()
}

func (m *WSManager) StopServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") == "development" {
			m.SimulateStop()
			return
		}

		m.SetStatus(Stopping)
		if err := runServerScript("stop"); err != nil {
			log.Println("Failed to stop server:", err)
			m.SetStatus(Online)
			return
		}

		time.Sleep(2 * time.Second)
		m.SetStatus(Offline)
	}()
}

func (m *WSManager) RestartServer() {
	go func() {
		if os.Getenv("ENVIRONMENT") == "development" {
			m.SimulateRestart()
			return
		}

		m.SetStatus(Restarting)
		if err := runServerScript("restart"); err != nil {
			log.Println("Failed to restart server:", err)
			m.SetStatus(Online)
			return
		}

		waitUntilOnline(m)
	}()
}

func waitUntilOnline(m *WSManager) {
	const (
		address = "127.0.0.1:25565"
		timeout = 2 * time.Second
	)

	for range 120 { // retry for up to ~120 seconds
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			conn.Close()
			m.SetStatus(Online)
			return
		}
		time.Sleep(1 * time.Second)
	}

	// Timed out
	m.SetStatus(Offline)
}
