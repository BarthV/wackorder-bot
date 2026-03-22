package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleOrderHelp processes the /order-help slash command.
func (h *handler) handleOrderHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	workflow := "**1- Passer une commande (la quantité est en cSCU ou en Unités)**\n" +
		"```/order (via une popup)\n/order component:Taranite quality:850 quantity:150\n/order component:Dolivine quantity:35\n```\n" +
		"**2- Rechercher les commandes en cours**\n" +
		"```/order-list (rechercher et filtrer les demandes en cours)\n/order-list component:Taranite older-than:1w (filtre par resource et age)\n/order-list mode:self (pour filtrer ses commandes)\n/order-list mode:done (commandes terminées)\n```\n" +
		"**3- Marquer la commande comme OK / collectée (minée, récoltée, ...)**\n" +
		"```/order-update id:42 status:ready\n```\n" +
		"**4- Terminer la commande (livraison)**\n" +
		"```/order-update id:42 status:done (commande unique)\n/order-update booked:done (toutes celles que j'ai passées 'ready')\n```\n" +
		"**Annuler une de ses propres commandes**\n" +
		"```/order-cancel id:42```\n" +
		"**Afficher les statistiques globales des commandes en cours**\n" +
		"```/order-stats\n/order-detail id:42 (commande unique)\n```\n" +
		"**Attention:** les commandes terminées sont automatiquement supprimées après 31 jours."

	var sb strings.Builder
	for i, r := range resources {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(r)
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "📖 G.A.L.E.R.E — gestion des commandes",
			Color:       colorOrdered,
			Description: workflow,
		},
		{
			Title:       "📦 Liste des ressources (case-insensitive)",
			Color:       colorReady,
			Description: sb.String(),
		},
	}

	respondEmbeds(s, i, embeds, discordgo.MessageFlagsEphemeral)
}
