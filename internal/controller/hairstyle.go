package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
	"haircutz/backend/internal/repository"
)

// HairstyleMediaDeleter removes orphaned hairstyle media from object storage (optional).
type HairstyleMediaDeleter interface {
	DeleteByPublicURL(publicURL string) error
}

// HairstyleDeleteGuard blocks delete when blocking appointments exist (wired in a later phase).
type HairstyleDeleteGuard interface {
	HasBlockingAppointments(ctx context.Context, hairstyleID primitive.ObjectID) (bool, error)
}

type HairstyleInput struct {
	Name                 string
	Description          string
	WalkInPriceKobo      int64
	HomeServicePriceKobo int64
	DurationMinutes      int
	ImageURLs            []string
	VideoURL             string
	Active               bool
}

var (
	ErrHairstyleDeleteBlocked = errors.New("cannot delete hairstyle with active or unresolved appointments")
)

type HairstyleController struct {
	repo   *repository.HairstyleRepository
	images HairstyleMediaDeleter
	guard  HairstyleDeleteGuard
	log    *slog.Logger
}

func NewHairstyleController(
	repo *repository.HairstyleRepository,
	images HairstyleMediaDeleter,
	guard HairstyleDeleteGuard,
	log *slog.Logger,
) *HairstyleController {
	if log == nil {
		log = slog.Default()
	}
	return &HairstyleController{repo: repo, images: images, guard: guard, log: log}
}

func (c *HairstyleController) Create(ctx context.Context, in HairstyleInput) (*model.Hairstyle, error) {
	normalized, err := normalizeAndValidateHairstyleInput(in)
	if err != nil {
		return nil, err
	}

	h := &model.Hairstyle{
		Name:                 normalized.Name,
		Description:          normalized.Description,
		WalkInPriceKobo:      normalized.WalkInPriceKobo,
		HomeServicePriceKobo: normalized.HomeServicePriceKobo,
		DurationMinutes:      normalized.DurationMinutes,
		ImageURLs:            normalized.ImageURLs,
		VideoURL:             normalized.VideoURL,
		Active:               normalized.Active,
	}

	if err := c.repo.Create(ctx, h); err != nil {
		return nil, fmt.Errorf("create hairstyle: %w", err)
	}
	return h, nil
}

func (c *HairstyleController) Update(ctx context.Context, id primitive.ObjectID, in HairstyleInput) (*model.Hairstyle, error) {
	normalized, err := normalizeAndValidateHairstyleInput(in)
	if err != nil {
		return nil, err
	}

	existing, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	previousURLs := hairstyleMediaURLs(*existing)

	existing.Name = normalized.Name
	existing.Description = normalized.Description
	existing.WalkInPriceKobo = normalized.WalkInPriceKobo
	existing.HomeServicePriceKobo = normalized.HomeServicePriceKobo
	existing.DurationMinutes = normalized.DurationMinutes
	existing.ImageURLs = normalized.ImageURLs
	existing.VideoURL = normalized.VideoURL
	existing.Active = normalized.Active

	if err := c.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update hairstyle: %w", err)
	}

	c.deleteOrphanedMedia(previousURLs, hairstyleMediaURLs(*existing))
	return existing, nil
}

func (c *HairstyleController) Get(ctx context.Context, id primitive.ObjectID) (*model.Hairstyle, error) {
	return c.repo.FindByID(ctx, id)
}

func (c *HairstyleController) List(ctx context.Context, f repository.HairstyleListFilter) (pagination.Paginated[model.Hairstyle], error) {
	items, total, err := c.repo.List(ctx, f)
	if err != nil {
		return pagination.Paginated[model.Hairstyle]{}, err
	}
	return pagination.NewPaginated(items, total, f.Page, f.PageSize), nil
}

func (c *HairstyleController) Delete(ctx context.Context, id primitive.ObjectID) error {
	if c.guard != nil {
		blocked, err := c.guard.HasBlockingAppointments(ctx, id)
		if err != nil {
			return fmt.Errorf("check appointments for hairstyle delete: %w", err)
		}
		if blocked {
			return ErrHairstyleDeleteBlocked
		}
	}

	existing, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	mediaURLs := hairstyleMediaURLs(*existing)
	if err := c.repo.Delete(ctx, id); err != nil {
		return err
	}

	c.deleteOrphanedMedia(mediaURLs, nil)
	return nil
}

type normalizedHairstyleInput struct {
	Name                 string
	Description          string
	WalkInPriceKobo      int64
	HomeServicePriceKobo int64
	DurationMinutes      int
	ImageURLs            []string
	VideoURL             string
	Active               bool
}

func normalizeAndValidateHairstyleInput(in HairstyleInput) (normalizedHairstyleInput, error) {
	out := normalizedHairstyleInput{
		Name:                 strings.TrimSpace(in.Name),
		Description:          strings.TrimSpace(in.Description),
		WalkInPriceKobo:      in.WalkInPriceKobo,
		HomeServicePriceKobo: in.HomeServicePriceKobo,
		DurationMinutes:      in.DurationMinutes,
		VideoURL:             strings.TrimSpace(in.VideoURL),
		Active:               in.Active,
	}
	if out.Name == "" {
		return out, fmt.Errorf("name is required")
	}
	if out.Description == "" {
		return out, fmt.Errorf("description is required")
	}
	if out.WalkInPriceKobo < 0 {
		return out, fmt.Errorf("walk-in price must be >= 0")
	}
	if out.HomeServicePriceKobo < 0 {
		return out, fmt.Errorf("home service price must be >= 0")
	}
	if out.DurationMinutes < 1 {
		return out, fmt.Errorf("duration must be at least 1 minute")
	}

	out.ImageURLs = normalizeImageURLs(in.ImageURLs)
	if len(out.ImageURLs) < 1 {
		return out, fmt.Errorf("at least one image is required")
	}
	if len(out.ImageURLs) > 3 {
		return out, fmt.Errorf("at most 3 images are allowed")
	}

	return out, nil
}

func normalizeImageURLs(urls []string) []string {
	out := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func IsHairstyleNotFound(err error) bool {
	return errors.Is(err, repository.ErrHairstyleNotFound)
}

func hairstyleMediaURLs(h model.Hairstyle) []string {
	out := make([]string, 0, len(h.ImageURLs)+1)
	for _, u := range h.ImageURLs {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	if u := strings.TrimSpace(h.VideoURL); u != "" {
		out = append(out, u)
	}
	return out
}

func mediaURLsToDelete(previous, current []string) []string {
	keep := make(map[string]struct{}, len(current))
	for _, u := range current {
		keep[u] = struct{}{}
	}
	var del []string
	for _, u := range previous {
		if _, ok := keep[u]; !ok {
			del = append(del, u)
		}
	}
	return del
}

func (c *HairstyleController) deleteOrphanedMedia(previous, current []string) {
	if c.images == nil {
		return
	}
	for _, u := range mediaURLsToDelete(previous, current) {
		if err := c.images.DeleteByPublicURL(u); err != nil {
			c.log.Warn("failed to delete orphaned hairstyle media", "url", u, "err", err)
			continue
		}
		c.log.Info("deleted orphaned hairstyle media", "url", u)
	}
}
