package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"haircutz/backend/internal/mail"
	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
	"haircutz/backend/internal/repository"
	"haircutz/backend/internal/schedule"
)

var (
	ErrSlotUnavailable         = errors.New("that time was just taken — pick another slot")
	ErrHairstyleInactive       = errors.New("hairstyle is not available")
	ErrInvalidStartTime        = errors.New("start time is not an available slot")
	ErrServiceClosed           = errors.New("service is not available on that day")
	ErrAddressRequired         = errors.New("address is required for home service")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrRescheduleNotAllowed    = errors.New("only missed appointments can be rescheduled")
	ErrSameTimeframe           = errors.New("you already have that timeframe")
	ErrRescheduleForbidden     = errors.New("tracking number or email does not match")
)

type AppointmentController struct {
	appointments    *repository.AppointmentRepository
	hairstyles      *repository.HairstyleRepository
	mail            *mail.Sender
	adminEmail      string
	clientPublicURL string
	log             *slog.Logger
	loc             *time.Location
}

func NewAppointmentController(
	appointments *repository.AppointmentRepository,
	hairstyles *repository.HairstyleRepository,
	mailer *mail.Sender,
	adminEmail string,
	clientPublicURL string,
	log *slog.Logger,
) (*AppointmentController, error) {
	if log == nil {
		log = slog.Default()
	}
	loc, err := schedule.LoadLocation()
	if err != nil {
		return nil, err
	}
	return &AppointmentController{
		appointments:    appointments,
		hairstyles:      hairstyles,
		mail:            mailer,
		adminEmail:      strings.TrimSpace(adminEmail),
		clientPublicURL: strings.TrimSuffix(clientPublicURL, "/"),
		log:             log,
		loc:             loc,
	}, nil
}

type AvailabilityResult struct {
	Date        string   `json:"date"`
	ServiceType string   `json:"serviceType"`
	DurationMin int      `json:"durationMinutes"`
	Slots       []string `json:"slots"` // RFC3339 in Africa/Lagos offset
	Closed      bool     `json:"closed"`
}

func (c *AppointmentController) Availability(
	ctx context.Context,
	dateStr string,
	hairstyleID primitive.ObjectID,
	service model.ServiceType,
) (AvailabilityResult, error) {
	day, err := time.ParseInLocation("2006-01-02", dateStr, c.loc)
	if err != nil {
		return AvailabilityResult{}, fmt.Errorf("date must be YYYY-MM-DD")
	}

	hs, err := c.hairstyles.FindByID(ctx, hairstyleID)
	if err != nil {
		return AvailabilityResult{}, err
	}
	if !hs.Active {
		return AvailabilityResult{}, ErrHairstyleInactive
	}

	window, open, err := schedule.WindowForDay(day, service, c.loc)
	if err != nil {
		return AvailabilityResult{}, err
	}
	out := AvailabilityResult{
		Date:        dateStr,
		ServiceType: string(service),
		DurationMin: hs.DurationMinutes,
		Slots:       []string{},
	}
	if !open {
		out.Closed = true
		return out, nil
	}

	busyAppts, err := c.appointments.FindBlockingInRange(ctx, window.Open.UTC(), window.Close.UTC())
	if err != nil {
		return AvailabilityResult{}, err
	}
	busy := make([]schedule.BusyInterval, 0, len(busyAppts))
	for _, a := range busyAppts {
		busy = append(busy, schedule.BusyInterval{
			Start: a.StartAt.In(c.loc),
			End:   a.EndAt.In(c.loc),
		})
	}

	var nowPtr *time.Time
	now := time.Now().In(c.loc)
	if sameCalendarDay(now, day) {
		nowPtr = &now
	}

	starts := schedule.AvailableStarts(window, time.Duration(hs.DurationMinutes)*time.Minute, busy, nowPtr)
	out.Slots = make([]string, 0, len(starts))
	for _, s := range starts {
		out.Slots = append(out.Slots, s.Format(time.RFC3339))
	}
	return out, nil
}

