package events

import (
	"context"
	"log"
	"net"
	"os/exec"
	"time"

)

func (sm *StatusManager) StartServer() {
	go func() {
		sm.SetStatus(Starting)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "systemctl", "start", "minecraft")
		if err := cmd.Run(); err != nil {
			log.Println("Failed to start server:", err)
			sm.SetStatus(Offline)
			return
		}

		// Wait for server to actually be ready for players
		waitUntilOnline(sm)
	}()
}

func (sm *StatusManager) StopServer() {
	go func() {
		sm.SetStatus(Stopping)

		cmd := exec.Command("systemctl", "stop", "minecraft")
		if err := cmd.Run(); err != nil {
			log.Println("Failed to stop server:", err)
			sm.SetStatus(Online)
			return
		}

		// Wait a bit to ensure it's down
		time.Sleep(2 * time.Second)
		sm.SetStatus(Offline)
	}()
}

func (sm *StatusManager) RestartServer() {
	go func() {
		sm.SetStatus(Restarting)

		cmd := exec.Command("systemctl", "restart", "minecraft")
		if err := cmd.Run(); err != nil {
			log.Println("Failed to restart server:", err)
			sm.SetStatus(Offline)
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

	for range 30 { // retry for up to ~30 seconds
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

