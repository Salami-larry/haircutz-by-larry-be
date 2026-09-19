package mail

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"haircutz/backend/internal/model"
)

func TestAppointmentPaidEmailTemplates(t *testing.T) {
	a := &model.Appointment{
		ID:        primitive.NewObjectID(),
		ServiceType: model.ServiceWalkIn,
		StartAt:   time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		EndAt:     time.Date(2026, 9, 20, 10, 45, 0, 0, time.UTC),
		Customer: model.AppointmentCustomer{
			Name:  "Ada",
			Email: "ada@example.com",
			Phone: "+234",
		},
		Hairstyle: model.HairstyleSnapshot{Name: "Fade"},
		TotalAmountKobo: 500000,
		TrackingNumber:  "HBL-20260920-ABC123",
	}

	subj, html, plain, err := AppointmentCustomerPaidEmail(a, "http://localhost:3000/track")
	if err != nil {
		t.Fatal(err)
	}
	if subj == "" || !strings.Contains(html, "HBL-20260920-ABC123") || !strings.Contains(plain, "Fade") {
		t.Fatalf("customer email incomplete")
	}

	asubj, ahtml, _, err := AppointmentAdminPaidEmail(a)
	if err != nil {
		t.Fatal(err)
	}
	if asubj == "" || !strings.Contains(ahtml, "ada@example.com") {
		t.Fatalf("admin email incomplete")
	}
}
