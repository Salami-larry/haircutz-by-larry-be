package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Hairstyle struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name                 string             `bson:"name" json:"name"`
	Description          string             `bson:"description" json:"description"`
	WalkInPriceKobo      int64              `bson:"walkInPriceKobo" json:"walkInPriceKobo"`
	HomeServicePriceKobo int64              `bson:"homeServicePriceKobo" json:"homeServicePriceKobo"`
	DurationMinutes      int                `bson:"durationMinutes" json:"durationMinutes"`
	ImageURLs            []string           `bson:"imageUrls" json:"imageUrls"`
	VideoURL             string             `bson:"videoUrl,omitempty" json:"videoUrl,omitempty"`
	Active               bool               `bson:"active" json:"active"`
	CreatedAt            time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time          `bson:"updatedAt" json:"updatedAt"`
}
