package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"haircutz/backend/internal/mail"
	"haircutz/backend/internal/model"
	"haircutz/backend/internal/paystack"
	"haircutz/backend/internal/repository"
)

var (
	ErrAbandonEmailMismatch = errors.New("email does not match appointment")
	ErrAbandonConflict      = errors.New("appointment cannot be abandoned")
	ErrNotAwaitingPayment   = errors.New("appointment is not awaiting payment")
	ErrPaystackNotConfigured = errors.New("paystack is not configured")
)

func IsAbandonEmailMismatch(err error) bool { return errors.Is(err, ErrAbandonEmailMismatch) }
func IsAbandonConflict(err error) bool      { return errors.Is(err, ErrAbandonConflict) }

type PaymentController struct {
	appointments    *repository.AppointmentRepository
	paystack        *paystack.Client
	mail            *mail.Sender
	adminEmail      string
	clientPublicURL string
	log             *slog.Logger
}

func NewPaymentController(
	appointments *repository.AppointmentRepository,
	ps *paystack.Client,
	mailer *mail.Sender,
	adminEmail string,
	clientPublicURL string,
	log *slog.Logger,
) *PaymentController {
	if log == nil {
		log = slog.Default()
	}
	return &PaymentController{
		appointments:    appointments,
		paystack:        ps,
		mail:            mailer,
		adminEmail:      adminEmail,
		clientPublicURL: strings.TrimSuffix(clientPublicURL, "/"),
		log:             log,
	}
}

type InitializePaymentResult struct {
	AccessCode       string `json:"accessCode"`
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	Reference        string `json:"reference"`
}

func (c *PaymentController) Initialize(ctx context.Context, appointmentID primitive.ObjectID) (InitializePaymentResult, error) {
	if c.paystack == nil {
		return InitializePaymentResult{}, ErrPaystackNotConfigured
	}

	appt, err := c.appointments.FindByID(ctx, appointmentID)
	if err != nil {
		return InitializePaymentResult{}, err
	}

	switch appt.Status {
	case model.AppointmentBooked:
		// ok
	case model.AppointmentAbandoned:
		now := time.Now().UTC()
		appt.Status = model.AppointmentBooked
		appt.AbandonedAt = nil
		appt.StatusHistory = append(appt.StatusHistory, model.AppointmentStatusEvent{
			Status: model.AppointmentBooked,
			At:     now,
			Note:   "Payment resumed",
		})
		if err := c.appointments.Update(ctx, appt); err != nil {
			return InitializePaymentResult{}, err
		}
	default:
		return InitializePaymentResult{}, ErrNotAwaitingPayment
	}

	init, err := c.paystack.InitializeTransaction(ctx, appt.Customer.Email, appt.TotalAmountKobo, appt.PaystackReference)
	if err != nil {
		return InitializePaymentResult{}, fmt.Errorf("initialize paystack: %w", err)
	}

	if init.Reference != "" && init.Reference != appt.PaystackReference {
		appt.PaystackReference = init.Reference
		if err := c.appointments.Update(ctx, appt); err != nil {
			return InitializePaymentResult{}, err
		}
	}

	return InitializePaymentResult{
		AccessCode:       init.AccessCode,
		AuthorizationURL: init.AuthorizationURL,
		Reference:        appt.PaystackReference,
	}, nil
}

type VerifyPaymentResult struct {
	AppointmentID  string                   `json:"appointmentId"`
	Status         model.AppointmentStatus  `json:"status"`
	TrackingNumber string                   `json:"trackingNumber,omitempty"`
	PaystackStatus string                   `json:"paystackStatus,omitempty"`
	CustomerEmail  string                   `json:"customerEmail"`
}

func (c *PaymentController) Verify(ctx context.Context, reference string) (VerifyPaymentResult, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return VerifyPaymentResult{}, fmt.Errorf("reference is required")
	}

	appt, err := c.appointments.FindByPaystackReference(ctx, reference)
	if err != nil {
		return VerifyPaymentResult{}, err
	}

	out := VerifyPaymentResult{
		AppointmentID:  appt.ID.Hex(),
		Status:         appt.Status,
		TrackingNumber: appt.TrackingNumber,
		CustomerEmail:  appt.Customer.Email,
	}

	if c.paystack == nil {
		return out, nil
	}

	v, err := c.paystack.VerifyTransaction(ctx, reference)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("verify paystack: %w", err)
	}
	out.PaystackStatus = v.Status

	if strings.EqualFold(v.Status, "success") {
		switch appt.Status {
		case model.AppointmentBooked, model.AppointmentAbandoned:
			if v.Amount > 0 && v.Amount != appt.TotalAmountKobo {
				return VerifyPaymentResult{}, fmt.Errorf("payment amount mismatch")
			}
			if err := c.HandleChargeSuccess(ctx, reference, v.Amount); err != nil {
				return VerifyPaymentResult{}, err
			}
			appt, err = c.appointments.FindByPaystackReference(ctx, reference)
			if err != nil {
				return VerifyPaymentResult{}, err
			}
			out.Status = appt.Status
			out.TrackingNumber = appt.TrackingNumber
		}
	}

	return out, nil
}

