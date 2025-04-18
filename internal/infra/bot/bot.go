package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/config"
)

var BotID string

func StartBot() error {
	bot, err := discordgo.New("Bot " + config.GetConfig().BotToken)
	if err != nil {
		return err
	}

	u, err := bot.User("@me")
	if err != nil {
		return err
	}

	BotID = u.ID

	bot.AddHandler(messageHandler)

	err = bot.Open()
	if err != nil {
		return err
	}

	log.Println("Bot is now running.  Press CTRL-C to exit.")
	return nil
}

func messageHandler(s *discordgo.Session, e *discordgo.MessageCreate) {
	if e.Author.ID == BotID {
		return
	}

	prefix := "!"
	if strings.HasPrefix(e.Content, prefix) {
		args := strings.Fields(e.Content)[strings.Index(e.Content, prefix):]
		cmd := args[0][len(prefix):]

		switch cmd {
		case "ping":
			_, err := s.ChannelMessageSend(e.ChannelID, "Pong!")
			if err != nil {
				log.Println("Failed sending Pong response:", err)
			}
		default:
			_, err := s.ChannelMessageSend(e.ChannelID, fmt.Sprintf("Unknown command %q.", cmd))
			if err != nil {
				log.Println("Failed sending Unknown Command response:", err)
			}
		}
	}
}
