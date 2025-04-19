package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/config"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/helpers"
)

type sseMessage struct {
	Status string `json:"status"`
}

var (
	notificationChannelID string
	statusMutex           sync.RWMutex
	currentServerStatus   string = "unknown"
)

func establishSSEConnection(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSE endpoint %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("SSE connection failed: Server responded with %s", resp.Status)
	}

	return resp, nil
}

func processSSEStream(s *discordgo.Session, resp *http.Response) {
	defer resp.Body.Close()
	log.Println("SSE connection established successfully. Reading stream...")
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading from SSE stream (connection closed?): %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			jsonData := strings.TrimPrefix(line, "data:")
			jsonData = strings.TrimSpace(jsonData)

			if jsonData == "" {
				log.Println("Received empty data line from SSE.")
				continue
			}

			var msg sseMessage
			err := json.Unmarshal([]byte(jsonData), &msg)
			if err != nil {
				log.Printf("Failed to decode SSE JSON: %v. Raw Data: '%s'", err, jsonData)
				continue
			}

			handleStatusUpdate(s, msg.Status)
		}
	}
}

func ConnectToSSE(s *discordgo.Session) {
	if s == nil {
		log.Println("CRITICAL ERROR: ConnectToSSE started with a nil Discord session. Aborting SSE connection.")
		return
	}

	cfg := config.GetConfig()
	sseURL := cfg.SSEURL
	notificationChannelID = cfg.NotificationChannelID
	log.Println("Notification channel ID:", notificationChannelID)

	log.Printf("Attempting to connect to SSE endpoint: %s", sseURL)

	for {
		resp, err := establishSSEConnection(sseURL)
		if err != nil {
			log.Printf("SSE connection attempt failed: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		processSSEStream(s, resp)

		log.Println("SSE connection lost. Waiting 5 seconds before reconnecting...")
		time.Sleep(5 * time.Second)
	}
}

func handleStatusUpdate(s *discordgo.Session, newStatus string) {
	if s == nil {
		log.Println("CRITICAL ERROR: ConnectToSSE started with a nil Discord session. Aborting SSE connection.")
		return
	}
	statusMutex.Lock()
	defer statusMutex.Unlock()

	oldStatus := currentServerStatus
	if oldStatus != newStatus {
		currentServerStatus = newStatus
		timestamp := helpers.GetTimeNow()
		log.Printf("Server status updated: %s -> %s", oldStatus, newStatus)

		emoji := helpers.GetStatusEmoji(newStatus)

		if notificationChannelID != "" && oldStatus != "unknown" {
			message := fmt.Sprintf("`%s UPDATE: SERVER %s %s`",
				timestamp,
				config.TitleCaser.String(newStatus),
				emoji,
			)
			_, err := s.ChannelMessageSend(notificationChannelID, message)
			if err != nil {
				log.Printf("Failed to send SSE notification to channel: %v", err)
			}
		}
	}
}

func GetCurrentStatusThreadSafe() string {
	statusMutex.RLock()
	defer statusMutex.RUnlock()
	return currentServerStatus
}
