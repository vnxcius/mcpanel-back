package events

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

func isMinecraftOnline(addr string) bool {
	slog.Info("Checking if Minecraft server is online", "addr", addr)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func NewClient(conn *websocket.Conn, m *WSManager) *Client {
	return &Client{
		connection: conn,
		manager:    m,
		egress:     make(chan Event, 8),
	}
}

func (c *Client) send(evt Event) {
	select {
	case c.egress <- evt:
	default:
		go c.manager.RemoveClient(c)
	}
}

func (c *Client) ReadMessages() {
	defer func() {
		// Clean up connection
		c.manager.RemoveClient(c)
	}()

	for {
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Client read error", "error", err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			slog.Error("Error unmarshalling message", "error", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			slog.Error("Error handleling message", "error", err)
		}

		slog.Info("Received message", "type", messageType, "payload", string(payload))
	}
}

func (c *Client) WriteMessages() {
	defer func() {
		slog.Info("Cleaning up connection in WriteMessages")
		c.manager.RemoveClient(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					slog.Error("WS connection closed", "error", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				slog.Error("Error marshalling message", "error", err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Error("Error sending message", "error", err)
				return
			}

			slog.Info("Sent message", "type", message.Type, "payload", string(data))
		}
	}
}
