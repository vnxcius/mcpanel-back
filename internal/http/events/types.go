package events

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Mod struct {
	Name string `json:"name"`
}

type ModList struct {
	Mods []Mod `json:"mods"`
}

type ServerStatus string

type WSManager struct {
	clients ClientList
	sync.RWMutex
	handlers map[string]EventHandler

	currentStatus ServerStatus
	serverAddr    string
}

type Client struct {
	connection *websocket.Conn
	manager    *WSManager

	// Buffered channel of outbound messages
	egress chan Event
}

type ClientList map[*Client]bool

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type StatusUpdateEvent struct {
	Status ServerStatus `json:"status"`
}

const (
	EventSendMessage   = "send_message"
	EventStatusUpdate  = "status_update"
	EventModlistUpdate = "modlist_update"
	EventLogAppend     = "log_append"
	EventLogSnapshot   = "log_snapshot"

	Starting   ServerStatus = "starting"
	Online     ServerStatus = "online"
	Offline    ServerStatus = "offline"
	Restarting ServerStatus = "restarting"
	Stopping   ServerStatus = "stopping"
)
