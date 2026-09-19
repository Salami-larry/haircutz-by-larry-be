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

	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
	"haircutz/backend/internal/repository"
	"haircutz/backend/internal/schedule"
)

var (
	ErrSlotUnavailable     = errors.New("that time was just taken — pick another slot")
	ErrHairstyleInactive   = errors.New("hairstyle is not available")
	ErrInvalidStartTime    = errors.New("start time is not an available slot")
	ErrServiceClosed       = errors.New("service is not available on that day")
	ErrAddressRequired     = errors.New("address is required for home service")
)

type AppointmentController struct {
	appointments *repository.AppointmentRepository
	hairstyles   *repository.HairstyleRepository
	log          *slog.Logger
	loc          *time.Location
}

func NewAppointmentController(
	appointments *repository.AppointmentRepository,
	hairstyles *repository.HairstyleRepository,
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
		appointments: appointments,
		hairstyles:   hairstyles,
		log:          log,
		loc:          loc,
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

	if err := c.assertStartIsAvailable(ctx, hs, in.ServiceType, start, end); err != nil {
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
