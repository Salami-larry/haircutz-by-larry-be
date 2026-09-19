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
	ReminderInterval   = 1 * time.Minute
	ReminderLeadWindow = 15 * time.Minute
	ReminderMinLead    = 15 * time.Minute
)

const reminderJobTimeout = 45 * time.Second

// RunAppointmentReminders emails customer + admin ~15m before start for paid/acknowledged
// appointments that were booked with at least ReminderMinLead notice.
func RunAppointmentReminders(
	ctx context.Context,
	appointments *repository.AppointmentRepository,
	mailer *mail.Sender,
	adminEmail string,
	clientPublicURL string,
	log *slog.Logger,
) {
	if appointments == nil || log == nil {
		return
	}

	trackURL := strings.TrimRight(clientPublicURL, "/") + "/track"

	runOnce := func() {
		jobCtx, cancel := context.WithTimeout(ctx, reminderJobTimeout)
		defer cancel()

		now := time.Now().UTC()
		items, err := appointments.FindDueReminders(jobCtx, now, ReminderLeadWindow, ReminderMinLead)
		if err != nil {
			log.Error("find due reminders failed", "err", err)
			return
		}
		for i := range items {
			a := &items[i]
			claimed, err := appointments.ClaimReminderSent(jobCtx, a.ID, now)
			if err != nil {
				log.Error("claim reminder failed", "err", err, "id", a.ID.Hex())
				continue
			}
			if !claimed {
				continue
			}

			if mailer == nil {
				log.Info("reminder claimed; smtp not configured", "id", a.ID.Hex(), "tracking", a.TrackingNumber)
				continue
			}

			subj, html, plain, err := mail.AppointmentCustomerReminderEmail(a, trackURL)
			if err != nil {
				log.Error("customer reminder template failed", "err", err, "id", a.ID.Hex())
				continue
			}
			if err := mailer.SendHTMLWithPlainAlt(a.Customer.Email, subj, plain, html); err != nil {
				log.Error("send customer reminder failed", "err", err, "id", a.ID.Hex())
			}

			if strings.TrimSpace(adminEmail) != "" {
				asubj, ahtml, aplain, err := mail.AppointmentAdminReminderEmail(a)
				if err != nil {
					log.Error("admin reminder template failed", "err", err, "id", a.ID.Hex())
					continue
				}
				if err := mailer.SendHTMLWithPlainAlt(adminEmail, asubj, aplain, ahtml); err != nil {
					log.Error("send admin reminder failed", "err", err, "id", a.ID.Hex())
				}
			}

			log.Info("appointment reminder sent", "id", a.ID.Hex(), "tracking", a.TrackingNumber)
		}
	}

	runOnce()

	ticker := time.NewTicker(ReminderInterval)
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
