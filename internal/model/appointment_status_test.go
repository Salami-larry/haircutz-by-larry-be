package model

import "testing"

func TestAllowedAdminTransition(t *testing.T) {
	cases := []struct {
		from, to AppointmentStatus
		ok       bool
	}{
		{AppointmentPaid, AppointmentAcknowledged, true},
		{AppointmentAcknowledged, AppointmentCompleted, true},
		{AppointmentAcknowledged, AppointmentMissed, true},
		{AppointmentPaid, AppointmentCompleted, false},
		{AppointmentPaid, AppointmentMissed, false},
		{AppointmentBooked, AppointmentPaid, false},
		{AppointmentBooked, AppointmentAcknowledged, false},
		{AppointmentCompleted, AppointmentMissed, false},
		{AppointmentMissed, AppointmentCompleted, false},
		{AppointmentAbandoned, AppointmentPaid, false},
		{AppointmentAcknowledged, AppointmentPaid, false},
		{AppointmentPaid, AppointmentPaid, false},
	}
	for _, tc := range cases {
		got := AllowedAdminTransition(tc.from, tc.to)
		if got != tc.ok {
			t.Fatalf("%s → %s: got %v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestAdminSettable(t *testing.T) {
	if AppointmentPaid.AdminSettable() {
		t.Fatal("paid should use mark-paid, not PATCH")
	}
	if !AppointmentAcknowledged.AdminSettable() || !AppointmentCompleted.AdminSettable() || !AppointmentMissed.AdminSettable() {
		t.Fatal("ack/completed/missed should be admin-settable")
	}
}
