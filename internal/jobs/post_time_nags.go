package jobs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"haircutz/backend/internal/mail"
	"haircutz/backend/internal/repository"
)

const (
	PostTimeNagCheckInterval = 1 * time.Minute
	PostTimeNagInterval      = 30 * time.Minute
)

const postTimeNagJobTimeout = 45 * time.Second

// RunPostTimeNags emails admin every PostTimeNagInterval after endAt for paid/acknowledged
// appointments until they become completed or missed.
func RunPostTimeNags(
	ctx context.Context,
	appointments *repository.AppointmentRepository,
	mailer *mail.Sender,
	adminEmail string,
	log *slog.Logger,
) {
	if appointments == nil || log == nil {
		return
	}

	runOnce := func() {
		jobCtx, cancel := context.WithTimeout(ctx, postTimeNagJobTimeout)
		defer cancel()

		now := time.Now().UTC()
		items, err := appointments.FindDuePostTimeNags(jobCtx, now, PostTimeNagInterval)
		if err != nil {
			log.Error("find due post-time nags failed", "err", err)
			return
		}
		for i := range items {
			a := &items[i]
			claimed, err := appointments.ClaimPostTimeNag(jobCtx, a.ID, now, PostTimeNagInterval)
			if err != nil {
				log.Error("claim post-time nag failed", "err", err, "id", a.ID.Hex())
				continue
			}
			if !claimed {
				continue
			}

			if mailer == nil || strings.TrimSpace(adminEmail) == "" {
				log.Info("post-time nag claimed; smtp/admin not configured", "id", a.ID.Hex(), "tracking", a.TrackingNumber)
				continue
			}

			subj, html, plain, err := mail.AppointmentAdminPostTimeNagEmail(a)
			if err != nil {
				log.Error("post-time nag template failed", "err", err, "id", a.ID.Hex())
				continue
			}
			if err := mailer.SendHTMLWithPlainAlt(adminEmail, subj, plain, html); err != nil {
				log.Error("send post-time nag failed", "err", err, "id", a.ID.Hex())
				continue
			}

			log.Info("post-time nag sent", "id", a.ID.Hex(), "tracking", a.TrackingNumber)
		}
	}

	runOnce()

	ticker := time.NewTicker(PostTimeNagCheckInterval)
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
