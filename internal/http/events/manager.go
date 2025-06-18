package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var (
	WebsocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}

	Manager  *WSManager
	logsPath string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logsPath = os.Getenv("LOGS_PATH")
}

func checkOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != os.Getenv("ALLOWED_ORIGINS") {
		return true
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
	c := NewClient(conn, m)

	m.Lock()
	m.clients[c] = true
	m.Unlock()

	go m.syncWithMinecraft()
	go c.WriteMessages()
	go c.ReadMessages()

	lines := make(chan string, 1000)
	go m.tailFile(logsPath, lines) // Start file tailing in a goroutine
	go func() {
		const maxLines = 500
		var (
			buf    []string
			offset int // next line to send
			mu     sync.Mutex
		)

		// Accumulate incoming lines
		go func() {
			for l := range lines {
				mu.Lock()
				buf = append(buf, l)

				if len(buf) > maxLines { // trim head
					drop := len(buf) - maxLines
					buf = buf[drop:]
					if offset >= drop {
						offset -= drop // adjust unread pointer
					} else {
						offset = 0
					}
				}
				mu.Unlock()
			}
		}()

		// Broadcast latest buffer every second
		tick := time.NewTicker(time.Second)
		defer tick.Stop()

		for range tick.C {
			mu.Lock()
			if offset < len(buf) {
				payload, _ := json.Marshal(struct {
					Lines []string `json:"lines"`
				}{Lines: buf[offset:]})

				m.broadcast(Event{Type: EventLogAppend, Payload: payload})
				offset = len(buf)
			}
			mu.Unlock()
		}
	}()

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

func (m *WSManager) getLastLogLines(n int) ([]string, error) {
	slog.Debug("Getting last log lines", "path", logsPath, "n", n)
	file, err := os.Open(filepath.Clean(logsPath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func (m *WSManager) tailFile(filePath string, lines chan<- string) {
	slog.Debug("Starting log tailing", "path", filePath)
	file, fi := openLog(filePath)
	defer file.Close()

	lastMod, offset := fi.ModTime(), fi.Size()
	file.Seek(0, io.SeekEnd)

	for {
		time.Sleep(time.Second)
		currentFi, err := os.Stat(filePath)
		if err != nil {
			log.Printf("file missing, waiting for recreation...")
			continue
		}

		if rotated(currentFi, lastMod, offset) {
			slog.Debug("Log file rotated, restarting tailing", "path", filePath)
			file.Close()
			file, fi = openLog(filePath)
			lastMod, offset = fi.ModTime(), 0
			continue
		}

		offset, lastMod = readNew(file, currentFi, offset, lines)
	}
}

func openLog(path string) (*os.File, os.FileInfo) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("failed to get file stats: %v", err)
	}
	return f, fi
}

func rotated(fi os.FileInfo, lastMod time.Time, offset int64) bool {
	return fi.ModTime().Before(lastMod) || fi.Size() < offset
}

func readNew(f *os.File, fi os.FileInfo, offset int64, out chan<- string) (int64, time.Time) {
	if fi.Size() <= offset {
		return offset, fi.ModTime()
	}
	f.Seek(offset, io.SeekStart)
	buf := make([]byte, fi.Size()-offset)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		log.Fatalf("failed to read file: %v", err)
	}
	lines := strings.SplitSeq(string(buf[:n]), "\n")
	for line := range lines {
		select {
		case out <- line:
		default: // drop if channel is full
			slog.Warn("Log channel full, dropping line")
		}
	}
	return fi.Size(), fi.ModTime()
}
