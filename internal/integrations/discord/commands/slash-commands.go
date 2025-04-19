package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/config"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/helpers"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/server"
)

var registeredCommands []*discordgo.ApplicationCommand

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "jockey-de-galinha",
			Description: "O Nether?? 💯🔥",
		},
		{
			Name:        "status",
			Description: "Veja o status atual do servidor.",
		},
		{
			Name:        "subcommands",
			Description: "Subcommands and command groups example",
			Options: []*discordgo.ApplicationCommandOption{
				// When a command has subcommands/subcommand groups
				// It must not have top-level options, they aren't accesible in the UI
				// in this case (at least not yet), so if a command has
				// subcommands/subcommand any groups registering top-level options
				// will cause the registration of the command to fail
				{
					Name:        "subcommand-group",
					Description: "Subcommands group",
					Options: []*discordgo.ApplicationCommandOption{
						// Also, subcommand groups aren't capable of
						// containing options, by the name of them, you can see
						// they can only contain subcommands
						{
							Name:        "nested-subcommand",
							Description: "Nested subcommand",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
				},
				// Also, you can create both subcommand groups and subcommands
				// in the command at the same time. But, there's some limits to
				// nesting, count of subcommands (top level and nested) and options.
				// Read the intro of slash-commands docs on Discord dev portal
				// to get more information
				{
					Name:        "subcommand",
					Description: "Top-level subcommand",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"jockey-de-galinha": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "JOCKEY DE GALINHA 🐔🐔💯💯💯🔥🔥🔥",
				},
			})
		},
		"status": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var emoji string
			status := server.GetCurrentStatusThreadSafe()
			switch status {
			case "online":
				emoji = "🟢"
			case "starting":
				emoji = "🔵"
			case "restarting", "stopping":
				emoji = "🟡"
			case "offline":
				emoji = "🔴"
			default:
				emoji = "❌"
			}

			timestamp := helpers.GetTimeNow()
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
					Content: fmt.Sprintf("`%s SERVER %s %s`",
						timestamp,
						config.TitleCaser.String(status),
						emoji,
					),
				},
			})

			time.AfterFunc(5*time.Second, func() {
				s.InteractionResponseDelete(i.Interaction)
			})
		},
		"subcommands": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			content := ""

			// As you can see, names of subcommands (nested, top-level)
			// and subcommand groups are provided through the arguments.
			switch options[0].Name {
			case "subcommand":
				content = "The top-level subcommand is executed. Now try to execute the nested one."
			case "subcommand-group":
				options = options[0].Options
				switch options[0].Name {
				case "nested-subcommand":
					content = "Nice, now you know how to execute nested commands too"
				default:
					content = "Oops, something went wrong.\n" +
						"Hol' up, you aren't supposed to see this message."
				}
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}
)

func RegisterSlashCommands(s *discordgo.Session) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	if config.BotID == "" {
		log.Println("[WARNING] BotID is empty, cannot register slash commands yet.")
		return
	}

	log.Println("Adding commands...")
	registeredCommands = make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		log.Printf("Adding command: %s\n", v.Name)
		cmd, err := s.ApplicationCommandCreate(config.BotID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	log.Println("Commands added successfully.")
}

func RemoveSlashCommands(s *discordgo.Session) {
	// remove commands on shutdown if true
	removeCommands := true

	if removeCommands {
		log.Println("Removing commands...")
		registeredCommands, err := s.ApplicationCommands(config.BotID, "")
		if err != nil {
			log.Fatalf("Could not fetch registered commands: %v", err)
		}

		for _, v := range registeredCommands {
			log.Printf("Removing command: %s\n", v.Name)
			err := s.ApplicationCommandDelete(config.BotID, "", v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}
}