type CreateAppointmentInput struct {
	HairstyleID primitive.ObjectID
	ServiceType model.ServiceType
	StartAt     time.Time // any zone; normalized to Lagos
	Customer    model.AppointmentCustomer
}

func (c *AppointmentController) Create(ctx context.Context, in CreateAppointmentInput) (*model.Appointment, error) {
	if err := validateCustomer(in.ServiceType, in.Customer); err != nil {
		return nil, err
	}

	hs, err := c.hairstyles.FindByID(ctx, in.HairstyleID)
	if err != nil {
		return nil, err
	}
	if !hs.Active {
		return nil, ErrHairstyleInactive
	}

	start := in.StartAt.In(c.loc)
	// Snap to exact minute (clients may send seconds)
	start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), 0, 0, c.loc)
	duration := time.Duration(hs.DurationMinutes) * time.Minute
	end := start.Add(duration)

	if err := c.assertStartIsAvailable(ctx, hs, in.ServiceType, start, end, nil); err != nil {
		return nil, err
	}

	amount := hs.WalkInPriceKobo
	if in.ServiceType == model.ServiceHomeService {
		amount = hs.HomeServicePriceKobo
	}

	imageURL := ""
	if len(hs.ImageURLs) > 0 {
		imageURL = hs.ImageURLs[0]
	}

	now := time.Now().UTC()
	appt := &model.Appointment{
		HairstyleID: hs.ID,
		Hairstyle: model.HairstyleSnapshot{
			HairstyleID:          hs.ID,
			Name:                 hs.Name,
			DurationMinutes:      hs.DurationMinutes,
			WalkInPriceKobo:      hs.WalkInPriceKobo,
			HomeServicePriceKobo: hs.HomeServicePriceKobo,
			ImageURL:             imageURL,
		},
		ServiceType: in.ServiceType,
		StartAt:     start.UTC(),
		EndAt:       end.UTC(),
		Customer: model.AppointmentCustomer{
			Name:    strings.TrimSpace(in.Customer.Name),
			Email:   strings.ToLower(strings.TrimSpace(in.Customer.Email)),
			Phone:   strings.TrimSpace(in.Customer.Phone),
			Address: strings.TrimSpace(in.Customer.Address),
			Notes:   strings.TrimSpace(in.Customer.Notes),
		},
		Status:          model.AppointmentBooked,
		StatusHistory: []model.AppointmentStatusEvent{{
			Status: model.AppointmentBooked,
			At:     now,
			Note:   "Hold created",
		}},
		TotalAmountKobo:   amount,
		PaystackReference: "hbl_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
	}

	overlap, err := c.appointments.CountOverlapping(ctx, appt.StartAt, appt.EndAt, nil)
	if err != nil {
		return nil, err
	}
	if overlap > 0 {
		return nil, ErrSlotUnavailable
	}

	if err := c.appointments.Create(ctx, appt); err != nil {
		return nil, fmt.Errorf("create appointment: %w", err)
	}

	// Race: another hold may have landed in the same window.
	overlap, err = c.appointments.CountOverlapping(ctx, appt.StartAt, appt.EndAt, nil)
	if err != nil {
		return nil, err
	}
	if overlap > 1 {
		_ = c.appointments.Delete(ctx, appt.ID)
		return nil, ErrSlotUnavailable
	}

	return appt, nil
}

