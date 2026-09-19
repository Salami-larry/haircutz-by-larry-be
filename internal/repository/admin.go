package repository

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"haircutz/backend/internal/model"
)

const adminsCollection = "admins"

var ErrAdminNotFound = errors.New("admin not found")

type AdminRepository struct {
	col *mongo.Collection
}

func NewAdminRepository(db *mongo.Database) *AdminRepository {
	return &AdminRepository{col: db.Collection(adminsCollection)}
}

func (r *AdminRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create admin indexes: %w", err)
	}
	return nil
}

func (r *AdminRepository) FindByEmail(ctx context.Context, email string) (*model.Admin, error) {
	var a model.Admin
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&a)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("find admin by email: %w", err)
	}
	return &a, nil
}
