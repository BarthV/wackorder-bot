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
	store          store.Repository
	registeredCmds []*discordgo.ApplicationCommand
	bgCancel       context.CancelFunc
}

// New creates a Bot but does not open the connection yet.
func New(token, corpID, logChannelID, recapChannelID string, adminRoleIDs []string, repo store.Repository) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsNone
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
	b.registeredCmds = registered
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

// Stop deregisters commands, stops the pruner, and closes the Discord session.
func (b *Bot) Stop() error {
	if b.bgCancel != nil {
		b.bgCancel()
	}
	for _, cmd := range b.registeredCmds {
		if err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.corpID, cmd.ID); err != nil {
			slog.Warn("failed to delete command", "name", cmd.Name, "err", err)
		}
	}
	return b.session.Close()
}
