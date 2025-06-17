package events

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var (
	WebsocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if origin := r.Header.Get("Origin"); origin != os.Getenv("ALLOWED_ORIGINS") {
				return false
			}
			return true
		},
	}

	Manager *WSManager
)

func NewManager(mcServerAddr string) *WSManager {
	status := Offline
	if isMinecraftOnline(mcServerAddr) {
		status = Online
	}
	return &WSManager{
		clients:       make(ClientList),
		handlers:      make(map[string]EventHandler),
		serverAddr:    mcServerAddr,
		currentStatus: status,
	}
}

func InitializeManager() {
	Manager = NewManager("localhost:25565")
}

func (m *WSManager) AddClient(conn *websocket.Conn) {
	c := NewClient(conn, m)

	m.Lock()
	m.clients[c] = true
	m.Unlock()

	go m.syncWithMinecraft()

	payload, _ := json.Marshal(StatusUpdateEvent{Status: m.GetStatus()})
	c.send(Event{Type: EventStatusUpdate, Payload: payload})

	go c.WriteMessages()
	go c.ReadMessages()
}

func (m *WSManager) RemoveClient(c *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[c]; ok {
		c.connection.Close()
		delete(m.clients, c)
	}
}

func (m *WSManager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("Unknown event type: " + event.Type)
	}
}

func (m *WSManager) broadcast(evt Event) {
	slog.Info("Broadcasting event", "type", evt.Type, "payload", string(evt.Payload))
	m.RLock()
	defer m.RUnlock()

	for c := range m.clients {
		select {
		case c.egress <- evt:
			slog.Info("Enqueued event for client", "type", evt.Type)
		default:
			slog.Warn("client buffer full, dropping event")
		}
	}
}

func (m *WSManager) syncWithMinecraft() {
	slog.Info("Syncing with Minecraft server...")
	if isMinecraftOnline(m.serverAddr) && m.GetStatus() != Online {
		slog.Info("Minecraft server corrected to online")
		m.SetStatus(Online)
	} else if !isMinecraftOnline(m.serverAddr) && m.GetStatus() != Offline {
		slog.Info("Minecraft server corrected to offline")
		m.SetStatus(Offline)
	}
}
