package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnxcius/sss-backend/internal/integrations/discord/bot"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/commands"
)

func main() {
	s, err := bot.StartBot()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Gracefully shutting down bot...")
	commands.RemoveSlashCommands(s)

	log.Println("Goodbye!")
}
