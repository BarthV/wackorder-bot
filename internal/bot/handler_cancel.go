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
		respond(s, i, errEmbed("Order ID is required."))
		return
	}
	orderID := idOpt.IntValue()

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Order #%d not found.", orderID)))
		return
	}

	isCreator := order.CreatorID == caller
	if err := model.ValidateTransition(order.Status, model.StatusCanceled, isCreator); err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	if err := h.store.UpdateStatus(context.Background(), orderID, model.StatusCanceled, nil); err != nil {
		slog.Error("failed to cancel order", "order_id", orderID, "by", caller, "err", err)
		respond(s, i, errEmbed("Failed to cancel the order. Please try again later."))
		return
	}

	slog.Info("order canceled", "order_id", orderID, "by", caller)
	respond(s, i, okEmbed(fmt.Sprintf("Order #%d has been canceled.", orderID)))
}
