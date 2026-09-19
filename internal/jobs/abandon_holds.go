package jobs

import (
	"context"
	"log/slog"
	"time"

	"haircutz/backend/internal/repository"
)

const (
	AbandonHoldInterval = 1 * time.Minute
	BookedHoldTTL       = 15 * time.Minute
)

const abandonHoldTimeout = 30 * time.Second

func BookedAbandonCutoff(now time.Time, ttl time.Duration) time.Time {
	return now.UTC().Add(-ttl)
}

// RunAbandonStaleBooked marks unpaid booked appointments older than TTL as abandoned.
func RunAbandonStaleBooked(ctx context.Context, appointments *repository.AppointmentRepository, log *slog.Logger) {
	if appointments == nil || log == nil {
		return
	}

	runOnce := func() {
		jobCtx, cancel := context.WithTimeout(ctx, abandonHoldTimeout)
		defer cancel()

		cutoff := BookedAbandonCutoff(time.Now(), BookedHoldTTL)
		n, err := appointments.AbandonStaleBooked(jobCtx, cutoff)
		if err != nil {
			log.Error("abandon stale booked failed", "err", err)
			return
		}
		if n > 0 {
			log.Info("abandoned stale booked appointments", "count", n, "cutoff", cutoff.Format(time.RFC3339))
		}
	}

	runOnce()

	ticker := time.NewTicker(AbandonHoldInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
