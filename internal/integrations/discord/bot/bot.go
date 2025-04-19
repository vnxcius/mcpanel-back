package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/commands"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/config"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/server"
)

func readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %s", event.User.ID)
	config.BotID = event.User.ID

	go commands.RegisterSlashCommands(s)
}

func StartBot() (*discordgo.Session, error) {
	s, err := config.NewDiscordSession()
	if err != nil {
		log.Fatalf("Error creating Discord session, %s", err)
	}

	s.AddHandler(readyHandler)
	s.AddHandler(commands.ChatCommands)

	s.Identify.Intents = discordgo.IntentsGuildMessages

	err = s.Open()
	if err != nil {
		s = nil
		return nil, fmt.Errorf("error opening websocket connection to Discord: %w", err)
	}

	log.Println("Discord session opened successfully. Starting SSE connection...")
	go server.ConnectToSSE(s)
	return s, nil
}
