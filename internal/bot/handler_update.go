package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/barthv/wackorder-bot/internal/model"
	"github.com/bwmarrin/discordgo"
)

// statusSelectCustomID is the prefix for string select custom IDs encoding the order ID.
const statusSelectCustomID = "order_status_select:"

// statusChoices lists user-selectable status values with display labels.
var statusChoices = []struct {
	name  string
	value string
}{
	{"📦 Prêt — ressource prête", "ready"},
	{"✅ Terminé — commande complétée", "done"},
	{"📋 Commandé — remettre en attente", "ordered"},
}

// handleOrderUpdateAutocomplete responds to autocomplete requests for the /order-update status option.
func (h *handler) handleOrderUpdateAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "status" && opt.Focused {
			query = strings.ToLower(strings.TrimSpace(opt.StringValue()))
			break
		}
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, c := range statusChoices {
		if query == "" || strings.Contains(c.value, query) || strings.Contains(strings.ToLower(c.name), query) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: c.name, Value: c.value})
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ //nolint:errcheck
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

// handleOrderUpdate processes the /order-update slash command.
// Priority: my-book > id+status > id alone (string select).
func (h *handler) handleOrderUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := optionMap(i.ApplicationCommandData().Options)

	// my-book takes precedence over all other options.
	if allReadyOpt, ok := opts["my-book"]; ok {
		newStatus, err := model.ParseStatus(allReadyOpt.StringValue())
		if err != nil {
			respond(s, i, errEmbed(err.Error()))
			return
		}
		h.applyAllMyReady(s, i, newStatus)
		return
	}

	idOpt, hasID := opts["id"]
	if !hasID {
		respond(s, i, errEmbed("Fournis un order ID ou utilise `my-book` pour traiter toutes tes commandes 'ready'."))
		return
	}
	orderID := idOpt.IntValue()

	if statusOpt, hasStatus := opts["status"]; hasStatus {
		newStatus, err := model.ParseStatus(statusOpt.StringValue())
		if err != nil {
			respond(s, i, errEmbed(err.Error()))
			return
		}
		h.applyStatusUpdate(s, i, orderID, newStatus)
		return
	}

	// No status given — fetch order and show a string select of valid next statuses.
	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Commande #%d introuvable.", orderID)))
		return
	}

	nexts := model.ValidNextStatuses(order.Status)
	if len(nexts) == 0 {
		respond(s, i, errEmbed(fmt.Sprintf("La commande #%d est au statut **%s** et ne peut plus être modifiée.", orderID, statusLabel(order.Status))))
		return
	}

	options := make([]discordgo.SelectMenuOption, len(nexts))
	for j, st := range nexts {
		options[j] = discordgo.SelectMenuOption{
			Label:   statusLabel(st),
			Value:   string(st),
			Default: false,
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{orderEmbed(order)},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    fmt.Sprintf("%s%d", statusSelectCustomID, orderID),
							Placeholder: "Choisir le nouveau statut…",
							Options:     options,
						},
					},
				},
			},
		},
	}); err != nil {
		slog.Debug("failed to send status select", "err", err)
	}
}

