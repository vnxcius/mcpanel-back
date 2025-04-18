package events

import "time"

func (sm *StatusManager) SimulateOperation(startStatus, endStatus ServerStatus, delay time.Duration) {
	sm.SetStatus(startStatus)
	time.Sleep(delay)
	sm.SetStatus(endStatus)
}

func (sm *StatusManager) SimulateStart() {
	go sm.SimulateOperation(Starting, Online, 5*time.Second) // Start -> Online in 5s
}

func (sm *StatusManager) SimulateStop() {
	go sm.SimulateOperation(Stopping, Offline, 3*time.Second) // Stop -> Offline in 3s
}

func (sm *StatusManager) SimulateRestart() {
	go func() {
		sm.SetStatus(Restarting)
		time.Sleep(5 * time.Second)
		sm.SetStatus(Online)
	}()
}
