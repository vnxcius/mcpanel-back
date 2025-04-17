package status

import (
	"sync"
	"time"
)

type ServerStatus string

const (
	Starting   ServerStatus = "starting"
	Online     ServerStatus = "online"
	Offline    ServerStatus = "offline"
	Restarting ServerStatus = "restarting"
	Stopping   ServerStatus = "stopping"
)

// statusManager holds the current state and manages broadcasting
type StatusManager struct {
	mu            sync.RWMutex
	currentStatus ServerStatus
	clients       map[chan ServerStatus]bool // Set of client channels
	updatesChan   chan ServerStatus          // Channel to send status updates into the manager
}

// NewStatusManager creates and initializes the manager
func NewStatusManager() *StatusManager {
	sm := &StatusManager{
		currentStatus: Offline, // Initial status
		clients:       make(map[chan ServerStatus]bool),
		updatesChan:   make(chan ServerStatus, 1), // Buffered channel
	}
	go sm.runBroadcaster() // Start the broadcaster goroutine
	return sm
}

// GetStatus safely returns the current status
func (sm *StatusManager) GetStatus() ServerStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentStatus
}

// SetStatus safely updates the status and triggers a broadcast
func (sm *StatusManager) SetStatus(newStatus ServerStatus) {
	sm.mu.Lock()
	if sm.currentStatus == newStatus {
		sm.mu.Unlock()
		return // No change, do nothing
	}
	sm.currentStatus = newStatus
	sm.mu.Unlock()

	// Send update to the broadcaster non-blockingly
	// This decouples the status setting from the broadcasting logic
	select {
	case sm.updatesChan <- newStatus:
	default:
		// Broadcaster might be busy, maybe log this if important
	}
}

// runBroadcaster listens for updates and sends them to clients
func (sm *StatusManager) runBroadcaster() {
	for statusUpdate := range sm.updatesChan {
		sm.mu.RLock() // Lock for reading the client list
		for clientChan := range sm.clients {
			// Send non-blockingly to prevent slow clients from blocking others
			select {
			case clientChan <- statusUpdate:
			default:
				// Client channel buffer is full or client is slow, maybe log or handle
			}
		}
		sm.mu.RUnlock()
	}
}

// AddClient registers a new client channel
func (sm *StatusManager) AddClient(clientChan chan ServerStatus) {
	sm.mu.Lock()
	sm.clients[clientChan] = true
	sm.mu.Unlock()
	// Send current status immediately
	clientChan <- sm.GetStatus()
}

// RemoveClient unregisters a client channel
func (sm *StatusManager) RemoveClient(clientChan chan ServerStatus) {
	sm.mu.Lock()
	delete(sm.clients, clientChan)
	close(clientChan) // Close the channel to signal listener to stop
	sm.mu.Unlock()
}

// --- Simulation Logic ---

// SimulateOperation simulates changing status over time
func (sm *StatusManager) SimulateOperation(startStatus, endStatus ServerStatus, delay time.Duration) {
	sm.SetStatus(startStatus)
	time.Sleep(delay) // Simulate work
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
