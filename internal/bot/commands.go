package bot

import "github.com/bwmarrin/discordgo"

// commands returns the slash command definitions to register with Discord.
func commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "order",
			Description: "Passer une commande de ressource (via popup si les options sont non déclarées)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "component",
					Description:  "Nom de la ressource désirée (Taranite, Hadanite, Riccite, ...)",
					Required:     false,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "quality",
					Description: "Qualité minimale",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "quantity",
					Description: "Quantité en cSCU ou en Unités (selon le type de ressource)",
					Required:    false,
					MinValue:    floatPtr(1),
				},
			},
		},
		{
			Name:        "order-list",
			Description: "Lister les commandes disponibles avec possibilité de filtrage",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mode",
					Description: "Filtrer les commandes affichées",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "self — mes commandes (tous statuts)", Value: "self"},
						{Name: "pending — commandes en attente (commandées / prêtes)", Value: "pending"},
						{Name: "all — toutes les commandes", Value: "all"},
						{Name: "booked — toutes les commande que j'ai passée en 'prêt'", Value: "booked"},
					{Name: "done — commandes terminées", Value: "done"},
					},
				},
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "component",
					Description:  "Filtrer par nom de ressource",
					Required:     false,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "older-than",
					Description: "Filtrer les commandes plus anciennes que la durée, ex. 7, 7d, 2w, 1mo",
					Required:    false,
				},
			},
		},
		{
			Name:        "order-update",
			Description: "Mettre à jour le statut d'une commande.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "booked",
					Description: "Mettre à jour toutes les commandes que j'ai passé en 'prêt'",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "ordered — remettre en attente", Value: "ordered"},
						{Name: "done — marquer comme terminées", Value: "done"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "Identifiant de la commande",
					Required:    false,
					MinValue:    floatPtr(1),
				},
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "status",
					Description:  "Nouveau statut (optionnel — une liste s'affiche sinon)",
					Required:     false,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "order-stats",
			Description: "Afficher un résumé des quantités en attente par ressource.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "jours",
					Description: "Nombre de jours dans l'histogramme (1-32, défaut 32)",
					Required:    false,
					MinValue:    floatPtr(1),
					MaxValue:    histDefaultCols,
				},
			},
		},
		{
			Name:        "order-help",
			Description: "Afficher l'aide et la liste des ressources disponibles.",
		},
		{
			Name:        "order-detail",
			Description: "Afficher tous les détails d'une commande.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "Identifiant de la commande",
					Required:    true,
					MinValue:    floatPtr(1),
				},
			},
		},
		{
			Name:        "order-cancel",
			Description: "Annule une de vos commandes.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "Identifiant de la commande à annuler",
					Required:    true,
					MinValue:    floatPtr(1),
				},
			},
		},
	}
}

func floatPtr(f float64) *float64 { return &f }
