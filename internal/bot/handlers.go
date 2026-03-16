package bot

import (
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/barthv/wackorder-bot/internal/store"
)

// handler holds shared dependencies for all command handlers.
type handler struct {
	store store.Repository
}

// onInteraction is the single entry-point registered with discordgo.
func (h *handler) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommandAutocomplete:
		switch i.ApplicationCommandData().Name {
		case "order", "order-list":
			h.handleOrderAutocomplete(s, i)
		case "order-update":
			h.handleOrderUpdateAutocomplete(s, i)
		}

	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "order":
			h.handleOrder(s, i)
		case "order-list":
			h.handleOrders(s, i)
		case "order-update":
			h.handleOrderUpdate(s, i)
		case "order-cancel":
			h.handleOrderCancel(s, i)
		case "order-stats":
			h.handleOrderStats(s, i)
		case "order-help":
			h.handleOrderHelp(s, i)
		}

	case discordgo.InteractionMessageComponent:
		cid := i.MessageComponentData().CustomID
		switch {
		case strings.HasPrefix(cid, statsFilterCustomID):
			h.handleStatsFilter(s, i)
		case strings.HasPrefix(cid, statusSelectCustomID):
			h.handleStatusSelect(s, i)
		}

	case discordgo.InteractionModalSubmit:
		if customID(i) == "order_modal" {
			h.handleOrderModalSubmit(s, i)
		}
	}
}

// customID extracts the base custom ID (before any ":" separator) from a modal submit.
func customID(i *discordgo.InteractionCreate) string {
	id := i.ModalSubmitData().CustomID
	for j, c := range id {
		if c == ':' {
			return id[:j]
		}
	}
	return id
}

// respond sends an interaction response. Errors are logged but not returned.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.InteractionResponseData) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
	if err != nil {
		// Log but don't panic; the interaction window may have expired (3s limit).
		slog.Debug("interaction respond failed", "err", err)
	}
}

// respondEmbeds sends one or more embeds as a single interaction response.
// Only the first embed is sent via the initial response; extras are follow-ups.
// Pass discordgo.MessageFlagsEphemeral to make the response visible only to the invoker.
func respondEmbeds(s *discordgo.Session, i *discordgo.InteractionCreate, embeds []*discordgo.MessageEmbed, flags ...discordgo.MessageFlags) {
	if len(embeds) == 0 {
		return
	}
	var f discordgo.MessageFlags
	for _, fl := range flags {
		f |= fl
	}
	respond(s, i, &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embeds[0]}, Flags: f})
	for _, e := range embeds[1:] {
		if _, err := s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{e},
			Flags:  f,
		}); err != nil {
			slog.Debug("follow-up message failed", "err", err)
		}
	}
}

// requireCallerID returns the caller's Discord user ID and responds with an
// error embed (returning false) if the identity cannot be determined.
func requireCallerID(s *discordgo.Session, i *discordgo.InteractionCreate) (string, bool) {
	id := callerID(i)
	if id == "" {
		respond(s, i, errEmbed("Impossible de déterminer ton identité. Utilise cette commande dans un serveur."))
		return "", false
	}
	return id, true
}

// callerID returns the Discord user ID of the interaction invoker, or "" if unknown.
func callerID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// callerName returns the display name of the interaction invoker.
func callerName(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		if i.Member.Nick != "" {
			return i.Member.Nick
		}
		return i.Member.User.Username
	}
	if i.User != nil {
		return i.User.Username
	}
	return "unknown"
}
