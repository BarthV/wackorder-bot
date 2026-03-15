package bot

import "github.com/bwmarrin/discordgo"

// commands returns the slash command definitions to register with Discord.
func commands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "order",
			Description: "Place a new component order. Omit options to use the modal form.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "component",
					Description: "Component name (e.g. 'Klaus & Werner FS-9 Shield Generator')",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "quality",
					Description: "Minimum quality (e.g. A, B, Mil-Spec)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "quantity",
					Description: "Quantity in cSCU",
					Required:    false,
					MinValue:    floatPtr(1),
				},
			},
		},
		{
			Name:        "orders",
			Description: "List orders. Defaults to your own orders.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "view",
					Description: "Which orders to show",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "mine — your own orders (all statuses)", Value: "mine"},
						{Name: "pending — all unfinished orders", Value: "pending"},
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
						{Name: "in-transit — meeting booked (requires meeting_date)", Value: "in-transit"},
						{Name: "done — order complete", Value: "done"},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "meeting_date",
					Description: "Meeting date for in-transit status, e.g. 2026-01-20T18:00:00Z",
					Required:    false,
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
