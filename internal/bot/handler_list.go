package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
	"github.com/bwmarrin/discordgo"
)

// handleOrders processes the /order-list slash command.
// All options (mode, component, older-than) are composable and applied as cumulative filters.
func (h *handler) handleOrders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	// Fetch base set from mode (default: pending).
	mode := "pending"
	if modeOpt, ok := opts["mode"]; ok {
		mode = modeOpt.StringValue()
	}

	var orders []model.Order
	var baseTitle string
	var err error

	switch mode {
	case "all":
		orders, err = h.store.ListAll(context.Background())
		baseTitle = "Toutes les commandes"
	case "self":
		caller, ok := requireCallerID(s, i)
		if !ok {
			return
		}
		orders, err = h.store.ListByCreator(context.Background(), caller)
		baseTitle = "Mes commandes"
	case "booked":
		caller, ok := requireCallerID(s, i)
		if !ok {
			return
		}
		orders, err = h.store.ListReadyByUpdater(context.Background(), caller)
		baseTitle = "Les commandes que je gère"
	default: // "pending"
		orders, err = h.store.ListPending(context.Background())
		baseTitle = "Commandes en attente"
	}

	if err != nil {
		slog.Error("failed to list orders", "mode", mode, "err", err)
		respond(s, i, errEmbed("Impossible de lister les commandes. Réessaie plus tard."))
		return
	}

	var filterLabels []string

	// Filter by component (case-insensitive substring match).
	if compOpt, ok := opts["component"]; ok {
		name := strings.TrimSpace(compOpt.StringValue())
		if name == "" {
			respond(s, i, errEmbed("Le nom de la ressource ne peut pas être vide."))
			return
		}
		nameLower := strings.ToLower(name)
		var filtered []model.Order
		for _, o := range orders {
			if strings.Contains(strings.ToLower(o.Component), nameLower) {
				filtered = append(filtered, o)
			}
		}
		orders = filtered
		filterLabels = append(filterLabels, name)
	}

	// Filter by older-than (orders created before now - duration).
	if olderOpt, ok := opts["older-than"]; ok {
		raw := strings.TrimSpace(olderOpt.StringValue())
		days, err := parseDurationDays(raw)
		if err != nil {
			respond(s, i, errEmbed("Format de durée invalide. Exemples : 7, 7d, 2w, 1mo."))
			return
		}
		before := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		var filtered []model.Order
		for _, o := range orders {
			if o.CreatedAt.Before(before) {
				filtered = append(filtered, o)
			}
		}
		orders = filtered
		filterLabels = append(filterLabels, ">"+raw)
	}

	title := baseTitle
	if len(filterLabels) > 0 {
		title += " — " + strings.Join(filterLabels, ", ")
	}
	respondEmbeds(s, i, orderListEmbeds(orders, title), discordgo.MessageFlagsEphemeral)
}

// parseDurationDays parses a human duration string and returns the equivalent number of days.
// Supported formats: plain integer (days), Nd (days), Nw (weeks), Nmo (months, 30 days each).
// Examples: "7", "7d", "2w", "1mo".
func parseDurationDays(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	suffixes := map[string]int{"d": 1, "w": 7, "mo": 30}
	for suffix, multiplier := range suffixes {
		if strings.HasSuffix(s, suffix) {
			n, err := strconv.Atoi(strings.TrimSuffix(s, suffix))
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("invalid duration %q", s)
			}
			return n * multiplier, nil
		}
	}

	// Plain integer = days.
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return n, nil
}
