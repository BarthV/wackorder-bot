package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
	"github.com/bwmarrin/discordgo"
)

const (
	statsFilterCustomID = "stats_filter:"
	histDefaultCols     = 32
	histMaxHeight       = 6
	histMaxButtons      = 25 // Discord: 5 rows × 5 buttons
)

// handleOrderStats processes the /order-stats slash command.
// It shows summed pending quantities per resource, an ASCII histogram of creation dates,
// and per-resource filter buttons.
func (h *handler) handleOrderStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	maxCols := histDefaultCols
	if colsOpt, ok := opts["jours"]; ok {
		v := int(colsOpt.IntValue())
		if v >= 1 && v <= histDefaultCols {
			maxCols = v
		}
	}

	orders, err := h.store.ListPending(context.Background())
	if err != nil {
		slog.Error("ListPending failed for stats", "err", err)
		respond(s, i, errEmbed("Impossible de récupérer les commandes. Réessaie plus tard."))
		return
	}

	if len(orders) == 0 {
		respond(s, i, okEmbed("Aucune commande en attente."))
		return
	}

	// Sum quantities per component.
	totals := make(map[string]int)
	for _, o := range orders {
		totals[o.Component] += o.Quantity
	}

	// Sort components alphabetically for consistent display.
	components := make([]string, 0, len(totals))
	for comp := range totals {
		components = append(components, comp)
	}
	sort.Strings(components)

	// One embed field per resource.
	fields := make([]*discordgo.MessageEmbedField, len(components))
	for j, comp := range components {
		fields[j] = &discordgo.MessageEmbedField{
			Name:   comp,
			Value:  fmt.Sprintf("%d", totals[comp]),
			Inline: true,
		}
	}

	// Histogram in a code block (creation dates of pending orders).
	description := ""
	if hist := buildHistogram(orders, maxCols, histMaxHeight); hist != "" {
		description = "```\n" + hist + "\n```"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Commandes en attente",
		Description: description,
		Fields:      fields,
		Color:       colorOrdered,
	}

	// Build filter buttons (one per component, max histMaxButtons = 5×5).
	shown := components
	if len(shown) > histMaxButtons {
		shown = shown[:histMaxButtons]
	}
	var componentRows []discordgo.MessageComponent
	var currentRow discordgo.ActionsRow
	for _, comp := range shown {
		currentRow.Components = append(currentRow.Components, discordgo.Button{
			Label:    comp,
			Style:    discordgo.SecondaryButton,
			CustomID: statsFilterCustomID + comp,
		})
		if len(currentRow.Components) == 5 {
			componentRows = append(componentRows, currentRow)
			currentRow = discordgo.ActionsRow{}
		}
	}
	if len(currentRow.Components) > 0 {
		componentRows = append(componentRows, currentRow)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: componentRows,
		},
	}); err != nil {
		slog.Debug("failed to send order stats", "err", err)
	}
}

// handleStatsFilter responds to a resource filter button by listing pending orders for that component.
func (h *handler) handleStatsFilter(s *discordgo.Session, i *discordgo.InteractionCreate) {
	component := strings.TrimPrefix(i.MessageComponentData().CustomID, statsFilterCustomID)
	if component == "" {
		return
	}

	all, err := h.store.SearchByComponent(context.Background(), component)
	if err != nil {
		slog.Error("SearchByComponent failed for stats filter", "component", component, "err", err)
		respond(s, i, errEmbed("Impossible de récupérer les commandes. Réessaie plus tard."))
		return
	}

	var pending []model.Order
	for _, o := range all {
		if o.Status == model.StatusOrdered || o.Status == model.StatusReady {
			pending = append(pending, o)
		}
	}

	title := fmt.Sprintf("Commandes en attente — %s", component)
	respondEmbeds(s, i, orderListEmbeds(pending, title), discordgo.MessageFlagsEphemeral)
}

// buildHistogram returns the body of an ASCII histogram of order creation dates.
// Columns run left (oldest / catch-all) to right (today), one column per day.
// Height is scaled so the tallest bucket fills maxHeight rows.
func buildHistogram(orders []model.Order, maxCols, maxHeight int) string {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	buckets := make([]int, maxCols)
	for _, o := range orders {
		day := o.CreatedAt.UTC().Truncate(24 * time.Hour)
		daysAgo := int(now.Sub(day).Hours() / 24)
		col := maxCols - 1 - daysAgo
		if col <= 0 {
			col = 0 // leftmost bucket: catch-all for anything older than maxCols-1 days
		}
		buckets[col]++
	}

	maxVal := 0
	for _, v := range buckets {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return ""
	}

	rows := make([]string, maxHeight)
	for rowIdx := maxHeight; rowIdx >= 1; rowIdx-- {
		var sb strings.Builder
		for col := 0; col < maxCols; col++ {
			// Ceiling division: any non-zero bucket shows at least 1 cell.
			filled := (buckets[col]*maxHeight + maxVal - 1) / maxVal
			if filled >= rowIdx {
				sb.WriteRune('█')
			} else {
				sb.WriteByte(' ')
			}
		}
		rows[maxHeight-rowIdx] = sb.String()
	}
	return strings.Join(rows, "\n")
}
