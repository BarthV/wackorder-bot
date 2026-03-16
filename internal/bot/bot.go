package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/barthv/wackorder-bot/internal/store"
)

// Bot manages the Discord session and command lifecycle.
type Bot struct {
	session        *discordgo.Session
	corpID         string
	store          store.Repository
	registeredCmds []*discordgo.ApplicationCommand
	pruneCancel    context.CancelFunc
}

// New creates a Bot but does not open the connection yet.
func New(token, corpID string, repo store.Repository) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsNone
	return &Bot{session: s, corpID: corpID, store: repo}, nil
}

// Start opens the Discord connection and registers slash commands.
func (b *Bot) Start() error {
	h := &handler{store: b.store}
	b.session.AddHandler(h.onInteraction)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	for _, cmd := range commands() {
		registered, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.corpID, cmd)
		if err != nil {
			return fmt.Errorf("register command %q: %w", cmd.Name, err)
		}
		b.registeredCmds = append(b.registeredCmds, registered)
		slog.Info("command registered", "name", cmd.Name)
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.pruneCancel = cancel
	go b.startPruner(ctx)

	slog.Info("wackorder bot is running", "corp_id", b.corpID)
	return nil
}

// Stop deregisters commands, stops the pruner, and closes the Discord session.
func (b *Bot) Stop() error {
	if b.pruneCancel != nil {
		b.pruneCancel()
	}
	for _, cmd := range b.registeredCmds {
		if err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.corpID, cmd.ID); err != nil {
			slog.Warn("failed to delete command", "name", cmd.Name, "err", err)
		}
	}
	return b.session.Close()
}
