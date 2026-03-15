package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/barthv/wackorder-bot/internal/model"
)

// handleOrderCancel processes the /order-cancel slash command.
func (h *handler) handleOrderCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	opts := optionMap(i.ApplicationCommandData().Options)
	idOpt, ok := opts["id"]
	if !ok {
		respond(s, i, errEmbed("L'identifiant de la commande est requis."))
		return
	}
	orderID := idOpt.IntValue()

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Commande #%d introuvable.", orderID)))
		return
	}

	isCreator := order.CreatorID == caller
	if err := model.ValidateTransition(order.Status, model.StatusCanceled, isCreator); err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	if err := h.store.UpdateStatus(context.Background(), orderID, model.StatusCanceled, caller); err != nil {
		slog.Error("failed to cancel order", "order_id", orderID, "by", caller, "err", err)
		respond(s, i, errEmbed("Impossible d'annuler la commande. Réessaie plus tard."))
		return
	}

	slog.Info("order canceled", "order_id", orderID, "by", caller)
	respond(s, i, okEmbed(fmt.Sprintf("Commande #%d annulée.", orderID)))
}