type AbandonPaymentInput struct {
	Reference string
	Email     string
}

func (c *PaymentController) Abandon(ctx context.Context, in AbandonPaymentInput) error {
	reference := strings.TrimSpace(in.Reference)
	email := strings.TrimSpace(in.Email)
	if reference == "" {
		return fmt.Errorf("reference is required")
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}

	appt, err := c.appointments.FindByPaystackReference(ctx, reference)
	if err != nil {
		return err
	}

	if appt.Status == model.AppointmentAbandoned {
		return nil
	}
	if !appt.MayAbandon() {
		return ErrAbandonConflict
	}
	if !strings.EqualFold(strings.TrimSpace(appt.Customer.Email), email) {
		return ErrAbandonEmailMismatch
	}

	now := time.Now().UTC()
	appt.Status = model.AppointmentAbandoned
	appt.AbandonedAt = &now
	appt.StatusHistory = append(appt.StatusHistory, model.AppointmentStatusEvent{
		Status: model.AppointmentAbandoned,
		At:     now,
		Note:   "Payment cancelled by customer",
	})
	return c.appointments.Update(ctx, appt)
}

func (c *PaymentController) HandleChargeSuccess(ctx context.Context, reference string, amountKobo int64) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return fmt.Errorf("reference is required")
	}

	appt, err := c.appointments.FindByPaystackReference(ctx, reference)
	if err != nil {
		return err
	}

	if amountKobo > 0 && amountKobo != appt.TotalAmountKobo {
		return fmt.Errorf("payment amount mismatch")
	}

	now := time.Now().UTC()
	changed := markAppointmentPaidIfUnpaid(appt, now)
	if err := c.appointments.Update(ctx, appt); err != nil {
		return err
	}
	if changed {
		c.log.Info("appointment marked paid", "appointmentId", appt.ID.Hex(), "reference", reference)
	}
	return c.maybeSendPaymentEmails(ctx, appt)
}

// MarkPaidAdmin manually marks booked/abandoned as paid (webhook fallback).
func (c *PaymentController) MarkPaidAdmin(ctx context.Context, id primitive.ObjectID, note string) (*model.Appointment, error) {
	appt, err := c.appointments.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !appt.MayMarkPaid() {
		return nil, ErrNotAwaitingPayment
	}

	now := time.Now().UTC()
	markAppointmentPaidIfUnpaid(appt, now)
	if n := strings.TrimSpace(note); n != "" && len(appt.StatusHistory) > 0 {
		appt.StatusHistory[len(appt.StatusHistory)-1].Note = n
	} else if len(appt.StatusHistory) > 0 {
		last := &appt.StatusHistory[len(appt.StatusHistory)-1]
		if last.Note == "" || last.Note == "Payment confirmed" {
			last.Note = "Marked paid by admin"
		}
	}

	if err := c.appointments.Update(ctx, appt); err != nil {
		return nil, err
	}
	if err := c.maybeSendPaymentEmails(ctx, appt); err != nil {
		c.log.Warn("paid emails after admin mark failed", "err", err, "appointmentId", appt.ID.Hex())
	}
	return appt, nil
}

func markAppointmentPaidIfUnpaid(a *model.Appointment, now time.Time) bool {
	changed := false
	switch a.Status {
	case model.AppointmentBooked, model.AppointmentAbandoned:
		a.Status = model.AppointmentPaid
		a.AbandonedAt = nil
		a.PaidAt = &now
		a.StatusHistory = append(a.StatusHistory, model.AppointmentStatusEvent{
			Status: model.AppointmentPaid,
			At:     now,
			Note:   "Payment confirmed",
		})
		changed = true
	default:
		if a.PaidAt == nil {
			a.PaidAt = &now
		}
	}
	if strings.TrimSpace(a.TrackingNumber) == "" {
		a.TrackingNumber = model.NewTrackingNumber(now)
	}
	return changed
}

func (c *PaymentController) maybeSendPaymentEmails(ctx context.Context, a *model.Appointment) error {
	if a.Notifications.EmailedAt != nil {
		return nil
	}
	now := time.Now().UTC()

	if c.mail == nil {
		a.Notifications.EmailedAt = &now
		return c.appointments.Update(ctx, a)
	}

	trackURL := c.clientPublicURL + "/track"
	subj, html, plain, err := mail.AppointmentCustomerPaidEmail(a, trackURL)
	if err != nil {
		return fmt.Errorf("customer paid email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(a.Customer.Email, subj, plain, html); err != nil {
		return fmt.Errorf("send customer paid email: %w", err)
	}

	if c.adminEmail != "" {
		asubj, ahtml, aplain, err := mail.AppointmentAdminPaidEmail(a)
		if err != nil {
			return fmt.Errorf("admin paid email: %w", err)
		}
		if err := c.mail.SendHTMLWithPlainAlt(c.adminEmail, asubj, aplain, ahtml); err != nil {
			return fmt.Errorf("send admin paid email: %w", err)
		}
	}

	a.Notifications.EmailedAt = &now
	return c.appointments.Update(ctx, a)
}
