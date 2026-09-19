package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AppointmentCustomer struct {
	Name    string `bson:"name" json:"name"`
	Email   string `bson:"email" json:"email"`
	Phone   string `bson:"phone" json:"phone"`
	Address string `bson:"address,omitempty" json:"address,omitempty"`
	Notes   string `bson:"notes,omitempty" json:"notes,omitempty"`
}

type HairstyleSnapshot struct {
	HairstyleID          primitive.ObjectID `bson:"hairstyleId" json:"hairstyleId"`
	Name                 string             `bson:"name" json:"name"`
	DurationMinutes      int                `bson:"durationMinutes" json:"durationMinutes"`
	WalkInPriceKobo      int64              `bson:"walkInPriceKobo" json:"walkInPriceKobo"`
	HomeServicePriceKobo int64              `bson:"homeServicePriceKobo" json:"homeServicePriceKobo"`
	ImageURL             string             `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
}

type AppointmentStatusEvent struct {
	Status AppointmentStatus `bson:"status" json:"status"`
	At     time.Time         `bson:"at" json:"at"`
	Note   string            `bson:"note,omitempty" json:"note,omitempty"`
}

type AppointmentNotifications struct {
	EmailedAt      *time.Time `bson:"emailedAt,omitempty" json:"emailedAt,omitempty"`
	ReminderSentAt *time.Time `bson:"reminderSentAt,omitempty" json:"reminderSentAt,omitempty"`
	PostTimeNagAt  *time.Time `bson:"postTimeNagAt,omitempty" json:"postTimeNagAt,omitempty"`
}

type Appointment struct {
	ID                primitive.ObjectID       `bson:"_id,omitempty" json:"id"`
	HairstyleID       primitive.ObjectID       `bson:"hairstyleId" json:"hairstyleId"`
	Hairstyle         HairstyleSnapshot        `bson:"hairstyle" json:"hairstyle"`
	ServiceType       ServiceType              `bson:"serviceType" json:"serviceType"`
	StartAt           time.Time                `bson:"startAt" json:"startAt"`
	EndAt             time.Time                `bson:"endAt" json:"endAt"`
	Customer          AppointmentCustomer      `bson:"customer" json:"customer"`
	Status            AppointmentStatus        `bson:"status" json:"status"`
	StatusHistory     []AppointmentStatusEvent `bson:"statusHistory" json:"statusHistory"`
	TotalAmountKobo   int64                    `bson:"totalAmountKobo" json:"totalAmountKobo"`
	PaystackReference string                   `bson:"paystackReference,omitempty" json:"paystackReference,omitempty"`
	TrackingNumber    string                   `bson:"trackingNumber,omitempty" json:"trackingNumber,omitempty"`
	PaidAt            *time.Time               `bson:"paidAt,omitempty" json:"paidAt,omitempty"`
	AbandonedAt       *time.Time               `bson:"abandonedAt,omitempty" json:"abandonedAt,omitempty"`
	Notifications     AppointmentNotifications `bson:"notifications,omitempty" json:"notifications,omitempty"`
	CreatedAt         time.Time                `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time                `bson:"updatedAt" json:"updatedAt"`
}
