package events

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"slices"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/helpers"
)

var (
	WebsocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}

	Manager        *WSManager
	logsPath       string
	allowedOrigins []string

	once sync.Once
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logsPath = os.Getenv("LOGS_PATH")
	allowedOrigins = strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
}

func contains(s []string, str string) bool {
	return slices.Contains(s, str)
}

func checkOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); !contains(allowedOrigins, origin) {
		return false
	}
	return true
}

func newManager() *WSManager {
	status := Offline
	if helpers.IsMinecraftCurrentlyOnline() {
		status = Online
	}
	return &WSManager{
		clients:       make(ClientList),
		handlers:      make(map[string]EventHandler),
		currentStatus: status,
	}
}

func InitializeManager() {
	Manager = newManager()
}

func (m *WSManager) AddClient(conn *websocket.Conn, ip string) {
	once.Do(func() {
		tailLogs()
	})
	c := NewClient(conn, m, ip)

	m.Lock()
	m.clients[c] = true
	m.Unlock()

	go m.syncWithMinecraft()
	go c.WriteMessages()
	go c.ReadMessages()

	// update server status
	statusPayload, _ := json.Marshal(StatusUpdateEvent{
		Status: m.GetStatus(),
	})
	c.send(Event{
		Type:    EventStatusUpdate,
		Payload: statusPayload,
	})

	// update modlist
	modPayload, err := helpers.GetMods()
	if err == nil {
		c.send(Event{
			Type:    EventModlistUpdate,
			Payload: modPayload,
		})
	} else {
		slog.Error("Failed to get mod list on client connect", "error", err)
	}

	// send log snapshot
	logSnapshot, err := getLastLogLines(350)
	if err == nil {
		payload, _ := json.Marshal(struct {
			Lines []string `json:"lines"`
		}{Lines: logSnapshot})

		c.send(Event{
			Type:    EventLogSnapshot,
			Payload: payload,
		})
	}
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
	m.RLock()
	defer m.RUnlock()

	for c := range m.clients {
		select {
		case c.egress <- evt:
			slog.Info("Broadcasting event", "type", evt.Type)
		default:
			slog.Warn("client buffer full, dropping event")
		}
	}
}

func (m *WSManager) syncWithMinecraft() {
	slog.Info("Syncing with Minecraft server...")
	status := m.GetStatus()

	if helpers.IsMinecraftCurrentlyOnline() && status == Offline {
		m.SetStatus(Online)
		slog.Info("Minecraft server corrected to online")
		return
	}

	if !helpers.IsMinecraftCurrentlyOnline() && status == Online {
		m.SetStatus(Offline)
		slog.Info("Minecraft server corrected to offline")
		return
	}
}
