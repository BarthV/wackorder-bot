package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/barthv/wackorder-bot/internal/store"
	"github.com/bwmarrin/discordgo"
)

// Bot manages the Discord session and command lifecycle.
type Bot struct {
	session        *discordgo.Session
	corpID         string
	logChannelID   string
	recapChannelID string
	adminRoleIDs   []string
	store    store.Repository
	bgCancel context.CancelFunc
}

// New creates a Bot but does not open the connection yet.
func New(token, corpID, logChannelID, recapChannelID string, adminRoleIDs []string, repo store.Repository, debug bool) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsNone
	if debug {
		s.LogLevel = discordgo.LogDebug
	}
	return &Bot{session: s, corpID: corpID, logChannelID: logChannelID, recapChannelID: recapChannelID, adminRoleIDs: adminRoleIDs, store: repo}, nil
}

// Start opens the Discord connection and registers slash commands.
func (b *Bot) Start() error {
	h := &handler{store: b.store, logChannelID: b.logChannelID, adminRoleIDs: b.adminRoleIDs}
	b.session.AddHandler(h.onInteraction)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	registered, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, b.corpID, commands())
	if err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	for _, cmd := range registered {
		slog.Info("command registered", "name", cmd.Name)
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.bgCancel = cancel
	go b.startPruner(ctx)
	go b.startRecap(ctx)

	guildName := b.corpID
	if guild, err := b.session.Guild(b.corpID); err == nil && guild.Name != "" {
		guildName = guild.Name
	} else {
		guildName = "-"
	}
	slog.Info("wackorder bot is running", "corp", fmt.Sprintf("%s (%s)", guildName, b.corpID))
	return nil
}

// Stop stops background tasks and closes the Discord session.
// Commands are not deregistered: BulkOverwrite on next Start() reconciles them.
func (b *Bot) Stop() error {
	if b.bgCancel != nil {
		b.bgCancel()
	}
	return b.session.Close()
}
