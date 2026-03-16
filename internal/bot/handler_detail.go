package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// handleOrderDetail processes the /order-detail slash command.
func (h *handler) handleOrderDetail(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)
	orderID := opts["id"].IntValue()

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Commande #%d introuvable.", orderID)))
		return
	}

	respondEmbeds(s, i, []*discordgo.MessageEmbed{orderDetailEmbed(order)}, discordgo.MessageFlagsEphemeral)
}
