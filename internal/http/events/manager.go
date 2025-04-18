package events

import (
	"log"
	"sync"
)

type StatusManager struct {
	mu            sync.RWMutex
	currentStatus ServerStatus
	clients       map[chan ServerStatus]bool
	updatesChan   chan ServerStatus
}

var ServerStatusManager *StatusManager

func InitializeStatusManager() {
	ServerStatusManager = NewStatusManager()
	log.Println("Status manager initialized")
}

func NewStatusManager() *StatusManager {
	sm := &StatusManager{
		currentStatus: Offline,
		clients:       make(map[chan ServerStatus]bool),
		updatesChan:   make(chan ServerStatus, 1),
	}
	go sm.runBroadcaster()
	return sm
}

func (sm *StatusManager) GetStatus() ServerStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentStatus
}

func (sm *StatusManager) SetStatus(newStatus ServerStatus) {
	sm.mu.Lock()
	if sm.currentStatus == newStatus {
		sm.mu.Unlock()
		return
	}
	sm.currentStatus = newStatus
	sm.mu.Unlock()

	select {
	case sm.updatesChan <- newStatus:
	default:
		// Broadcaster might be busy, maybe log this if important
		log.Println("Status update dropped")
	}
}

func (sm *StatusManager) runBroadcaster() {
	for statusUpdate := range sm.updatesChan {
		sm.mu.RLock()
		for clientChan := range sm.clients {
			select {
			case clientChan <- statusUpdate:
			default:
				log.Println("Status update dropped")
			}
		}
		sm.mu.RUnlock()
	}
}

func (sm *StatusManager) AddClient(clientChan chan ServerStatus) {
	sm.mu.Lock()
	sm.clients[clientChan] = true
	sm.mu.Unlock()
	clientChan <- sm.GetStatus()
}

func (sm *StatusManager) RemoveClient(clientChan chan ServerStatus) {
	sm.mu.Lock()
	delete(sm.clients, clientChan)
	close(clientChan)
	sm.mu.Unlock()
}
