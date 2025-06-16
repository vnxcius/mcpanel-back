package events

import (
	"log"
	"net"
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

func (sm *StatusManager) StartServer() {
	go func() {
		sm.SetStatus(Starting)

		if err := runServerScript("start"); err != nil {
			log.Println("Failed to start server:", err)
			sm.SetStatus(Offline)
			return
		}

		waitUntilOnline(sm)
	}()
}

func (sm *StatusManager) StopServer() {
	go func() {
		sm.SetStatus(Stopping)

		if err := runServerScript("stop"); err != nil {
			log.Println("Failed to stop server:", err)
			sm.SetStatus(Online)
			return
		}

		time.Sleep(2 * time.Second)
		sm.SetStatus(Offline)
	}()
}

func (sm *StatusManager) RestartServer() {
	go func() {
		sm.SetStatus(Restarting)

		if err := runServerScript("restart"); err != nil {
			log.Println("Failed to restart server:", err)
			sm.SetStatus(Online)
			return
		}

		waitUntilOnline(sm)
	}()
}

func waitUntilOnline(sm *StatusManager) {
	const (
		address = "127.0.0.1:25565"
		timeout = 2 * time.Second
	)

	for range 120 { // retry for up to ~120 seconds
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			conn.Close()
			sm.SetStatus(Online)
			return
		}
		time.Sleep(1 * time.Second)
	}

	// Timed out
	sm.SetStatus(Offline)
}
