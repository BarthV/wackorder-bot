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
					Name:        "view",
					Description: "Filtrer les commandes affichées",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "self — mes commandes (tous statuts)", Value: "self"},
						{Name: "pending — commandes en attente (commandé / prêt)", Value: "pending"},
						{Name: "all — toutes les commandes", Value: "all"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "component",
					Description: "Filtrer par nom de ressource (insensible à la casse, tous utilisateurs)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "since",
					Description: "Filtrer par date de création, ex. 2026-01-15 (tous utilisateurs)",
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
					Name:        "all-my-ready",
					Description: "Mettre à jour toutes mes commandes 'prêtes' vers ce statut (priorité sur les autres options)",
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
					Description: "Nombre de jours dans l'histogramme (1–32, défaut 32)",
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
