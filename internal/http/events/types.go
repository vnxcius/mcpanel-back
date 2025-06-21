package events

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type WSManager struct {
	clients ClientList
	sync.RWMutex
	handlers map[string]EventHandler

	currentStatus ServerStatus
}

type Client struct {
	connection *websocket.Conn
	manager    *WSManager
	ip         string

	// Buffered channel of outbound messages
	egress chan Event
}

type ClientList map[*Client]bool

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

type StatusUpdateEvent struct {
	Status ServerStatus `json:"status"`
}

type ServerStatus string

const (
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
