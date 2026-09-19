package model

import "fmt"

type ServiceType string

const (
	ServiceWalkIn      ServiceType = "walk_in"
	ServiceHomeService ServiceType = "home_service"
)

func (s ServiceType) Valid() bool {
	switch s {
	case ServiceWalkIn, ServiceHomeService:
		return true
	default:
		return false
	}
}

func ParseServiceType(raw string) (ServiceType, error) {
	s := ServiceType(raw)
	if !s.Valid() {
		return "", fmt.Errorf("invalid service type %q", raw)
	}
	return s, nil
}

type AppointmentStatus string

const (
	AppointmentBooked       AppointmentStatus = "booked"
	AppointmentPaid         AppointmentStatus = "paid"
	AppointmentAcknowledged AppointmentStatus = "acknowledged"
	AppointmentCompleted    AppointmentStatus = "completed"
	AppointmentMissed       AppointmentStatus = "missed"
	AppointmentAbandoned    AppointmentStatus = "abandoned"
)

func (s AppointmentStatus) Valid() bool {
	switch s {
	case AppointmentBooked, AppointmentPaid, AppointmentAcknowledged,
		AppointmentCompleted, AppointmentMissed, AppointmentAbandoned:
		return true
	default:
		return false
	}
}

// BlockingStatuses occupy a calendar slot.
func BlockingStatuses() []AppointmentStatus {
	return []AppointmentStatus{
		AppointmentBooked,
		AppointmentPaid,
		AppointmentAcknowledged,
	}
}

func (s AppointmentStatus) BlocksSlot() bool {
	switch s {
	case AppointmentBooked, AppointmentPaid, AppointmentAcknowledged:
		return true
	default:
		return false
	}
}

// AdminSettable is true for statuses an admin may set via PATCH /status
// (paid uses mark-paid / Paystack instead).
func (s AppointmentStatus) AdminSettable() bool {
	switch s {
	case AppointmentAcknowledged, AppointmentCompleted, AppointmentMissed:
		return true
	default:
		return false
	}
}

// AllowedAdminTransition is the Phase 6 allow-list:
// paid → acknowledged; acknowledged → completed|missed.
func AllowedAdminTransition(current, next AppointmentStatus) bool {
	if !next.Valid() || !next.AdminSettable() || current == next {
		return false
	}
	switch current {
	case AppointmentPaid:
		return next == AppointmentAcknowledged
	case AppointmentAcknowledged:
		return next == AppointmentCompleted || next == AppointmentMissed
	default:
		return false
	}
}

// NextAdminStatuses returns UI-friendly next steps from current.
func NextAdminStatuses(current AppointmentStatus) []AppointmentStatus {
	switch current {
	case AppointmentPaid:
		return []AppointmentStatus{AppointmentAcknowledged}
	case AppointmentAcknowledged:
		return []AppointmentStatus{AppointmentCompleted, AppointmentMissed}
	default:
		return nil
	}
}

// BlockingForHairstyleDelete are statuses that prevent deleting a hairstyle.
func BlockingForHairstyleDelete() []AppointmentStatus {
	return []AppointmentStatus{
		AppointmentBooked,
		AppointmentPaid,
		AppointmentAcknowledged,
		AppointmentMissed,
	}
}
