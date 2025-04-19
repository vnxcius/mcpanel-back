package config

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	config *Config
	once   sync.Once
)

type Config struct {
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
	Token       string `mapstructure:"TOKEN"`

	BotToken              string `mapstructure:"BOT_TOKEN"`
	NotificationChannelID string `mapstructure:"NOTIFICATION_CHANNEL_ID"`
	BotPrefix             string `mapstructure:"BOT_PREFIX"`

	PostgresDSN       string        `mapstructure:"POSTGRES_DSN"`
	DBName            string        `mapstructure:"DB_NAME"`
	DBUser            string        `mapstructure:"DB_USER"`
	DBPassword        string        `mapstructure:"DB_PASSWORD"`
	DBHost            string        `mapstructure:"DB_HOST"`
	DBPort            string        `mapstructure:"DB_PORT"`
	DBMaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBMaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
	DBLogMode         bool          `mapstructure:"DB_LOG_MODE"`
}

func GetConfig() *Config {
	once.Do(func() {
		viper.SetDefault("PORT", "4000")
		viper.SetDefault("ENVIRONMENT", "development")
		viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
		viper.SetDefault("DB_MAX_OPEN_CONNS", 100)
		viper.SetDefault("DB_CONN_MAX_LIFETIME", "1h")
		viper.SetDefault("DB_LOG_MODE", true)

		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Fatalf("Fatal error config file: %s \n", err)
			} else {
				log.Println("[WARNING]: .env config file not found, relying on defaults and system ENV variables.")
			}
		}

		if viper.GetString("BOT_TOKEN") == "" {
			log.Fatal("Bot configuration error: Missing bot token (BOT_TOKEN env variable)")
		}
		if viper.GetString("NOTIFICATION_CHANNEL_ID") == "" {
			log.Fatal("Bot configuration error: Missing notification channel ID (NOTIFICATION_CHANNEL_ID env variable)")
		}

		if err := viper.Unmarshal(&config); err != nil {
			log.Fatalf("Error unmarshalling config, %s", err)
		}

		lifetimeStr := viper.GetString("DB_CONN_MAX_LIFETIME")
		parsedLifetime, err := time.ParseDuration(lifetimeStr)
		if err != nil {
			log.Printf(
				"Warning: Invalid DB_CONN_MAX_LIFETIME format '%s', using default 1h. Error: %v\n",
				lifetimeStr,
				err,
			)
			parsedLifetime = time.Hour
		}
		config.DBConnMaxLifetime = parsedLifetime
	})

	return config
}
