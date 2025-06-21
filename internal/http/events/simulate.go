package events

import (
	"log/slog"
	"time"
)

func simulateOperation(startStatus, endStatus ServerStatus, delay time.Duration) {
	Manager.SetStatus(startStatus)
	time.Sleep(delay)
	Manager.SetStatus(endStatus)
}

// Start -> Online in 2s
func simulateStart() {
	slog.Info("Simulating server start...")
	go simulateOperation(Starting, Online, 2*time.Second)
}

// Stop -> Offline in 2s
func simulateStop() {
	slog.Info("Simulating server stop...")
	go simulateOperation(Stopping, Offline, 2*time.Second)
}

// Restart -> Online in 2s
func simulateRestart() {
	slog.Info("Simulating server restart...")
	go simulateOperation(Restarting, Online, 2*time.Second)
}
