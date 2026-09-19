package model

import (
	"crypto/rand"
	"fmt"
	"time"
)

func NewTrackingNumber(now time.Time) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return fmt.Sprintf("HBL-%s-%s", now.UTC().Format("20060102"), string(b))
}

func (a *Appointment) MayAbandon() bool {
	return a.Status == AppointmentBooked
}

func (a *Appointment) MayMarkPaid() bool {
	return a.Status == AppointmentBooked || a.Status == AppointmentAbandoned
}