func (c *AppointmentController) assertStartIsAvailable(
	ctx context.Context,
	hs *model.Hairstyle,
	service model.ServiceType,
	start, end time.Time,
	excludeID *primitive.ObjectID,
) error {
	window, open, err := schedule.WindowForDay(start, service, c.loc)
	if err != nil {
		return err
	}
	if !open {
		return ErrServiceClosed
	}
	if start.Before(window.Open) || !end.After(start) || end.After(window.Close) {
		return ErrInvalidStartTime
	}

	now := time.Now().In(c.loc)
	if sameCalendarDay(now, start) && !start.After(now) {
		return ErrInvalidStartTime
	}

	busyAppts, err := c.appointments.FindBlockingInRange(ctx, window.Open.UTC(), window.Close.UTC())
	if err != nil {
		return err
	}
	busy := make([]schedule.BusyInterval, 0, len(busyAppts))
	for _, a := range busyAppts {
		if excludeID != nil && a.ID == *excludeID {
			continue
		}
		busy = append(busy, schedule.BusyInterval{
			Start: a.StartAt.In(c.loc),
			End:   a.EndAt.In(c.loc),
		})
	}

	var nowPtr *time.Time
	if sameCalendarDay(now, start) {
		nowPtr = &now
	}
	starts := schedule.AvailableStarts(window, time.Duration(hs.DurationMinutes)*time.Minute, busy, nowPtr)
	for _, s := range starts {
		if s.Equal(start) {
			return nil
		}
	}
	return ErrInvalidStartTime
}

func (c *AppointmentController) ListAdmin(ctx context.Context, f repository.AppointmentListFilter) (pagination.Paginated[model.Appointment], error) {
	items, total, err := c.appointments.List(ctx, f)
	if err != nil {
		return pagination.Paginated[model.Appointment]{}, err
	}
	return pagination.NewPaginated(items, total, f.Page, f.PageSize), nil
}

func (c *AppointmentController) Get(ctx context.Context, id primitive.ObjectID) (*model.Appointment, error) {
	return c.appointments.FindByID(ctx, id)
}

type TrackAppointmentInput struct {
	TrackingNumber string
	Email          string
}

type TrackAppointmentView struct {
	ID              string                      `json:"id"`
	TrackingNumber  string                      `json:"trackingNumber"`
	Status          model.AppointmentStatus     `json:"status"`
	StatusHistory   []model.AppointmentStatusEvent `json:"statusHistory"`
	ServiceType     model.ServiceType           `json:"serviceType"`
	StartAt         time.Time                   `json:"startAt"`
	EndAt           time.Time                   `json:"endAt"`
	TotalAmountKobo int64                       `json:"totalAmountKobo"`
	Hairstyle       model.HairstyleSnapshot     `json:"hairstyle"`
	CustomerName    string                      `json:"customerName"`
	CanReschedule   bool                        `json:"canReschedule"`
}

func (c *AppointmentController) Track(ctx context.Context, in TrackAppointmentInput) (TrackAppointmentView, error) {
	tracking := strings.TrimSpace(in.TrackingNumber)
	email := strings.TrimSpace(in.Email)
	if tracking == "" || email == "" {
		return TrackAppointmentView{}, fmt.Errorf("trackingNumber and email are required")
	}

	appt, err := c.appointments.FindForTrack(ctx, tracking, email)
	if err != nil {
		return TrackAppointmentView{}, err
	}

	return TrackAppointmentView{
		ID:              appt.ID.Hex(),
		TrackingNumber:  appt.TrackingNumber,
		Status:          appt.Status,
		StatusHistory:   appt.StatusHistory,
		ServiceType:     appt.ServiceType,
		StartAt:         appt.StartAt,
		EndAt:           appt.EndAt,
		TotalAmountKobo: appt.TotalAmountKobo,
		Hairstyle:       appt.Hairstyle,
		CustomerName:    appt.Customer.Name,
		CanReschedule:   appt.MayReschedule(),
	}, nil
}

type RescheduleAppointmentInput struct {
	TrackingNumber string
	Email          string
	StartAt        time.Time
}

