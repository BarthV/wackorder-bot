package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	maxComponentLen = 200
	maxQualityLen   = 50
)

// handleOrder processes the /order slash command.
// If all three options are present, creates the order directly.
// Otherwise, opens a modal to collect missing fields.
func (h *handler) handleOrder(s *discordgo.Session, i *discordgo.InteractionCreate) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	opts := optionMap(i.ApplicationCommandData().Options)

	component, hasComp := opts["component"]
	quality, hasQual := opts["quality"]
	quantityOpt, hasQty := opts["quantity"]

	// All fields provided — create directly.
	if hasComp && hasQual && hasQty {
		comp := strings.TrimSpace(component.StringValue())
		qual := strings.TrimSpace(quality.StringValue())

		if err := validateOrderFields(comp, qual); err != nil {
			respond(s, i, errEmbed(err.Error()))
			return
		}

		qty := int(quantityOpt.IntValue())
		if qty <= 0 {
			respond(s, i, errEmbed("Quantity must be greater than 0."))
			return
		}

		id, err := h.store.Create(context.Background(), caller, callerName(i), comp, qual, qty)
		if err != nil {
			slog.Error("failed to create order", "by", caller, "err", err)
			respond(s, i, errEmbed("Failed to create order. Please try again later."))
			return
		}
		order, err := h.store.GetByID(context.Background(), id)
		if err != nil {
			respond(s, i, okEmbed(fmt.Sprintf("Order #%d created.", id)))
			return
		}
		slog.Info("order created", "order_id", id, "by", caller)
		respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(order)})
		return
	}

	// Missing at least one field — open modal with pre-filled values.
	compValue, qualValue, qtyValue := "", "", ""
	if hasComp {
		compValue = component.StringValue()
	}
	if hasQual {
		qualValue = quality.StringValue()
	}
	if hasQty {
		qtyValue = strconv.FormatInt(quantityOpt.IntValue(), 10)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "order_modal",
			Title:    "Place an Order",
			Components: []discordgo.MessageComponent{
				textRow("component", "Component", "e.g. Klaus & Werner FS-9 Shield Generator", compValue, true),
				textRow("quality", "Minimum Quality", "e.g. A, B, Mil-Spec (leave blank for any)", qualValue, false),
				textRow("quantity", "Quantity (cSCU)", "e.g. 10", qtyValue, true),
			},
		},
	}); err != nil {
		slog.Debug("failed to open order modal", "err", err)
	}
}

// handleOrderModalSubmit handles the modal submission for /order.
func (h *handler) handleOrderModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	data := i.ModalSubmitData()
	fields := modalFields(data.Components)

	component := strings.TrimSpace(fields["component"])
	quality := strings.TrimSpace(fields["quality"])
	qtyStr := strings.TrimSpace(fields["quantity"])

	if err := validateOrderFields(component, quality); err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		respond(s, i, errEmbed("Quantity must be a positive integer."))
		return
	}

	id, err := h.store.Create(context.Background(), caller, callerName(i), component, quality, qty)
	if err != nil {
		slog.Error("failed to create order", "by", caller, "err", err)
		respond(s, i, errEmbed("Failed to create order. Please try again later."))
		return
	}

	order, err := h.store.GetByID(context.Background(), id)
	if err != nil {
		respond(s, i, okEmbed(fmt.Sprintf("Order #%d created.", id)))
		return
	}
	slog.Info("order created", "order_id", id, "by", caller)
	respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(order)})
}

// validateOrderFields checks component and quality field constraints.
func validateOrderFields(component, quality string) error {
	if component == "" {
		return fmt.Errorf("Component name cannot be empty.")
	}
	if len(component) > maxComponentLen {
		return fmt.Errorf("Component name must be %d characters or fewer.", maxComponentLen)
	}
	if len(quality) > maxQualityLen {
		return fmt.Errorf("Quality must be %d characters or fewer.", maxQualityLen)
	}
	return nil
}

// --- helpers ---

// optionMap converts a slice of options into a name→option map for easy lookup.
func optionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	m := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

// textRow builds a single-line text input component row for a modal.
func textRow(id, label, placeholder, value string, required bool) discordgo.MessageComponent {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:    id,
				Label:       label,
				Style:       discordgo.TextInputShort,
				Placeholder: placeholder,
				Value:       value,
				Required:    required,
			},
		},
	}
}

// modalFields extracts CustomID→Value pairs from modal component rows.
// Handles both value and pointer types since discordgo's unmarshaling can produce either.
func modalFields(components []discordgo.MessageComponent) map[string]string {
	fields := make(map[string]string)
	for _, row := range components {
		var rowComps []discordgo.MessageComponent
		switch v := row.(type) {
		case discordgo.ActionsRow:
			rowComps = v.Components
		case *discordgo.ActionsRow:
			rowComps = v.Components
		default:
			continue
		}
		for _, comp := range rowComps {
			switch ti := comp.(type) {
			case discordgo.TextInput:
				fields[ti.CustomID] = ti.Value
			case *discordgo.TextInput:
				fields[ti.CustomID] = ti.Value
			}
		}
	}
	return fields
}
