package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleOrders processes the /orders slash command.
func (h *handler) handleOrders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	// /orders component:<name> — search across all users, ignores view/since
	if compOpt, ok := opts["component"]; ok {
		name := strings.TrimSpace(compOpt.StringValue())
		if name == "" {
			respond(s, i, errEmbed("Component name cannot be empty."))
			return
		}
		orders, err := h.store.SearchByComponent(context.Background(), name)
		if err != nil {
			slog.Error("SearchByComponent failed", "component", name, "err", err)
			respond(s, i, errEmbed("Failed to search orders. Please try again later."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, fmt.Sprintf("Orders matching \"%s\"", name)), discordgo.MessageFlagsEphemeral)
		return
	}

	// /orders since:<date> — filter by creation date, all users.
	// Dates are interpreted as midnight UTC.
	if sinceOpt, ok := opts["since"]; ok {
		raw := strings.TrimSpace(sinceOpt.StringValue())
		since, err := parseDate(raw)
		if err != nil {
			respond(s, i, errEmbed("Invalid date format. Use YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ."))
			return
		}
		orders, err := h.store.ListSince(context.Background(), since)
		if err != nil {
			slog.Error("ListSince failed", "since", raw, "err", err)
			respond(s, i, errEmbed("Failed to list orders. Please try again later."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Orders since "+raw), discordgo.MessageFlagsEphemeral)
		return
	}

	// /orders [view:<mine|pending|all>]
	view := "mine"
	if viewOpt, ok := opts["view"]; ok {
		view = viewOpt.StringValue()
	}

	switch view {
	case "pending":
		orders, err := h.store.ListPending(context.Background())
		if err != nil {
			slog.Error("ListPending failed", "err", err)
			respond(s, i, errEmbed("Failed to list orders. Please try again later."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Pending orders"), discordgo.MessageFlagsEphemeral)

	case "all":
		orders, err := h.store.ListAll(context.Background())
		if err != nil {
			slog.Error("ListAll failed", "err", err)
			respond(s, i, errEmbed("Failed to list orders. Please try again later."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "All orders"), discordgo.MessageFlagsEphemeral)

	default: // "mine"
		caller, ok := requireCallerID(s, i)
		if !ok {
			return
		}
		orders, err := h.store.ListByCreator(context.Background(), caller)
		if err != nil {
			slog.Error("ListByCreator failed", "caller", caller, "err", err)
			respond(s, i, errEmbed("Failed to list orders. Please try again later."))
			return
		}
		respondEmbeds(s, i, orderListEmbeds(orders, "Your orders"), discordgo.MessageFlagsEphemeral)
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
