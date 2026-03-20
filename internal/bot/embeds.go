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
		{Name: "Commandé par", Value: fmt.Sprintf("<@%s>", o.CreatorID), Inline: true},
		// {Name: "Créé le", Value: formatTime(o.CreatedAt), Inline: true},
	}

	if o.MinQuality != 0 {
		fields = append([]*discordgo.MessageEmbedField{
			fields[0],
			{Name: "Qualité", Value: fmt.Sprintf("%d", o.MinQuality), Inline: true},
		}, fields[1:]...)
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Commande #%d", o.ID),
		Color:  statusColor(o.Status),
		Fields: fields,
		// Footer: &discordgo.MessageEmbedFooter{
		// 	Text: fmt.Sprintf("Mis à jour : %s", formatTime(o.UpdatedAt)),
		// },
	}
}

// orderDetailEmbed builds a full-detail Discord embed for a single order,
// including the last-updater and exact updated_at timestamp.
func orderDetailEmbed(o *model.Order) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Ressource", Value: o.Component, Inline: true},
		{Name: "Quantité", Value: fmt.Sprintf("%d cSCU", o.Quantity), Inline: true},
		{Name: "Statut", Value: statusLabel(o.Status), Inline: true},
		{Name: "Commandé par", Value: fmt.Sprintf("<@%s>", o.CreatorID), Inline: true},
		{Name: "Créé le", Value: formatTime(o.CreatedAt), Inline: true},
	}

	if o.MinQuality != 0 {
		fields = append([]*discordgo.MessageEmbedField{
			fields[0],
			{Name: "Qualité", Value: fmt.Sprintf("%d", o.MinQuality), Inline: true},
		}, fields[1:]...)
	}

	updater := "—"
	if o.UpdatedBy != "" {
		updater = fmt.Sprintf("<@%s>", o.UpdatedBy)
	}
	fields = append(fields,
		&discordgo.MessageEmbedField{Name: "Dernier modificateur", Value: updater, Inline: true},
		&discordgo.MessageEmbedField{Name: "Dernière mise à jour", Value: formatTime(o.UpdatedAt), Inline: true},
	)

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("Commande #%d — détail", o.ID),
		Color:  statusColor(o.Status),
		Fields: fields,
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

	const perEmbed = 15 // rows per embed to stay within 4096 char limit
	var embeds []*discordgo.MessageEmbed

	for i := 0; i < len(orders); i += perEmbed {
		end := i + perEmbed
		if end > len(orders) {
			end = len(orders)
		}
		chunk := orders[i:end]

		var sb strings.Builder
		sb.WriteString("```\n")
		sb.WriteString(fmt.Sprintf("%-5s %-14s %-10s %7s  %-4s %-12s %s\n",
			"ID", "Ressource", "Statut", "Qty", "Qual", "Par", "Date"))
		sb.WriteString(fmt.Sprintf("%-5s %-14s %-10s %7s  %-4s %-12s %s\n",
			"-----", "--------------", "----------", "-------", "----", "------------", "----------"))
		for _, o := range chunk {
			comp := o.Component
			if len(comp) > 14 {
				comp = comp[:13] + "."
			}
			status := statusShort(o.Status)
			quality := "-"
			if o.MinQuality != 0 {
				quality = fmt.Sprintf("%d", o.MinQuality)
			}
			creator := o.CreatorName
			if len(creator) > 12 {
				creator = creator[:11] + "."
			}
			sb.WriteString(fmt.Sprintf("#%-4d %-14s %-10s %7d  %-4s %-12s %s\n",
				o.ID, comp, status, o.Quantity, quality, creator, formatDate(o.CreatedAt)))
		}
		sb.WriteString("```")

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

// orderListPlain builds a plain-text message (no embed) for a list of orders.
func orderListPlain(orders []model.Order, title string) string {
	if len(orders) == 0 {
		return fmt.Sprintf("**%s**\n_Aucune commande trouvée._", title)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** (%d)\n", title, len(orders)))
	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("%-5s %-14s %-10s %7s  %-4s %-12s %s\n",
		"ID", "Ressource", "Statut", "Qty", "Qual", "Par", "Date"))
	sb.WriteString(fmt.Sprintf("%-5s %-14s %-10s %7s  %-4s %-12s %s\n",
		"-----", "--------------", "----------", "-------", "----", "------------", "----------"))
	for _, o := range orders {
		comp := o.Component
		if len(comp) > 14 {
			comp = comp[:13] + "."
		}
		quality := "-"
		if o.MinQuality != 0 {
			quality = fmt.Sprintf("%d", o.MinQuality)
		}
		creator := o.CreatorName
		if len(creator) > 12 {
			creator = creator[:11] + "."
		}
		sb.WriteString(fmt.Sprintf("#%-4d %-14s %-10s %7d  %-4s %-12s %s\n",
			o.ID, comp, statusShort(o.Status), o.Quantity, quality, creator, formatDate(o.CreatedAt)))
	}
	sb.WriteString("```")
	return sb.String()
}

func statusShort(s model.Status) string {
	switch s {
	case model.StatusOrdered:
		return "ordered"
	case model.StatusReady:
		return "ready"
	case model.StatusDone:
		return "done"
	case model.StatusCanceled:
		return "canceled"
	}
	return string(s)
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