func (c *AppointmentController) Reschedule(ctx context.Context, id primitive.ObjectID, in RescheduleAppointmentInput) (*model.Appointment, error) {
	tracking := strings.TrimSpace(in.TrackingNumber)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if tracking == "" || email == "" {
		return nil, fmt.Errorf("trackingNumber and email are required")
	}

	appt, err := c.appointments.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !appt.MayReschedule() {
		return nil, ErrRescheduleNotAllowed
	}
	if !strings.EqualFold(strings.TrimSpace(appt.TrackingNumber), tracking) ||
		!strings.EqualFold(strings.TrimSpace(appt.Customer.Email), email) {
		return nil, ErrRescheduleForbidden
	}

	hs, err := c.hairstyles.FindByID(ctx, appt.HairstyleID)
	if err != nil {
		return nil, err
	}
	if !hs.Active {
		return nil, ErrHairstyleInactive
	}

	start := in.StartAt.In(c.loc)
	start = time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), 0, 0, c.loc)
	duration := time.Duration(hs.DurationMinutes) * time.Minute
	end := start.Add(duration)

	oldStart := appt.StartAt.In(c.loc)
	oldEnd := appt.EndAt.In(c.loc)
	oldStart = time.Date(oldStart.Year(), oldStart.Month(), oldStart.Day(), oldStart.Hour(), oldStart.Minute(), 0, 0, c.loc)
	oldEnd = time.Date(oldEnd.Year(), oldEnd.Month(), oldEnd.Day(), oldEnd.Hour(), oldEnd.Minute(), 0, 0, c.loc)
	if start.Equal(oldStart) && end.Equal(oldEnd) {
		return nil, ErrSameTimeframe
	}

	excludeID := appt.ID
	if err := c.assertStartIsAvailable(ctx, hs, appt.ServiceType, start, end, &excludeID); err != nil {
		return nil, err
	}

	overlap, err := c.appointments.CountOverlapping(ctx, start.UTC(), end.UTC(), &excludeID)
	if err != nil {
		return nil, err
	}
	if overlap > 0 {
		return nil, ErrSlotUnavailable
	}

	now := time.Now().UTC()
	appt.StartAt = start.UTC()
	appt.EndAt = end.UTC()
	appt.Status = model.AppointmentPaid
	appt.AbandonedAt = nil
	if appt.PaidAt == nil {
		appt.PaidAt = &now
	}
	appt.StatusHistory = append(appt.StatusHistory, model.AppointmentStatusEvent{
		Status: model.AppointmentPaid,
		At:     now,
		Note:   "Rescheduled after missed (free)",
	})

	if err := c.appointments.Update(ctx, appt); err != nil {
		return nil, fmt.Errorf("reschedule appointment: %w", err)
	}

	emailAppt := *appt
	go func() {
		if err := c.sendRescheduleEmails(&emailAppt); err != nil {
			c.log.Error("reschedule emails failed", "appointmentId", emailAppt.ID.Hex(), "err", err)
			return
		}
		c.log.Info("reschedule emails sent", "appointmentId", emailAppt.ID.Hex())
	}()

	return appt, nil
}

