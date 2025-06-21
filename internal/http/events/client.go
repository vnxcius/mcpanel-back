package events

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10 // 90% of pongWait
)

func NewClient(conn *websocket.Conn, m *WSManager) *Client {
	return &Client{
		connection: conn,
		manager:    m,
		egress:     make(chan Event, 500),
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

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error("Error setting read deadline", "error", err)
		return
	}

	c.connection.SetReadLimit(512)
	c.connection.SetPongHandler(c.pongHandler)

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
		c.manager.RemoveClient(c)
	}()

	ticker := time.NewTicker(pingInterval)
	slog.Debug("Starting write loop", "ip", c.connection.RemoteAddr().String())
	for {
		select {
		case message, ok := <-c.egress:
			slog.Debug("Sending message", "type", message.Type)
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

			slog.Info("Sent message", "type", message.Type)
		case <-ticker.C:
			slog.Info("Ping sent to client", "ip", c.connection.RemoteAddr().String())
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				slog.Error("Error sending ping", "error", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMessage string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
