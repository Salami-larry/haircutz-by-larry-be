package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
)

const appointmentsCollection = "appointments"

var ErrAppointmentNotFound = errors.New("appointment not found")

type AppointmentRepository struct {
	col *mongo.Collection
}

func NewAppointmentRepository(db *mongo.Database) *AppointmentRepository {
	return &AppointmentRepository{col: db.Collection(appointmentsCollection)}
}

func (r *AppointmentRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "startAt", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "hairstyleId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "paystackReference", Value: 1}}, Options: options.Index().SetSparse(true)},
		{Keys: bson.D{{Key: "trackingNumber", Value: 1}}, Options: options.Index().SetSparse(true)},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create appointment indexes: %w", err)
	}
	return nil
}

func (r *AppointmentRepository) Create(ctx context.Context, a *model.Appointment) error {
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.ID.IsZero() {
		a.ID = primitive.NewObjectID()
	}
	_, err := r.col.InsertOne(ctx, a)
	if err != nil {
		return fmt.Errorf("insert appointment: %w", err)
	}
	return nil
}

func (r *AppointmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete appointment: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrAppointmentNotFound
	}
	return nil
}

func (r *AppointmentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Appointment, error) {
	var a model.Appointment
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&a)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("find appointment: %w", err)
	}
	return &a, nil
}

func (r *AppointmentRepository) Update(ctx context.Context, a *model.Appointment) error {
	a.UpdatedAt = time.Now().UTC()
	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": a.ID}, a)
	if err != nil {
		return fmt.Errorf("update appointment: %w", err)
	}
	if res.MatchedCount == 0 {
		return ErrAppointmentNotFound
	}
	return nil
}

func (r *AppointmentRepository) FindByPaystackReference(ctx context.Context, reference string) (*model.Appointment, error) {
	var a model.Appointment
	err := r.col.FindOne(ctx, bson.M{"paystackReference": reference}).Decode(&a)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("find appointment by reference: %w", err)
	}
	return &a, nil
}

func (r *AppointmentRepository) FindForTrack(ctx context.Context, trackingNumber, email string) (*model.Appointment, error) {
	filter := bson.M{
		"trackingNumber": strings.TrimSpace(trackingNumber),
		"customer.email": strings.ToLower(strings.TrimSpace(email)),
	}
	var a model.Appointment
	err := r.col.FindOne(ctx, filter).Decode(&a)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("find appointment for track: %w", err)
	}
	return &a, nil
}

// FindBlockingInRange returns appointments whose [startAt,endAt) overlaps [rangeStart,rangeEnd)
// and whose status blocks the calendar.
func (r *AppointmentRepository) FindBlockingInRange(ctx context.Context, rangeStart, rangeEnd time.Time) ([]model.Appointment, error) {
	filter := bson.M{
		"status": bson.M{"$in": model.BlockingStatuses()},
		"startAt": bson.M{"$lt": rangeEnd},
		"endAt":   bson.M{"$gt": rangeStart},
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "startAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find blocking appointments: %w", err)
	}
	defer cur.Close(ctx)

	items := make([]model.Appointment, 0)
	for cur.Next(ctx) {
		var a model.Appointment
		if err := cur.Decode(&a); err != nil {
			return nil, fmt.Errorf("decode appointment: %w", err)
		}
		items = append(items, a)
	}
	return items, cur.Err()
}

func (r *AppointmentRepository) CountOverlapping(ctx context.Context, start, end time.Time, excludeID *primitive.ObjectID) (int64, error) {
	filter := bson.M{
		"status":  bson.M{"$in": model.BlockingStatuses()},
		"startAt": bson.M{"$lt": end},
		"endAt":   bson.M{"$gt": start},
	}
	if excludeID != nil {
		filter["_id"] = bson.M{"$ne": *excludeID}
	}
	n, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count overlapping appointments: %w", err)
	}
	return n, nil
}

// AbandonStaleBooked marks booked appointments with createdAt before cutoff as abandoned.
func (r *AppointmentRepository) AbandonStaleBooked(ctx context.Context, cutoff time.Time) (int64, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"status":    model.AppointmentBooked,
		"createdAt": bson.M{"$lt": cutoff},
	}
	update := bson.M{
		"$set": bson.M{
			"status":      model.AppointmentAbandoned,
			"abandonedAt": now,
			"updatedAt":   now,
		},
		"$push": bson.M{
			"statusHistory": model.AppointmentStatusEvent{
				Status: model.AppointmentAbandoned,
				At:     now,
				Note:   "Unpaid hold expired",
			},
		},
	}
	res, err := r.col.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("abandon stale booked: %w", err)
	}
	return res.ModifiedCount, nil
}

type AppointmentListFilter struct {
	Status   *model.AppointmentStatus
	DayStart *time.Time // inclusive UTC/local instant for day filter
	DayEnd   *time.Time // exclusive
	Page     int
	PageSize int
}

func (r *AppointmentRepository) List(ctx context.Context, f AppointmentListFilter) ([]model.Appointment, int64, error) {
	if f.Page < 1 {
		f.Page = pagination.DefaultPage
	}
	if f.PageSize < 1 {
		f.PageSize = pagination.DefaultPageSize
	}

	filter := bson.M{}
	if f.Status != nil {
		filter["status"] = *f.Status
	}
	if f.DayStart != nil && f.DayEnd != nil {
		filter["startAt"] = bson.M{
			"$gte": *f.DayStart,
			"$lt":  *f.DayEnd,
		}
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count appointments: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "startAt", Value: 1}}).
		SetSkip(int64((f.Page - 1) * f.PageSize)).
		SetLimit(int64(f.PageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list appointments: %w", err)
	}
	defer cur.Close(ctx)

	items := make([]model.Appointment, 0, f.PageSize)
	for cur.Next(ctx) {
		var a model.Appointment
		if err := cur.Decode(&a); err != nil {
			return nil, 0, fmt.Errorf("decode appointment: %w", err)
		}
		items = append(items, a)
	}
	return items, total, cur.Err()
}

