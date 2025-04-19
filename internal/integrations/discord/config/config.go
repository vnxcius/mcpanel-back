package config

import (
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	configBot *Config
	once      sync.Once
)

type Config struct {
	BotToken              string `mapstructure:"BOT_TOKEN"`
	NotificationChannelID string `mapstructure:"NOTIFICATION_CHANNEL_ID"`
	BotPrefix             string `mapstructure:"BOT_PREFIX"`
	SSEURL                string `mapstructure:"SSE_URL"`
}

var (
	BotID          string
	DiscordSession *discordgo.Session
	TitleCaser     cases.Caser = cases.Upper(language.English)
)

func NewDiscordSession() (*discordgo.Session, error) {
	session, err := discordgo.New("Bot " + GetConfig().BotToken)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func GetConfig() *Config {
	once.Do(func() {
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Fatalf("Fatal error config file: %s \n", err)
			}
		}

		if viper.GetString("SSE_URL") == "" {
			log.Fatal("Bot configuration error: Missing SSE URL (SSE_URL env variable)")
		}

		if viper.GetString("BOT_PREFIX") == "" {
			log.Fatal("Bot configuration error: Missing bot prefix (BOT_PREFIX env variable)")
		}

		if viper.GetString("BOT_TOKEN") == "" {
			log.Fatal("Bot configuration error: Missing bot token (BOT_TOKEN env variable)")
		}
		if viper.GetString("NOTIFICATION_CHANNEL_ID") == "" {
			log.Fatal("Bot configuration error: Missing notification channel ID (NOTIFICATION_CHANNEL_ID env variable)")
		}

		if err := viper.Unmarshal(&configBot); err != nil {
			log.Fatalf("Error unmarshalling config, %s", err)
		}

	})

	return configBot
}
