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
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "component",
					Description: "Nom de la ressource désirée (Taranite, Hadanite, Riccite, ...)",
					Required:    false,
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
			Name:        "order-view",
			Description: "Lister les commandes enregistrées avec possibilité de filtrage",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "view",
					Description: "Which orders to show",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "self — your own orders (all statuses)", Value: "self"},
						{Name: "pending — unfinished orders (ordered / ready)", Value: "pending"},
						{Name: "all — every order", Value: "all"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "component",
					Description: "Filter by component name (case-insensitive, all users)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "since",
					Description: "Filter by creation date, e.g. 2026-01-15 (all users)",
					Required:    false,
				},
			},
		},
		{
			Name:        "order-update",
			Description: "Update an order's status. Omit options to use the modal form.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "Order ID",
					Required:    false,
					MinValue:    floatPtr(1),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "status",
					Description: "New status",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "ready — item produced/gathered", Value: "ready"},
						{Name: "done — order complete", Value: "done"},
					},
				},
			},
		},
		{
			Name:        "order-cancel",
			Description: "Cancel one of your orders.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "Order ID to cancel",
					Required:    true,
					MinValue:    floatPtr(1),
				},
			},
		},
	}
}

func floatPtr(f float64) *float64 { return &f }