// HasBlockingAppointments reports whether any appointment for the hairstyle
// is booked/paid/acknowledged/missed (blocks hairstyle delete).
func (r *AppointmentRepository) HasBlockingAppointments(ctx context.Context, hairstyleID primitive.ObjectID) (bool, error) {
	filter := bson.M{
		"hairstyleId": hairstyleID,
		"status":      bson.M{"$in": model.BlockingForHairstyleDelete()},
	}
	err := r.col.FindOne(ctx, filter).Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("check blocking appointments: %w", err)
	}
	return true, nil
}

func reminderStatuses() []model.AppointmentStatus {
	return []model.AppointmentStatus{model.AppointmentPaid, model.AppointmentAcknowledged}
}

// FindDueReminders returns paid/acknowledged appointments starting within the next window
// that have not been reminded, excluding short-lead bookings (created within lead of start).
func (r *AppointmentRepository) FindDueReminders(ctx context.Context, now time.Time, window, minLead time.Duration) ([]model.Appointment, error) {
	now = now.UTC()
	filter := bson.M{
		"status": bson.M{"$in": reminderStatuses()},
		"startAt": bson.M{
			"$gt":  now,
			"$lte":  now.Add(window),
		},
		"$or": []bson.M{
			{"notifications.reminderSentAt": bson.M{"$exists": false}},
			{"notifications.reminderSentAt": nil},
		},
		"$expr": bson.M{
			"$gte": bson.A{
				bson.M{"$subtract": bson.A{"$startAt", "$createdAt"}},
				int64(minLead / time.Millisecond),
			},
		},
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "startAt", Value: 1}}).SetLimit(50))
	if err != nil {
		return nil, fmt.Errorf("find due reminders: %w", err)
	}
	defer cur.Close(ctx)

	items := make([]model.Appointment, 0)
	for cur.Next(ctx) {
		var a model.Appointment
		if err := cur.Decode(&a); err != nil {
			return nil, fmt.Errorf("decode reminder appointment: %w", err)
		}
		items = append(items, a)
	}
	return items, cur.Err()
}

// ClaimReminderSent sets reminderSentAt if still unset. Returns false if already claimed.
func (r *AppointmentRepository) ClaimReminderSent(ctx context.Context, id primitive.ObjectID, at time.Time) (bool, error) {
	at = at.UTC()
	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id": id,
		"$or": []bson.M{
			{"notifications.reminderSentAt": bson.M{"$exists": false}},
			{"notifications.reminderSentAt": nil},
		},
	}, bson.M{"$set": bson.M{
		"notifications.reminderSentAt": at,
		"updatedAt":                    at,
	}})
	if err != nil {
		return false, fmt.Errorf("claim reminder sent: %w", err)
	}
	return res.ModifiedCount > 0, nil
}

// FindDuePostTimeNags returns paid/acknowledged appointments past endAt that need an admin nag.
func (r *AppointmentRepository) FindDuePostTimeNags(ctx context.Context, now time.Time, nagInterval time.Duration) ([]model.Appointment, error) {
	now = now.UTC()
	cutoff := now.Add(-nagInterval)
	filter := bson.M{
		"status": bson.M{"$in": reminderStatuses()},
		"endAt":  bson.M{"$lt": now},
		"$or": []bson.M{
			{"notifications.postTimeNagAt": bson.M{"$exists": false}},
			{"notifications.postTimeNagAt": nil},
			{"notifications.postTimeNagAt": bson.M{"$lte": cutoff}},
		},
	}
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "endAt", Value: 1}}).SetLimit(50))
	if err != nil {
		return nil, fmt.Errorf("find due post-time nags: %w", err)
	}
	defer cur.Close(ctx)

	items := make([]model.Appointment, 0)
	for cur.Next(ctx) {
		var a model.Appointment
		if err := cur.Decode(&a); err != nil {
			return nil, fmt.Errorf("decode post-time nag appointment: %w", err)
		}
		items = append(items, a)
	}
	return items, cur.Err()
}

// ClaimPostTimeNag sets postTimeNagAt when due (nil or older than interval). Returns false if not claimed.
func (r *AppointmentRepository) ClaimPostTimeNag(ctx context.Context, id primitive.ObjectID, at time.Time, nagInterval time.Duration) (bool, error) {
	at = at.UTC()
	cutoff := at.Add(-nagInterval)
	res, err := r.col.UpdateOne(ctx, bson.M{
		"_id":    id,
		"status": bson.M{"$in": reminderStatuses()},
		"$or": []bson.M{
			{"notifications.postTimeNagAt": bson.M{"$exists": false}},
			{"notifications.postTimeNagAt": nil},
			{"notifications.postTimeNagAt": bson.M{"$lte": cutoff}},
		},
	}, bson.M{"$set": bson.M{
		"notifications.postTimeNagAt": at,
		"updatedAt":                   at,
	}})
	if err != nil {
		return false, fmt.Errorf("claim post-time nag: %w", err)
	}
	return res.ModifiedCount > 0, nil
}