// handleStatusSelect processes a string select interaction for status update.
func (h *handler) handleStatusSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse order ID from custom ID suffix.
	idStr := strings.TrimPrefix(i.MessageComponentData().CustomID, statusSelectCustomID)
	var orderID int64
	if _, err := fmt.Sscanf(idStr, "%d", &orderID); err != nil || orderID <= 0 {
		respondComponent(s, i, errEmbed("Identifiant de commande invalide."))
		return
	}

	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}

	newStatus, err := model.ParseStatus(values[0])
	if err != nil {
		respondComponent(s, i, errEmbed(err.Error()))
		return
	}

	caller := callerID(i)
	if caller == "" {
		respondComponent(s, i, errEmbed("Impossible de déterminer ton identité. Utilise cette commande dans un serveur."))
		return
	}

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respondComponent(s, i, errEmbed(fmt.Sprintf("Commande #%d introuvable.", orderID)))
		return
	}

	isCreator := order.CreatorID == caller
	if err := model.ValidateTransition(order.Status, newStatus, isCreator); err != nil {
		respondComponent(s, i, errEmbed(err.Error()))
		return
	}

	if err := h.store.UpdateStatus(context.Background(), orderID, newStatus, caller); err != nil {
		slog.Error("failed to update order status", "order_id", orderID, "new_status", newStatus, "by", caller, "err", err)
		respondComponent(s, i, errEmbed("Impossible de mettre à jour la commande. Réessaie plus tard."))
		return
	}

	slog.Info("order status updated", "order_id", orderID, "new_status", newStatus, "by", caller)

	updated, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respondComponent(s, i, okEmbed(fmt.Sprintf("Commande #%d mise à jour : %s.", orderID, newStatus)))
		return
	}

	// Replace the select message with the updated order embed.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ //nolint:errcheck
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{orderEmbed(updated)},
			Components: []discordgo.MessageComponent{}, // remove the select
		},
	})
}

// applyStatusUpdate validates and applies a status transition from a slash command interaction.
func (h *handler) applyStatusUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, orderID int64, newStatus model.Status) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	order, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, errEmbed(fmt.Sprintf("Commande #%d introuvable.", orderID)))
		return
	}

	isCreator := order.CreatorID == caller
	if err := model.ValidateTransition(order.Status, newStatus, isCreator); err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	if err := h.store.UpdateStatus(context.Background(), orderID, newStatus, caller); err != nil {
		slog.Error("failed to update order status", "order_id", orderID, "new_status", newStatus, "by", caller, "err", err)
		respond(s, i, errEmbed("Impossible de mettre à jour la commande. Réessaie plus tard."))
		return
	}

	slog.Info("order status updated", "order_id", orderID, "new_status", newStatus, "by", caller)

	updated, err := h.store.GetByID(context.Background(), orderID)
	if err != nil {
		respond(s, i, okEmbed(fmt.Sprintf("Commande #%d mise à jour : %s.", orderID, newStatus)))
		return
	}
	respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(updated)})
}

// applyAllMyReady bulk-updates all "ready" orders last updated by the caller to newStatus.
// Only ready→ordered and ready→done transitions are permitted.
func (h *handler) applyAllMyReady(s *discordgo.Session, i *discordgo.InteractionCreate, newStatus model.Status) {
	if newStatus != model.StatusOrdered && newStatus != model.StatusDone {
		respond(s, i, errEmbed("my-book n'accepte que les statuts 'ordered' ou 'done'."))
		return
	}

	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	orders, err := h.store.ListReadyByUpdater(context.Background(), caller)
	if err != nil {
		slog.Error("failed to list ready orders by updater", "by", caller, "err", err)
		respond(s, i, errEmbed("Impossible de récupérer les commandes. Réessaie plus tard."))
		return
	}

	if len(orders) == 0 {
		respond(s, i, errEmbed("Aucune commande 'prête' trouvée dont tu es le dernier modificateur."))
		return
	}

	var updated, failed int
	for _, o := range orders {
		if err := model.ValidateTransition(o.Status, newStatus, o.CreatorID == caller); err != nil {
			failed++
			continue
		}
		if err := h.store.UpdateStatus(context.Background(), o.ID, newStatus, caller); err != nil {
			slog.Error("failed to bulk-update order", "order_id", o.ID, "new_status", newStatus, "by", caller, "err", err)
			failed++
		} else {
			updated++
		}
	}

	slog.Info("bulk ready update", "new_status", newStatus, "updated", updated, "failed", failed, "by", caller)

	msg := fmt.Sprintf("%d commande(s) mises à jour vers **%s**.", updated, statusLabel(newStatus))
	if failed > 0 {
		msg += fmt.Sprintf(" %d échec(s).", failed)
	}
	respond(s, i, okEmbed(msg))
}

// respondComponent replaces the original component message content in-place.
func respondComponent(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.InteractionResponseData) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ //nolint:errcheck
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: data,
	})
}
