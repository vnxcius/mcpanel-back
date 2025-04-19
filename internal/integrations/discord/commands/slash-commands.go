package commands

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/vnxcius/sss-backend/internal/config"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/helpers"
	"github.com/vnxcius/sss-backend/internal/integrations/discord/server"
)

var (
	authorizedUserIDs  map[string]bool
	registeredCommands []*discordgo.ApplicationCommand
)

func init() {
	authorizedUserIDs = make(map[string]bool)

	cfg := config.GetConfig()
	if cfg == nil || cfg.AuthorizedUserIDs == "" {
		fmt.Println("Warning: AuthorizedUserIDs not found in config or is empty. No users will be authorized for protected commands.")
		return
	}

	idSlice := strings.Split(cfg.AuthorizedUserIDs, ",")

	loadedCount := 0
	// Correctly iterate over the SLICE VALUES
	for _, idStr := range idSlice {
		trimmedID := strings.TrimSpace(idStr)
		if trimmedID != "" {
			authorizedUserIDs[trimmedID] = true
			loadedCount++
		}
	}

	if loadedCount > 0 {
		fmt.Printf("Successfully loaded %d authorized Discord User ID(s) from config.\n", loadedCount)
	} else {
		fmt.Println("Warning: Config AuthorizedUserIDs was set but contained no valid IDs after parsing.")
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "help",
			Description: "Veja a lista de comandos disponíveis.",
		},
		{
			Name:        "jockey-de-galinha",
			Description: "CHICKENJOCKEY!",
		},
		{
			Name:        "o-nether",
			Description: "THENETHER!",
		},
		{
			Name:        "pedra-e-aço",
			Description: "FLINTANDSTEEL!",
		},
		{
			Name:        "elitros",
			Description: "QUITALESSESELITROS!",
		},
		{
			Name:        "status",
			Description: "Veja o status atual do servidor.",
		},
		{
			Name:        "server",
			Description: "Alterar o status do servidor.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "start",
					Description: "Iniciar o servidor.",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "stop",
					Description: "Parar o servidor.",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "restart",
					Description: "Reiniciar o servidor.",
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var embedFields []*discordgo.MessageEmbedField
			for _, cmd := range commands {
				embedFields = append(embedFields, &discordgo.MessageEmbedField{
					Name:   fmt.Sprintf("`/%s`", cmd.Name),
					Value:  cmd.Description,
					Inline: false,
				})
			}
			embed := &discordgo.MessageEmbed{
				Title:  "📋 Comandos disponíveis",
				Color:  0x4F545C,
				Fields: embedFields,
				Author: &discordgo.MessageEmbedAuthor{
					IconURL: "https://i.imgur.com/2D8ih6t.png",
					Name:    "Steve",
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: "https://i.imgur.com/WsSmkT3.png",
				},
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
					Flags:  discordgo.MessageFlagsEphemeral,
				},
			})
		},
		"jockey-de-galinha": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "JOCKEY DE GALINHA 🐔🐔💯🗣️🗣️💯🔥🙏🗣️🙏",
				},
			})
		},
		"o-nether": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "O NETHER 🗣️🔥🔥🔥🗣️🔥💯💯❌🧢",
				},
			})
		},
		"pedra-e-aço": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "PEDRA E AÇO 🗣️🗣️🪨🪨🪨💯💯🔥🙏🔥🙏",
				},
			})
		},
		"elitros": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "QUE TAL ESSES ÉLITROS❓❓🪽🪽🪽🗣️🗣️💯💯🔥🙏",
				},
			})
		},
		"status": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			status := server.GetCurrentStatusThreadSafe()
			emoji := helpers.GetStatusEmoji(status)
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
		"server": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			userID := i.Member.User.ID
			if !isUserAuthorized(userID) {
				sendEphemeralResponse(s, i, "❌ VOCÊ NÃO TEM PERMISSÃO PARA USAR `/server`.")
				return
			}

			options := i.ApplicationCommandData().Options
			if len(options) == 0 {
				sendEphemeralResponse(s, i, "`❌ Erro Interno: Comando precisa ser especificado.`")
				return
			}
			subCommand := options[0]

			cfg := config.GetConfig()
			apiBaseURL := cfg.APIUrl
			accessToken := cfg.Token

			var apiEndpoint string
			switch subCommand.Name {
			case "start":
				apiEndpoint = "/api/v1/start"
			case "stop":
				apiEndpoint = "/api/v1/stop"
			case "restart":
				apiEndpoint = "/api/v1/restart"
			default:
				sendEphemeralResponse(s, i, fmt.Sprintf("❌ Subcomando desconhecido: %s", subCommand.Name))
				return
			}
			apiURL := apiBaseURL + apiEndpoint

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				fmt.Printf("Error sending deferred response: %v\n", err)
				return
			}

			req, err := http.NewRequest("POST", apiURL, nil)
			if err != nil {
				editEphemeralResponse(s, i, fmt.Sprintf("`❌ Erro Interno: Falha ao criar requisição para API: %v`", err))
				return
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				editEphemeralResponse(s, i, fmt.Sprintf("`❌ Erro: Falha ao conectar à API (%s): %v`", apiURL, err))
				return
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyString := strings.TrimSpace(string(bodyBytes))

			var responseMsg string
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if bodyString != "" {
					responseMsg = fmt.Sprintf("`✅ Comando '%s' aceito pela API: %s`", subCommand.Name, bodyString)
				} else {
					responseMsg = fmt.Sprintf("`✅ Comando '%s' aceito pela API (Status: %s).`", subCommand.Name, resp.Status)
				}
			} else {
				responseMsg = fmt.Sprintf("`⚠️ API retornou erro para '%s': %s`", subCommand.Name, resp.Status)
				if bodyString != "" {
					responseMsg = fmt.Sprintf("`⚠️ Erro da API para '%s' (%s): %s`", subCommand.Name, resp.Status, bodyString)
				}
			}
			editEphemeralResponse(s, i, responseMsg)
		},
	}
)

func isUserAuthorized(userID string) bool {
	_, authorized := authorizedUserIDs[userID]
	return authorized
}

func sendEphemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error sending ephemeral response for interaction %s: %v\n", i.ID, err)
	}
}

func editEphemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		fmt.Printf("Error editing ephemeral response for interaction %s: %v\n", i.ID, err)
	}
}

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
