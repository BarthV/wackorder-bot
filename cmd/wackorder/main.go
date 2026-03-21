package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/barthv/wackorder-bot/internal/bot"
	"github.com/barthv/wackorder-bot/internal/config"
	"github.com/barthv/wackorder-bot/internal/db"
	"github.com/barthv/wackorder-bot/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	setupLogger(cfg.LogLevel, cfg.LogFormat)

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "path", cfg.DBPath, "err", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.InitSchema(database); err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database ready", "path", cfg.DBPath)

	repo := store.New(database)

	b, err := bot.New(cfg.DiscordToken, cfg.CorpID, cfg.LogChannelID, cfg.RecapChannelID, cfg.AdminRoleIDs, repo, cfg.Debug)
	if err != nil {
		slog.Error("failed to create bot", "err", err)
		os.Exit(1)
	}

	if err := b.Start(); err != nil {
		slog.Error("failed to start bot", "err", err)
		os.Exit(1)
	}

	// Block until SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	if err := b.Stop(); err != nil {
		slog.Warn("error during bot shutdown", "err", err)
	}
	slog.Info("goodbye")
}

func setupLogger(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
