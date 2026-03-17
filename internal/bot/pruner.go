package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const pruneRetention = 14 * 24 * time.Hour // 2 weeks

// startPruner runs an immediate prune then repeats daily at midnight UTC.
// It returns when ctx is cancelled.
func (b *Bot) startPruner(ctx context.Context) {
	b.runPrune(ctx)

	for {
		now := time.Now().UTC()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		timer := time.NewTimer(time.Until(nextMidnight))

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			b.runPrune(ctx)
		}
	}
}

// runPrune deletes done orders older than pruneRetention and logs the result.
func (b *Bot) runPrune(ctx context.Context) {
	before := time.Now().UTC().Add(-pruneRetention)
	n, err := b.store.Prune(ctx, before)
	if err != nil {
		slog.Error("prune failed", "err", err)
		return
	}
	if n > 0 {
		slog.Info("pruned old done orders", "count", n, "older_than", before.Format("2006-01-02"))
		logAction(b.session, b.logChannelID, fmt.Sprintf("Nettoyages des commandes terminées : %d commandes (>2w)", n))
	}
}
