package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleOrderHelp processes the /order-help slash command.
func (h *handler) handleOrderHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	workflow := "**1. Passer une commande (la quantité est en cSCU)**\n" +
		"```/order (via une popup)\n/order component:Taranite quality:850 quantity:150\n/order component:Dolivine quantity:35\n```\n" +
		"**2. Rechercher les commandes en cours**\n" +
		"```/order-list (pour un recap des demandes en cours)\n/order-list view:pending component:Taranite\n/order-list view:self (pour regarder ses propres commandes)\n```\n" +
		"**3. Marquer la commande comme disponible (minée, récoltée, produite ...)**\n" +
		"```/order-update id:42 status:ready```\n" +
		"**4. Terminer la commande (livraison)**\n" +
		"```/order-update id:42 status:done```\n" +
		"**Annuler une de ses propres commandes**\n" +
		"```/order-cancel id:42```"

	var sb strings.Builder
	for i, r := range resources {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(r)
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "📖 G.A.L.E.R.E — Aide à la prise de commande",
			Color:       colorOrdered,
			Description: workflow,
		},
		{
			Title:       "📦 Ressources supportées",
			Color:       colorReady,
			Description: sb.String(),
		},
	}

	respondEmbeds(s, i, embeds, discordgo.MessageFlagsEphemeral)
}
