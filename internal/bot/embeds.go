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
		return "📋 Commandé"
	case model.StatusReady:
		return "📦 Prêt"
	case model.StatusDone:
		return "✅ Terminé"
	case model.StatusCanceled:
		return "❌ Annulé"
	}
	return string(s)
}

// orderEmbed builds a Discord embed for a single order.
func orderEmbed(o *model.Order) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Ressource", Value: o.Component, Inline: true},
		{Name: "Quantité", Value: fmt.Sprintf("%d cSCU", o.Quantity), Inline: true},
		{Name: "Statut", Value: statusLabel(o.Status), Inline: true},
		{Name: "Commandé par", Value: fmt.Sprintf("%s (<@%s>)", o.CreatorName, o.CreatorID), Inline: true},
		{Name: "Créé le", Value: formatTime(o.CreatedAt), Inline: true},
	}

	if o.MinQuality != "0" && o.MinQuality != "" {
		fields = append([]*discordgo.MessageEmbedField{
			fields[0],
			{Name: "Qualité min.", Value: o.MinQuality, Inline: true},
		}, fields[1:]...)
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Commande #%d", o.ID),
		Color:  statusColor(o.Status),
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Mis à jour : %s", formatTime(o.UpdatedAt)),
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
				Description: "_Aucune commande trouvée._",
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
			quality := ""
			if o.MinQuality != "0" && o.MinQuality != "" {
				quality = fmt.Sprintf(" - Q%s", o.MinQuality)
			}
			line := fmt.Sprintf("`#%d` **%s** — %s | %d cSCU%s | <@%s> (%s)\n",
				o.ID, o.Component, statusLabel(o.Status),
				o.Quantity, quality,
				o.CreatorID, formatDate(o.CreatedAt))
			sb.WriteString(line)
		}

		pageTitle := title
		if len(embeds) > 0 {
			pageTitle = fmt.Sprintf("%s (suite)", title)
		}

		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       pageTitle,
			Description: sb.String(),
			Color:       colorOrdered,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("%d commande(s)", len(orders)),
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
