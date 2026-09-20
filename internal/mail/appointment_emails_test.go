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
		ID:          primitive.NewObjectID(),
		ServiceType: model.ServiceWalkIn,
		StartAt:     time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		EndAt:       time.Date(2026, 9, 20, 10, 45, 0, 0, time.UTC),
		Customer: model.AppointmentCustomer{
			Name:  "Ada",
			Email: "ada@example.com",
			Phone: "+234",
		},
		Hairstyle:       model.HairstyleSnapshot{Name: "Fade"},
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

	csubj, chtml, cplain, err := AppointmentCustomerCompletedEmail(a)
	if err != nil {
		t.Fatal(err)
	}
	if csubj == "" || !strings.Contains(chtml, "complete") || !strings.Contains(chtml, "Leave a Google review") {
		t.Fatalf("completed email incomplete")
	}
	if !strings.Contains(chtml, "kgmid") || !strings.Contains(cplain, GoogleReviewURL) {
		t.Fatalf("completed email missing Google review link")
	}

	msubj, mhtml, mplain, err := AppointmentCustomerMissedEmail(a, "http://localhost:3000/track")
	if err != nil {
		t.Fatal(err)
	}
	if msubj == "" || !strings.Contains(mhtml, "reschedule") || !strings.Contains(mplain, "Track") {
		t.Fatalf("missed email incomplete")
	}

	rsubj, rhtml, _, err := AppointmentCustomerRescheduledEmail(a, "http://localhost:3000/track")
	if err != nil {
		t.Fatal(err)
	}
	if rsubj == "" || !strings.Contains(rhtml, "rescheduled") {
		t.Fatalf("customer reschedule email incomplete")
	}
	asubj2, ahtml2, _, err := AppointmentAdminRescheduledEmail(a)
	if err != nil {
		t.Fatal(err)
	}
	if asubj2 == "" || !strings.Contains(ahtml2, "paid") {
		t.Fatalf("admin reschedule email incomplete")
	}

	rsubj2, rhtml2, rplain2, err := AppointmentCustomerReminderEmail(a, "http://localhost:3000/track")
	if err != nil {
		t.Fatal(err)
	}
	if rsubj2 == "" || !strings.Contains(rhtml2, "15 minutes") || !strings.Contains(rplain2, "Track") {
		t.Fatalf("customer reminder email incomplete")
	}
	arsubj, arhtml, _, err := AppointmentAdminReminderEmail(a)
	if err != nil {
		t.Fatal(err)
	}
	if arsubj == "" || !strings.Contains(arhtml, "ada@example.com") {
		t.Fatalf("admin reminder email incomplete")
	}
	nsubj, nhtml, nplain, err := AppointmentAdminPostTimeNagEmail(a)
	if err != nil {
		t.Fatal(err)
	}
	if nsubj == "" || !strings.Contains(nhtml, "completed") || !strings.Contains(nplain, "missed") {
		t.Fatalf("admin post-time nag email incomplete")
	}
}
