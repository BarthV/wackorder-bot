package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/barthv/wackorder-bot/internal/model"
	"github.com/bwmarrin/discordgo"
)

// Status colors for Discord embeds.
const (
	colorOrdered  = 0x5865F2 // blurple
	colorReady    = 0x57F287 // green
	colorDone     = 0x95A5A6 // grey
	colorCanceled = 0xED4245 // red
	colorError    = 0xED4245 // red
)

func statusColor(s model.Status) int {
	switch s {
	case model.StatusOrdered:
		return colorOrdered
	case model.StatusReady:
		return colorReady
	case model.StatusDone:
		return colorDone
	case model.StatusCanceled:
		return colorCanceled
	}
	return colorOrdered
}

func statusLabel(s model.Status) string {
	switch s {
	case model.StatusOrdered:
		return "📋 Ordered"
	case model.StatusReady:
		return "📦 Ready"
	case model.StatusDone:
		return "✅ Done"
	case model.StatusCanceled:
		return "❌ Canceled"
	}
	return string(s)
}

// orderEmbed builds a Discord embed for a single order.
func orderEmbed(o *model.Order) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Component", Value: o.Component, Inline: true},
		{Name: "Min Quality", Value: qualityOrAny(o.MinQuality), Inline: true},
		{Name: "Quantity", Value: fmt.Sprintf("%d cSCU", o.Quantity), Inline: true},
		{Name: "Status", Value: statusLabel(o.Status), Inline: true},
		{Name: "Ordered by", Value: fmt.Sprintf("%s (<@%s>)", o.CreatorName, o.CreatorID), Inline: true},
		{Name: "Created", Value: formatTime(o.CreatedAt), Inline: true},
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Order #%d", o.ID),
		Color:  statusColor(o.Status),
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Last updated: %s", formatTime(o.UpdatedAt)),
		},
	}
}

// orderListEmbeds builds one or more Discord embeds for a list of orders.
// Discord limits: 25 fields per embed, 6000 chars total.
func orderListEmbeds(orders []model.Order, title string) []*discordgo.MessageEmbed {
	if len(orders) == 0 {
		return []*discordgo.MessageEmbed{
			{
				Title:       title,
				Description: "_No orders found._",
				Color:       colorDone,
			},
		}
	}

	const perEmbed = 20 // keep well under the 25-field limit
	var embeds []*discordgo.MessageEmbed

	for i := 0; i < len(orders); i += perEmbed {
		end := i + perEmbed
		if end > len(orders) {
			end = len(orders)
		}
		chunk := orders[i:end]

		var sb strings.Builder
		for _, o := range chunk {
			line := fmt.Sprintf("`#%d` **%s** — %s | %d cSCU - Q%s | <@%s> (%s)\n",
				o.ID, o.Component, statusLabel(o.Status),
				o.Quantity, qualityOrAny(o.MinQuality),
				o.CreatorID, formatDate(o.CreatedAt))
			sb.WriteString(line)
		}

		pageTitle := title
		if len(embeds) > 0 {
			pageTitle = fmt.Sprintf("%s (cont.)", title)
		}

		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       pageTitle,
			Description: sb.String(),
			Color:       colorOrdered,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("%d order(s) total", len(orders)),
			},
		})
	}

	return embeds
}

// errEmbed builds an ephemeral error embed.
func errEmbed(msg string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Description: "❌ " + msg,
				Color:       colorError,
			},
		},
	}
}

// okEmbed builds a simple success embed (ephemeral).
func okEmbed(msg string) *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Description: "✅ " + msg,
				Color:       colorReady,
			},
		},
	}
}

func qualityOrAny(q string) string {
	if q == "" {
		return "any"
	}
	return q
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02")
}
