package bot

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const recapHour = 9 // 09:00 UTC

// startRecap fires the daily recap at recapHour UTC.
// It returns when ctx is cancelled.
func (b *Bot) startRecap(ctx context.Context) {
	if b.recapChannelID == "" {
		slog.Info("recap disabled (RECAP_CHANNEL_ID not set)")
		return
	}

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), recapHour, 0, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		wait := time.Until(next)
		slog.Info("recap: next scheduled", "at", next.Format(time.RFC3339), "in", wait.Truncate(time.Second))
		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			b.sendRecap(ctx)
		}
	}
}

// sendRecap builds and sends the daily recap message.
func (b *Bot) sendRecap(ctx context.Context) {
	pending, err := b.store.ListPending(ctx)
	if err != nil {
		slog.Error("recap: ListPending failed", "err", err)
		return
	}
	for i, j := 0, len(pending)-1; i < j; i, j = i+1, j-1 {
		pending[i], pending[j] = pending[j], pending[i]
	}

	if len(pending) == 0 {
		slog.Info("recap: no pending orders, skipping")
		return
	}

	done, err := b.store.ListDone(ctx)
	if err != nil {
		slog.Error("recap: ListDone failed", "err", err)
		return
	}

	// --- Build message ---
	var msg strings.Builder

	msg.WriteString("# Recap quotidien des commandes\n\n")

	// Count by status
	var ordered, ready int
	for _, o := range pending {
		switch o.Status {
		case "ready":
			ready++
		default:
			ordered++
		}
	}

	msg.WriteString(fmt.Sprintf("> **%d** commande(s) en cours  |  ", len(pending)))
	msg.WriteString(fmt.Sprintf("**%d** ordered  /  **%d** ready\n", ordered, ready))

	// --- Markdown table ---
	msg.WriteString("### Dernieres commandes passées\n")
	msg.WriteString("```\n")
	msg.WriteString(fmt.Sprintf("%-5s %-15s %-8s %7s  %s\n", "ID", "Ressource", "Statut", "Qty", "Qualite"))
	msg.WriteString(fmt.Sprintf("%-5s %-15s %-8s %7s  %s\n", "-----", "---------------", "--------", "-------", "-------"))

	const maxRows = 10
	for i, o := range pending {
		if i >= maxRows {
			msg.WriteString(fmt.Sprintf("%-5s ... +%d commande(s)\n", "", len(pending)-maxRows))
			break
		}
		status := "ordered"
		if o.Status == "ready" {
			status = "ready"
		}
		comp := o.Component
		if len(comp) > 15 {
			comp = comp[:14] + "."
		}
		quality := "-"
		if o.MinQuality != 0 {
			quality = fmt.Sprintf("%d", o.MinQuality)
		}
		msg.WriteString(fmt.Sprintf("#%-4d %-15s %-8s %7d  %s\n",
			o.ID, comp, status, o.Quantity, quality))
	}
	msg.WriteString("```\n\n")

	// --- Send text message ---
	content := msg.String()
	_, err = b.session.ChannelMessageSend(b.recapChannelID, content)
	if err != nil {
		slog.Error("recap: failed to send text message", "err", err)
	}

	// --- Generate and send chart PNG ---
	pngData, err := generateRecapChart(pending, done)
	if err != nil {
		slog.Error("recap: chart generation failed", "err", err)
		return
	}

	_, err = b.session.ChannelFileSend(b.recapChannelID, "recap_chart.png", bytes.NewReader(pngData))
	if err != nil {
		slog.Error("recap: failed to send chart", "err", err)
	}

	slog.Info("daily recap sent", "pending", len(pending), "done", len(done), "channel", b.recapChannelID)
}
