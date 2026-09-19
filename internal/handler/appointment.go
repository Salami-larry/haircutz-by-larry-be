package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/model"
	"haircutz/backend/internal/pagination"
	"haircutz/backend/internal/repository"
	"haircutz/backend/internal/schedule"
)

type AppointmentHandler struct {
	ctrl *controller.AppointmentController
}

func NewAppointmentHandler(ctrl *controller.AppointmentController) *AppointmentHandler {
	return &AppointmentHandler{ctrl: ctrl}
}

type createAppointmentRequest struct {
	HairstyleID string `json:"hairstyleId"`
	ServiceType string `json:"serviceType"`
	StartAt     string `json:"startAt"`
	Customer    struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
		Notes   string `json:"notes"`
	} `json:"customer"`
}

func (h *AppointmentHandler) Availability(c *gin.Context) {
	dateStr := strings.TrimSpace(c.Query("date"))
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required (YYYY-MM-DD)"})
		return
	}

	hsID, err := parseObjectID(c.Query("hairstyleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hairstyleId"})
		return
	}

	service, err := model.ParseServiceType(strings.TrimSpace(c.Query("serviceType")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceType must be walk_in or home_service"})
		return
	}

	result, err := h.ctrl.Availability(c.Request.Context(), dateStr, hsID, service)
	if err != nil {
		writeAppointmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AppointmentHandler) Create(c *gin.Context) {
	var req createAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	hsID, err := parseObjectID(req.HairstyleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hairstyleId"})
		return
	}

	service, err := model.ParseServiceType(strings.TrimSpace(req.ServiceType))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceType must be walk_in or home_service"})
		return
	}

	start, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartAt))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "startAt must be RFC3339"})
		return
	}

	appt, err := h.ctrl.Create(c.Request.Context(), controller.CreateAppointmentInput{
		HairstyleID: hsID,
		ServiceType: service,
		StartAt:     start,
		Customer: model.AppointmentCustomer{
			Name:    req.Customer.Name,
			Email:   req.Customer.Email,
			Phone:   req.Customer.Phone,
			Address: req.Customer.Address,
			Notes:   req.Customer.Notes,
		},
	})
	if err != nil {
		writeAppointmentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, appt)
}

func (h *AppointmentHandler) ListAdmin(c *gin.Context) {
	pageParams, err := pagination.ParseQuery(c.Query("page"), c.Query("page_size"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f := repository.AppointmentListFilter{
		Page:     pageParams.Page,
		PageSize: pageParams.PageSize,
	}

	if statusStr := strings.TrimSpace(c.Query("status")); statusStr != "" {
		st := model.AppointmentStatus(statusStr)
		if !st.Valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		f.Status = &st
	}

	if dateStr := strings.TrimSpace(c.Query("date")); dateStr != "" {
		loc, err := schedule.LoadLocation()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "timezone unavailable"})
			return
		}
		day, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
			return
		}
		start := day
		end := day.Add(24 * time.Hour)
		startUTC := start.UTC()
		endUTC := end.UTC()
		f.DayStart = &startUTC
		f.DayEnd = &endUTC
	}

	result, err := h.ctrl.ListAdmin(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list appointments"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AppointmentHandler) GetAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment id"})
		return
	}
	appt, err := h.ctrl.Get(c.Request.Context(), id)
	if err != nil {
		writeAppointmentError(c, err)
		return
	}
	c.JSON(http.StatusOK, appt)
}

func writeAppointmentError(c *gin.Context, err error) {
	switch {
	case controller.IsHairstyleNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "hairstyle not found"})
	case controller.IsAppointmentNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
	case errors.Is(err, controller.ErrSlotUnavailable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "slot_unavailable"})
	case errors.Is(err, controller.ErrHairstyleInactive):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, controller.ErrInvalidStartTime),
		errors.Is(err, controller.ErrServiceClosed),
		errors.Is(err, controller.ErrAddressRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case strings.Contains(err.Error(), "required") ||
		strings.Contains(err.Error(), "must be") ||
		strings.Contains(err.Error(), "YYYY-MM-DD"):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}
