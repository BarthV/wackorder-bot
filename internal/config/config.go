package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DiscordToken string
	GuildID      string
	DBPath       string
	LogLevel     string
	LogFormat    string
}

// Load reads configuration from environment variables and returns a validated Config.
func Load() (*Config, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable is required")
	}

	guildID := os.Getenv("CORP_ID")
	if guildID == "" {
		return nil, fmt.Errorf("CORP_ID environment variable is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wackorder.db"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}

	return &Config{
		DiscordToken: token,
		GuildID:      guildID,
		DBPath:       dbPath,
		LogLevel:     logLevel,
		LogFormat:    logFormat,
	}, nil
}
