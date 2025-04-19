package helpers

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

func GetTimeNow() string {
	return time.Now().In(
		time.FixedZone("America/Sao_Paulo", -3*60*60),
	).Format("15:04:05")
}

func GetStatusEmoji(status string) string {
	switch status {
	case "online":
		return "🟢"
	case "starting":
		return "🔵"
	case "restarting", "stopping":
		return "🟡"
	case "offline":
		return "🔴"
	default:
		return "❌" // Unknown or error status
	}
}

func DeleteSentMessage(s *discordgo.Session, channelID, userMessage, botMessage string) {
	time.AfterFunc(5*time.Second, func() {
		s.ChannelMessageDelete(channelID, userMessage)
		s.ChannelMessageDelete(channelID, botMessage)
	})
}
