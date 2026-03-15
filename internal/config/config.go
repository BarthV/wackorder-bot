package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DiscordToken string
	CorpID       string
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

	corpID := os.Getenv("CORP_ID")
	if corpID == "" {
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
		CorpID:       corpID,
		DBPath:       dbPath,
		LogLevel:     logLevel,
		LogFormat:    logFormat,
	}, nil
}
