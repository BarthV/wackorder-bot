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
// It shows summed pending quantities per resource (sorted by quantity desc),
// an ASCII histogram of creation dates with axes, and per-resource filter buttons.
func (h *handler) handleOrderStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

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

	// Determine histogram column count: explicit option or default (full width).
	maxCols := histDefaultCols
	if colsOpt, ok := opts["jours"]; ok {
		v := int(colsOpt.IntValue())
		if v >= 1 && v <= histDefaultCols {
			maxCols = v
		}
	}

	// Sum quantities per component.
	totals := make(map[string]int)
	for _, o := range orders {
		totals[o.Component] += o.Quantity
	}

	// Sort components by quantity descending; ties broken alphabetically.
	components := make([]string, 0, len(totals))
	for comp := range totals {
		components = append(components, comp)
	}
	sort.Slice(components, func(a, b int) bool {
		if totals[components[a]] != totals[components[b]] {
			return totals[components[a]] > totals[components[b]]
		}
		return components[a] < components[b]
	})

	// Resources code block: 1 per line, quantities right-aligned.
	maxNameLen := 0
	for _, comp := range components {
		if len(comp) > maxNameLen {
			maxNameLen = len(comp)
		}
	}
	maxQtyLen := len(fmt.Sprintf("%d", totals[components[0]])) // already sorted desc
	var resLines []string
	for _, comp := range components {
		resLines = append(resLines, fmt.Sprintf("%-*s  %*d", maxNameLen, comp, maxQtyLen, totals[comp]))
	}
	resourcesBlock := "```\n" + strings.Join(resLines, "\n") + "\n```"

	// Histogram code block.
	histBlock := ""
	if hist := buildHistogram(orders, maxCols, histMaxHeight); hist != "" {
		histBlock = "```\n" + hist + "\n```\n"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Commandes en attente",
		Description: histBlock + resourcesBlock,
		Color:       colorOrdered,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d commande(s) en attente", len(orders)),
		},
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

// buildHistogram returns an ASCII histogram with y-axis labels and x-axis graduation.
// Columns: leftmost = catch-all for older orders, rightmost = today.
// Example output (maxCols=10, maxHeight=4):
//
//	4│    █    █
//	3│    █    █  █
//	2│ █  █  █ █  █
//	1│ █  █  █ █  █  █
//	 └──────────────→
//	  +9d    +4d  now
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

	// Y-axis label width (number of digits of the max value).
	yLabelWidth := len(fmt.Sprintf("%d", maxVal))

	var lines []string

	// Bar rows, top to bottom.
	for rowIdx := maxHeight; rowIdx >= 1; rowIdx-- {
		// The count threshold for this row level (ceiling division).
		rowVal := (rowIdx*maxVal + maxHeight - 1) / maxHeight
		var sb strings.Builder
		fmt.Fprintf(&sb, "%*d│", yLabelWidth, rowVal)
		for col := 0; col < maxCols; col++ {
			filled := (buckets[col]*maxHeight + maxVal - 1) / maxVal
			if filled >= rowIdx {
				sb.WriteRune('█')
			} else {
				sb.WriteByte(' ')
			}
		}
		lines = append(lines, sb.String())
	}

	// X-axis line: spaces aligned with y-axis, then corner + dashes + arrow.
	xPad := strings.Repeat(" ", yLabelWidth) + "└"
	lines = append(lines, xPad+strings.Repeat("─", maxCols)+"→")

	// X-axis labels.
	lines = append(lines, strings.Repeat(" ", yLabelWidth+1)+buildXAxisLabels(maxCols))

	return strings.Join(lines, "\n")
}

// buildXAxisLabels returns a maxCols-wide string with day labels at regular intervals.
// Leftmost position: "+Nd" catch-all label. Rightmost: "now". Every ~8 cols in between.
func buildXAxisLabels(maxCols int) string {
	buf := []rune(strings.Repeat(" ", maxCols))

	place := func(col int, label string) {
		for k, ch := range label {
			if col+k < len(buf) {
				buf[col+k] = ch
			}
		}
	}

	leftLabel := fmt.Sprintf("+%dd", maxCols-1)
	nowLabel := "now"
	nowPos := maxCols - len(nowLabel)

	place(0, leftLabel)
	if nowPos > len(leftLabel) {
		place(nowPos, nowLabel)
	}

	// Intermediate labels every 8 columns, only if they don't collide with "now".
	for col := 8; col < nowPos-2; col += 8 {
		place(col, fmt.Sprintf("+%dd", maxCols-1-col))
	}

	return string(buf)
}
