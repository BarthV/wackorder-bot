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
	maxComponentLen = 64
	maxQualityLen   = 4
)

// handleOrder processes the /order slash command.
// If component and quantity are both provided, creates the order directly.
// If component is provided but quantity is missing, opens a 2-field modal (quality + quantity).
// If component is missing, opens the full 3-field modal (component + quality + quantity).
func (h *handler) handleOrder(s *discordgo.Session, i *discordgo.InteractionCreate) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	opts := optionMap(i.ApplicationCommandData().Options)

	component, hasComp := opts["component"]
	quality, hasQual := opts["quality"]
	quantityOpt, hasQty := opts["quantity"]

	// Component and quantity provided — create directly (quality defaults to 0).
	if hasComp && hasQty {
		comp := strings.TrimSpace(component.StringValue())
		qual := 0
		if hasQual {
			parsed, err := strconv.Atoi(strings.TrimSpace(quality.StringValue()))
			if err != nil || parsed < 0 {
				respond(s, i, errEmbed("La qualité doit être un entier positif ou nul."))
				return
			}
			qual = parsed
		}

		canonical, err := validateOrderFields(comp)
		if err != nil {
			respond(s, i, errEmbed(err.Error()))
			return
		}
		comp = canonical

		qty := int(quantityOpt.IntValue())
		if qty <= 0 {
			respond(s, i, errEmbed("La quantité doit être supérieure à 0."))
			return
		}

		id, err := h.store.Create(context.Background(), caller, callerName(i), comp, qual, qty)
		if err != nil {
			slog.Error("failed to create order", "by", caller, "err", err)
			respond(s, i, errEmbed("Impossible de créer la commande. Réessaie plus tard."))
			return
		}
		logAction(s, h.logChannelID, fmt.Sprintf("#%d %s ( %d Q%d ) - Commandée par <@%s>", id, comp, qty, qual, caller))
		order, err := h.store.GetByID(context.Background(), id)
		if err != nil {
			respond(s, i, okEmbed(fmt.Sprintf("Commande #%d créée.", id)))
			return
		}
		slog.Info("order created", "order_id", id, "by", caller)
		respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(order)})
		return
	}

	qualValue := "0"
	if hasQual {
		qualValue = quality.StringValue()
	}
	qtyValue := ""
	if hasQty {
		qtyValue = strconv.FormatInt(quantityOpt.IntValue(), 10)
	}

	compValue := ""
	if hasComp {
		compValue = strings.TrimSpace(component.StringValue())
	}
	openOrderModal(s, i, compValue, qualValue, qtyValue)
}

// handleOrderModalSubmit handles the modal submission for /order.
// Component, quality, and quantity are all read from the modal fields.
func (h *handler) handleOrderModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	caller, ok := requireCallerID(s, i)
	if !ok {
		return
	}

	data := i.ModalSubmitData()
	fields := modalFields(data.Components)

	component := strings.TrimSpace(fields["component"])

	qualStr := strings.TrimSpace(fields["quality"])
	qual := 0
	if qualStr != "" {
		parsed, err := strconv.Atoi(qualStr)
		if err != nil || parsed < 0 {
			respond(s, i, errEmbed("La qualité doit être un entier positif ou nul."))
			return
		}
		qual = parsed
	}
	qtyStr := strings.TrimSpace(fields["quantity"])

	canonical, err := validateOrderFields(component)
	if err != nil {
		respond(s, i, errEmbed(err.Error()))
		return
	}
	component = canonical

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		respond(s, i, errEmbed("La quantité doit être un entier positif."))
		return
	}

	id, err := h.store.Create(context.Background(), caller, callerName(i), component, qual, qty)
	if err != nil {
		slog.Error("failed to create order", "by", caller, "err", err)
		respond(s, i, errEmbed("Impossible de créer la commande. Réessaie plus tard."))
		return
	}
	logAction(s, h.logChannelID, fmt.Sprintf("#%d %s ( %d Q%d ) - Commandée par <@%s>", id, component, qty, qual, caller))

	order, err := h.store.GetByID(context.Background(), id)
	if err != nil {
		respond(s, i, okEmbed(fmt.Sprintf("Commande #%d créée.", id)))
		return
	}
	slog.Info("order created", "order_id", id, "by", caller)
	respondEmbeds(s, i, []*discordgo.MessageEmbed{orderEmbed(order)})
}

// validateOrderFields checks component field constraints and returns the canonical resource name.
func validateOrderFields(component string) (string, error) {
	if component == "" {
		return "", fmt.Errorf("Le nom de la ressource ne peut pas être vide.")
	}
	canonical, ok := canonicalResource(component)
	if !ok {
		return "", fmt.Errorf("Ressource inconnue : %q.\nUtilise `/order-help` pour voir la liste des ressources disponibles.", component)
	}
	return canonical, nil
}

// openOrderModal opens the order creation modal with component encoded in the CustomID.
// openOrderModal opens the 3-field order creation modal, pre-filling component if already known.
func openOrderModal(s *discordgo.Session, i *discordgo.InteractionCreate, component, qual, qty string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "order_modal",
			Title:    "G.A.L.E.R.E - Nouvelle commande",
			Components: []discordgo.MessageComponent{
				textRow("component", "Ressource", "ex. Taranite, Hadanite, Riccite ...", component, true),
				textRow("quality", "Qualité (laisser vide si aucune qualité)", "ex. 750", qual, false),
				textRow("quantity", "Quantité (cSCU | Unités)", "ex. 150", qty, true),
			},
		},
	}); err != nil {
		slog.Debug("failed to open order modal", "err", err)
	}
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
