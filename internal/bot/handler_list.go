package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleOrders processes the /order-list slash command.
func (h *handler) handleOrders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	// /order-list component:<name> — search across all users, ignores view/since
	if compOpt, ok := opts["component"]; ok {
		name := strings.TrimSpace(compOpt.StringValue())
		if name == "" {
			respond(s, i, errEmbed("Le nom de la ressource ne peut pas être vide."))
			return
		}
		orders, err := h.store.SearchByComponent(context.Background(), name)
		if err != nil {
			slog.Error("SearchByComponent failed", "component", name, "err", err)
			respond(s, i, errEmbed("Impossible de rechercher les commandes. Réessaie plus tard."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, fmt.Sprintf("Commandes correspondant à \"%s\"", name)), discordgo.MessageFlagsEphemeral)
		return
	}

	// /order-list since:<date> — filter by creation date, all users.
	// Dates are interpreted as midnight UTC.
	if sinceOpt, ok := opts["since"]; ok {
		raw := strings.TrimSpace(sinceOpt.StringValue())
		since, err := parseDate(raw)
		if err != nil {
			respond(s, i, errEmbed("Format de date invalide. Utilise YYYY-MM-DD ou YYYY-MM-DDTHH:MM:SSZ."))
			return
		}
		orders, err := h.store.ListSince(context.Background(), since)
		if err != nil {
			slog.Error("ListSince failed", "since", raw, "err", err)
			respond(s, i, errEmbed("Impossible de lister les commandes. Réessaie plus tard."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Commandes depuis le "+raw), discordgo.MessageFlagsEphemeral)
		return
	}

	// /order-list [view:<self|pending|all>]
	// Default value is "pending"
	view := "pending"
	if viewOpt, ok := opts["view"]; ok {
		view = viewOpt.StringValue()
	}

	switch view {
	case "pending":
		orders, err := h.store.ListPending(context.Background())
		if err != nil {
			slog.Error("ListPending failed", "err", err)
			respond(s, i, errEmbed("Impossible de lister les commandes. Réessaie plus tard."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Commandes en attente"), discordgo.MessageFlagsEphemeral)

	case "all":
		orders, err := h.store.ListAll(context.Background())
		if err != nil {
			slog.Error("ListAll failed", "err", err)
			respond(s, i, errEmbed("Impossible de lister les commandes. Réessaie plus tard."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Toutes les commandes"), discordgo.MessageFlagsEphemeral)

	default: // "pending"
		caller, ok := requireCallerID(s, i)
		if !ok {
			return
		}
		orders, err := h.store.ListByCreator(context.Background(), caller)
		if err != nil {
			slog.Error("ListByCreator failed", "caller", caller, "err", err)
			respond(s, i, errEmbed("Impossible de lister les commandes. Réessaie plus tard."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Mes commandes"), discordgo.MessageFlagsEphemeral)
	}
}

// parseDate parses a date string in RFC3339 or YYYY-MM-DD (interpreted as midnight UTC) format.
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q: expected YYYY-MM-DD or RFC3339 (e.g. 2026-01-15 or 2026-01-15T18:00:00Z)", s)
}
