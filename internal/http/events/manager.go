package events

import (
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"slices"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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

func newManager(mcServerAddr string) *WSManager {
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
	Manager = newManager("localhost:25565")
}

func (m *WSManager) AddClient(conn *websocket.Conn) {
	once.Do(func() {
		tailLogs()
	})
	c := NewClient(conn, m)

	m.Lock()
	m.clients[c] = true
	m.Unlock()

	go m.syncWithMinecraft()
	go c.WriteMessages()
	go c.ReadMessages()

	// Update server status
	payload, _ := json.Marshal(StatusUpdateEvent{Status: m.GetStatus()})
	c.send(Event{Type: EventStatusUpdate, Payload: payload})

	// Update modlist
	modPayload, err := m.getModlistPayload()
	if err == nil {
		c.send(Event{Type: EventModlistUpdate, Payload: modPayload})
	} else {
		slog.Error("Failed to get mod list on client connect", "error", err)
	}

	// Send log snapshot
	logSnapshot, err := m.getLastLogLines(200)
	if err == nil {
		payload, _ := json.Marshal(struct {
			Lines []string `json:"lines"`
		}{Lines: logSnapshot})

		c.send(Event{Type: EventLogSnapshot, Payload: payload})
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
	if isMinecraftOnline(m.serverAddr) && m.GetStatus() != Online {
		slog.Info("Minecraft server corrected to online")
		m.SetStatus(Online)
	} else if !isMinecraftOnline(m.serverAddr) && m.GetStatus() != Offline {
		slog.Info("Minecraft server corrected to offline")
		m.SetStatus(Offline)
	}
}

func (m *WSManager) getModlistPayload() ([]byte, error) {
	entries, err := os.ReadDir(os.Getenv("MODS_PATH"))
	if err != nil {
		return nil, err
	}

	var mods []Mod
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".jar") {
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			mods = append(mods, Mod{Name: name})
		}
	}

	return json.Marshal(ModList{Mods: mods})
}
