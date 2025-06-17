package events

import (
	"log/slog"
	"time"
)

func (m *WSManager) SimulateOperation(startStatus, endStatus ServerStatus, delay time.Duration) {
	m.SetStatus(startStatus)
	time.Sleep(delay)
	m.SetStatus(endStatus)
}

func (sm *WSManager) SimulateStart() {
	slog.Info("Simulating server start...")
	go sm.SimulateOperation(Starting, Online, 5*time.Second) // Start -> Online in 5s
}

func (sm *WSManager) SimulateStop() {
	slog.Info("Simulating server stop...")
	go sm.SimulateOperation(Stopping, Offline, 3*time.Second) // Stop -> Offline in 3s
}

func (sm *WSManager) SimulateRestart() {
	slog.Info("Simulating server restart...")
	go func() {
		sm.SetStatus(Restarting)
		time.Sleep(5 * time.Second)
		sm.SetStatus(Online)
	}() // Restart -> Online in 5s
}
