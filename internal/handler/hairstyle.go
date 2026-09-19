package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/pagination"
	"haircutz/backend/internal/repository"
)

type HairstyleHandler struct {
	ctrl *controller.HairstyleController
}

func NewHairstyleHandler(ctrl *controller.HairstyleController) *HairstyleHandler {
	return &HairstyleHandler{ctrl: ctrl}
}

type hairstyleRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	WalkInPriceKobo      int64    `json:"walkInPriceKobo"`
	HomeServicePriceKobo int64    `json:"homeServicePriceKobo"`
	DurationMinutes      int      `json:"durationMinutes"`
	ImageURLs            []string `json:"imageUrls"`
	VideoURL             string   `json:"videoUrl"`
	Active               bool     `json:"active"`
}

func (h *HairstyleHandler) Create(c *gin.Context) {
	var req hairstyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	hs, err := h.ctrl.Create(c.Request.Context(), requestToHairstyleInput(req))
	if err != nil {
		writeHairstyleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, hs)
}

func (h *HairstyleHandler) Update(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hairstyle id"})
		return
	}

	var req hairstyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	hs, err := h.ctrl.Update(c.Request.Context(), id, requestToHairstyleInput(req))
	if err != nil {
		writeHairstyleError(c, err)
		return
	}
	c.JSON(http.StatusOK, hs)
}

func (h *HairstyleHandler) Get(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hairstyle id"})
		return
	}

	hs, err := h.ctrl.Get(c.Request.Context(), id)
	if err != nil {
		writeHairstyleError(c, err)
		return
	}
	c.JSON(http.StatusOK, hs)
}

func (h *HairstyleHandler) List(c *gin.Context) {
	pageParams, err := pagination.ParseQuery(c.Query("page"), c.Query("page_size"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f := repository.HairstyleListFilter{
		Query:    strings.TrimSpace(c.Query("q")),
		Page:     pageParams.Page,
		PageSize: pageParams.PageSize,
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "active must be true or false"})
			return
		}
		f.Active = &active
	}

	result, err := h.ctrl.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list hairstyles"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *HairstyleHandler) Delete(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hairstyle id"})
		return
	}

	if err := h.ctrl.Delete(c.Request.Context(), id); err != nil {
		writeHairstyleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func requestToHairstyleInput(req hairstyleRequest) controller.HairstyleInput {
	return controller.HairstyleInput{
		Name:                 req.Name,
		Description:          req.Description,
		WalkInPriceKobo:      req.WalkInPriceKobo,
		HomeServicePriceKobo: req.HomeServicePriceKobo,
		DurationMinutes:      req.DurationMinutes,
		ImageURLs:            req.ImageURLs,
		VideoURL:             req.VideoURL,
		Active:               req.Active,
	}
}

func parseObjectID(raw string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(strings.TrimSpace(raw))
}

func writeHairstyleError(c *gin.Context, err error) {
	switch {
	case controller.IsHairstyleNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "hairstyle not found"})
	case errors.Is(err, controller.ErrHairstyleDeleteBlocked):
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
			"code":  "hairstyle_delete_blocked",
		})
	case strings.Contains(err.Error(), "required") ||
		strings.Contains(err.Error(), "must be") ||
		strings.Contains(err.Error(), "at least") ||
		strings.Contains(err.Error(), "at most"):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}
