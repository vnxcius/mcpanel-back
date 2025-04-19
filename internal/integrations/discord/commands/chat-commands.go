package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/config"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/helpers"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/server"
)

type commandHandler func(s *discordgo.Session, e *discordgo.MessageCreate, args []string)

var chatCommandHandlers = map[string]commandHandler{
	"ping":   ping,
	"status": status,
}

func ping(s *discordgo.Session, e *discordgo.MessageCreate, args []string) {
	_, err := s.ChannelMessageSend(e.ChannelID, "Pong!")
	if err != nil {
		log.Printf("CommandHandler Error: Failed sending 'Pong!' response for channel %s: %v",
			e.ChannelID,
			err,
		)
	}
}

func status(s *discordgo.Session, e *discordgo.MessageCreate, args []string) {
	currentStatus := server.GetCurrentStatusThreadSafe()
	emoji := helpers.GetStatusEmoji(currentStatus)
	timestamp := helpers.GetTimeNow()

	message := fmt.Sprintf("`%s SERVER IS %s %s`",
		timestamp,
		config.TitleCaser.String(currentStatus),
		emoji,
	)

	sentMessage, err := s.ChannelMessageSend(e.ChannelID, message)
	if err != nil {
		log.Printf("CommandHandler Error: Failed sending status response for channel %s: %v",
			e.ChannelID,
			err,
		)
	}

	helpers.DeleteSentMessage(s, e.ChannelID, e.Message.ID, sentMessage.ID)
}

func unknownCommand(s *discordgo.Session, e *discordgo.MessageCreate, cmd, prefix string) {
	message := fmt.Sprintf("Comando desconhecido `%s%s`.", prefix, cmd)
	_, err := s.ChannelMessageSend(e.ChannelID, message)
	if err != nil {
		log.Printf(
			"CommandHandler Error: Failed sending unknown command response for channel %s: %v",
			e.ChannelID,
			err,
		)
	}
}

func ChatCommands(s *discordgo.Session, e *discordgo.MessageCreate) {
	if e.Author.ID == config.BotID {
		return
	}

	prefix := config.GetConfig().BotPrefix
	if !strings.HasPrefix(e.Content, prefix) {
		return
	}

	contentWithoutPrefix := strings.TrimPrefix(e.Content, prefix)
	args := strings.Fields(contentWithoutPrefix)
	if len(args) == 0 {
		return // Only prefix was sent
	}

	cmd := strings.ToLower(args[0])
	commandArgs := args[1:]

	if handler, found := chatCommandHandlers[cmd]; found {
		handler(s, e, commandArgs) // Execute the found handler
	} else {
		unknownCommand(s, e, cmd, prefix) // Command not in map
	}
}
