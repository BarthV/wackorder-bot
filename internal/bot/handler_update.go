package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/barthv/wackorder-bot/internal/model"
)

// handleOrderUpdate processes the /order-update slash command.
func (h *handler) handleOrderUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)
	idOpt, hasID := opts["id"]
	statusOpt, hasStatus := opts["status"]

	// Both id and status provided — update directly.
	if hasID && hasStatus {
		orderID := idOpt.IntValue()
		newStatus, err := model.ParseStatus(statusOpt.StringValue())
		if err != nil {
			respond(s, i, errEmbed(err.Error()))
			return
		}
		h.applyUpdate(s, i, orderID, newStatus)
		return
	}

	// Missing options — open modal.
	idValue := ""
	if hasID {
		idValue = fmt.Sprintf("%d", idOpt.IntValue())
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "update_modal",
			Title:    "Update Order Status",
			Components: []discordgo.MessageComponent{
				textRow("id", "Order ID", "e.g. 42", idValue, true),
				textRow("status", "New Status", "ready / done", "", true),
			},
		},
	}); err != nil {
		slog.Debug("failed to open update modal", "err", err)
	}
}

// handleUpdateModalSubmit handles the modal submission for /order-update.
func (h *handler) handleUpdateModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	fields := modalFields(data.Components)

	idStr := strings.TrimSpace(fields["id"])
	statusStr := strings.TrimSpace(fields["status"])

	var orderID int64
	if _, err := fmt.Sscanf(idStr, "%d", &orderID); err != nil || orderID <= 0 {
		respond(s, i, errEmbed("Invalid order ID."))
		return
	}

	newStatus, err := model.ParseStatus(statusStr)
	if err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	h.applyUpdate(s, i, orderID, newStatus)
}

// applyUpdate validates and applies a status transition, then responds with the updated order embed.
func (h *handler) applyUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, orderID int64, newStatus model.Status) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Order #%d not found.", orderID)))
		return
	}

	isCreator := order.CreatorID == caller
	if err := model.ValidateTransition(order.Status, newStatus, isCreator); err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	if err := h.store.UpdateStatus(context.Background(), orderID, newStatus); err != nil {
		slog.Error("failed to update order status", "order_id", orderID, "new_status", newStatus, "by", caller, "err", err)
		respond(s, i, errEmbed("Failed to update the order. Please try again later."))
		return
	}

	slog.Info("order status updated", "order_id", orderID, "new_status", newStatus, "by", caller)

	updated, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, okEmbed(fmt.Sprintf("Order #%d updated to %s.", orderID, newStatus)))
		return
	}
	respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(updated)})
}
