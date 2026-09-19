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

// BlockingForHairstyleDelete are statuses that prevent deleting a hairstyle.
func BlockingForHairstyleDelete() []AppointmentStatus {
	return []AppointmentStatus{
		AppointmentBooked,
		AppointmentPaid,
		AppointmentAcknowledged,
		AppointmentMissed,
	}
}
