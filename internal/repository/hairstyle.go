package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
)

const hairstylesCollection = "hairstyles"

var ErrHairstyleNotFound = errors.New("hairstyle not found")

type HairstyleRepository struct {
	col *mongo.Collection
}

func NewHairstyleRepository(db *mongo.Database) *HairstyleRepository {
	return &HairstyleRepository{col: db.Collection(hairstylesCollection)}
}

func (r *HairstyleRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "active", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create hairstyle indexes: %w", err)
	}
	return nil
}

type HairstyleListFilter struct {
	Active   *bool
	Query    string
	Page     int
	PageSize int
}

func (r *HairstyleRepository) Create(ctx context.Context, h *model.Hairstyle) error {
	now := time.Now().UTC()
	h.CreatedAt = now
	h.UpdatedAt = now
	if h.ID.IsZero() {
		h.ID = primitive.NewObjectID()
	}
	_, err := r.col.InsertOne(ctx, h)
	if err != nil {
		return fmt.Errorf("insert hairstyle: %w", err)
	}
	return nil
}

func (r *HairstyleRepository) Update(ctx context.Context, h *model.Hairstyle) error {
	h.UpdatedAt = time.Now().UTC()
	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": h.ID}, h)
	if err != nil {
		return fmt.Errorf("update hairstyle: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrHairstyleNotFound
	}
	return nil
}

func (r *HairstyleRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Hairstyle, error) {
	var h model.Hairstyle
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&h)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrHairstyleNotFound
		}
		return nil, fmt.Errorf("find hairstyle: %w", err)
	}
	return &h, nil
}

func (r *HairstyleRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete hairstyle: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrHairstyleNotFound
	}
	return nil
}

func (r *HairstyleRepository) List(ctx context.Context, f HairstyleListFilter) ([]model.Hairstyle, int64, error) {
	if f.Page < 1 {
		f.Page = pagination.DefaultPage
	}
	if f.PageSize < 1 {
		f.PageSize = pagination.DefaultPageSize
	}

	filter := bson.M{}
	if f.Active != nil {
		filter["active"] = *f.Active
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		escaped := regexp.QuoteMeta(q)
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": escaped, "$options": "i"}},
			{"description": bson.M{"$regex": escaped, "$options": "i"}},
		}
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count hairstyles: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((f.Page - 1) * f.PageSize)).
		SetLimit(int64(f.PageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list hairstyles: %w", err)
	}
	defer cur.Close(ctx)

	items := make([]model.Hairstyle, 0, f.PageSize)
	for cur.Next(ctx) {
		var h model.Hairstyle
		if err := cur.Decode(&h); err != nil {
			return nil, 0, fmt.Errorf("decode hairstyle: %w", err)
		}
		items = append(items, h)
	}
	if err := cur.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate hairstyles: %w", err)
	}
	return items, total, nil
}
