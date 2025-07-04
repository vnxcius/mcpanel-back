package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"slices"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/vnxcius/mcpanel-back/internal/otp"
	"github.com/vnxcius/mcpanel-back/internal/utils"
)

type WSManager struct {
	clients ClientList
	sync.RWMutex
	handlers map[string]EventHandler
	otps     otp.RetentionMap

	currentStatus string
}

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

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	if origin != "" && slices.Contains(allowedOrigins, origin) {
		return true
	}

	token := r.Header.Get("X-Bot-Token")
	slog.Warn("Origin not allowed, checking if is discord bot")
	return token == os.Getenv("DISCORD_BOT_TOKEN")
}

func newManager(ctx context.Context) *WSManager {
	status := "offline"
	if utils.IsMinecraftCurrentlyOnline() {
		status = "online"
	}
	return &WSManager{
		clients:       make(ClientList),
		handlers:      make(map[string]EventHandler),
		currentStatus: status,
		otps:          otp.NewRetentionMap(ctx, 5*time.Minute),
	}
}

func InitializeManager() {
	//TODO: pass context
	Manager = newManager(context.Background())
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
	modPayload, err := utils.GetMods()
	if err == nil {
		c.send(Event{
			Type:    EventModlist,
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

	// changelog
	modlistChangelog, err := utils.GetModlistChangelog()
	if err == nil {
		payload, _ := json.Marshal(modlistChangelog)
		c.send(Event{
			Type:    EventModlistChangelog,
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
			slog.Debug("Broadcasting event", "type", evt.Type)
		default:
			slog.Warn("client buffer full, dropping event")
		}
	}
}

func (m *WSManager) syncWithMinecraft() {
	slog.Info("Syncing with Minecraft server...")
	status := m.GetStatus()

	if utils.IsMinecraftCurrentlyOnline() && status == "offline" {
		m.SetStatus("online")
		slog.Info("Minecraft server corrected to online")
		return
	}

	if !utils.IsMinecraftCurrentlyOnline() && status == "online" {
		m.SetStatus("offline")
		slog.Info("Minecraft server corrected to offline")
		return
	}
}