func (c *AppointmentController) sendRescheduleEmails(a *model.Appointment) error {
	if c.mail == nil {
		c.log.Info("skipping reschedule email: smtp not configured", "appointmentId", a.ID.Hex())
		return nil
	}
	trackURL := c.clientPublicURL + "/track"
	subj, html, plain, err := mail.AppointmentCustomerRescheduledEmail(a, trackURL)
	if err != nil {
		return fmt.Errorf("customer reschedule email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(a.Customer.Email, subj, plain, html); err != nil {
		return fmt.Errorf("send customer reschedule email: %w", err)
	}
	if c.adminEmail != "" {
		asubj, ahtml, aplain, err := mail.AppointmentAdminRescheduledEmail(a)
		if err != nil {
			return fmt.Errorf("admin reschedule email: %w", err)
		}
		if err := c.mail.SendHTMLWithPlainAlt(c.adminEmail, asubj, aplain, ahtml); err != nil {
			return fmt.Errorf("send admin reschedule email: %w", err)
		}
	}
	return nil
}

type UpdateAppointmentStatusInput struct {
	Status model.AppointmentStatus
	Note   string
}

func (c *AppointmentController) UpdateStatus(ctx context.Context, id primitive.ObjectID, in UpdateAppointmentStatusInput) (*model.Appointment, error) {
	if !in.Status.AdminSettable() {
		return nil, fmt.Errorf("status %q cannot be set via this endpoint", in.Status)
	}

	appt, err := c.appointments.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !model.AllowedAdminTransition(appt.Status, in.Status) {
		return nil, ErrInvalidStatusTransition
	}

	previous := appt.Status
	now := time.Now().UTC()
	note := strings.TrimSpace(in.Note)
	if note == "" {
		switch in.Status {
		case model.AppointmentAcknowledged:
			note = "Acknowledged by admin"
		case model.AppointmentCompleted:
			note = "Marked completed"
		case model.AppointmentMissed:
			note = "Marked missed"
		}
	}

	appt.Status = in.Status
	appt.StatusHistory = append(appt.StatusHistory, model.AppointmentStatusEvent{
		Status: in.Status,
		At:     now,
		Note:   note,
	})

	if err := c.appointments.Update(ctx, appt); err != nil {
		return nil, fmt.Errorf("update appointment status: %w", err)
	}

	emailAppt := *appt
	go func() {
		if err := c.sendOutcomeEmail(&emailAppt); err != nil {
			c.log.Error("appointment status email failed",
				"appointmentId", emailAppt.ID.Hex(),
				"from", previous,
				"to", in.Status,
				"err", err,
			)
			return
		}
		if in.Status == model.AppointmentCompleted || in.Status == model.AppointmentMissed {
			c.log.Info("appointment status email sent",
				"appointmentId", emailAppt.ID.Hex(),
				"to", in.Status,
			)
		}
	}()

	return appt, nil
}

func (c *AppointmentController) sendOutcomeEmail(a *model.Appointment) error {
	switch a.Status {
	case model.AppointmentCompleted, model.AppointmentMissed:
		// continue
	default:
		return nil // acknowledged: no email
	}
	if c.mail == nil {
		c.log.Info("skipping outcome email: smtp not configured", "appointmentId", a.ID.Hex())
		return nil
	}

	trackURL := c.clientPublicURL + "/track"
	var subject, html, plain string
	var err error
	switch a.Status {
	case model.AppointmentCompleted:
		subject, html, plain, err = mail.AppointmentCustomerCompletedEmail(a)
	case model.AppointmentMissed:
		subject, html, plain, err = mail.AppointmentCustomerMissedEmail(a, trackURL)
	}
	if err != nil {
		return fmt.Errorf("render outcome email: %w", err)
	}
	if err := c.mail.SendHTMLWithPlainAlt(a.Customer.Email, subject, plain, html); err != nil {
		return fmt.Errorf("send outcome email: %w", err)
	}
	return nil
}

func validateCustomer(service model.ServiceType, cust model.AppointmentCustomer) error {
	if strings.TrimSpace(cust.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(cust.Email) == "" {
		return fmt.Errorf("email is required")
	}
	if strings.TrimSpace(cust.Phone) == "" {
		return fmt.Errorf("phone is required")
	}
	if service == model.ServiceHomeService && strings.TrimSpace(cust.Address) == "" {
		return ErrAddressRequired
	}
	return nil
}

func sameCalendarDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func IsAppointmentNotFound(err error) bool {
	return errors.Is(err, repository.ErrAppointmentNotFound)
}

// AppointmentDeleteGuard adapts the appointment repo for hairstyle delete checks.
type AppointmentDeleteGuard struct {
	repo *repository.AppointmentRepository
}

func NewAppointmentDeleteGuard(repo *repository.AppointmentRepository) *AppointmentDeleteGuard {
	return &AppointmentDeleteGuard{repo: repo}
}

func (g *AppointmentDeleteGuard) HasBlockingAppointments(ctx context.Context, hairstyleID primitive.ObjectID) (bool, error) {
	if g == nil || g.repo == nil {
		return false, nil
	}
	return g.repo.HasBlockingAppointments(ctx, hairstyleID)
}
