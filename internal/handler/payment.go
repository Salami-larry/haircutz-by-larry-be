package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/paystack"
)

const rawBodyKey = "rawBody"

func PaystackWebhookRawBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		c.Set(rawBodyKey, raw)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		c.Next()
	}
}

func rawBodyFromContext(c *gin.Context) ([]byte, bool) {
	v, ok := c.Get(rawBodyKey)
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

type PaymentHandler struct {
	ctrl           *controller.PaymentController
	paystackSecret string
}

func NewPaymentHandler(ctrl *controller.PaymentController, paystackSecret string) *PaymentHandler {
	return &PaymentHandler{ctrl: ctrl, paystackSecret: paystackSecret}
}

func (h *PaymentHandler) Initialize(c *gin.Context) {
	var req struct {
		AppointmentID string `json:"appointmentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	id, err := parseObjectID(req.AppointmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointmentId"})
		return
	}

	result, err := h.ctrl.Initialize(c.Request.Context(), id)
	if err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *PaymentHandler) Verify(c *gin.Context) {
	result, err := h.ctrl.Verify(c.Request.Context(), c.Query("reference"))
	if err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *PaymentHandler) Abandon(c *gin.Context) {
	var req struct {
		Reference string `json:"reference"`
		Email     string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.ctrl.Abandon(c.Request.Context(), controller.AbandonPaymentInput{
		Reference: req.Reference,
		Email:     req.Email,
	}); err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "abandoned"})
}

func (h *PaymentHandler) MarkPaidAdmin(c *gin.Context) {
	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment id"})
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)

	appt, err := h.ctrl.MarkPaidAdmin(c.Request.Context(), id, req.Note)
	if err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, appt)
}

func (h *PaymentHandler) PaystackWebhook(c *gin.Context) {
	raw, ok := rawBodyFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing body"})
		return
	}
	sig := c.GetHeader("x-paystack-signature")
	if !paystack.VerifyWebhookSignature(h.paystackSecret, raw, sig) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var event paystack.WebhookEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event"})
		return
	}
	if !strings.EqualFold(event.Event, "charge.success") {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	data, err := paystack.ParseChargeSuccess(event.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid charge data"})
		return
	}
	if err := h.ctrl.HandleChargeSuccess(c.Request.Context(), data.Reference, data.Amount); err != nil {
		writePaymentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func writePaymentError(c *gin.Context, err error) {
	switch {
	case controller.IsAppointmentNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
	case controller.IsAbandonEmailMismatch(err):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case controller.IsAbandonConflict(err):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, controller.ErrPaystackNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case errors.Is(err, controller.ErrNotAwaitingPayment):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case strings.Contains(err.Error(), "amount mismatch"):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case strings.Contains(err.Error(), "required"):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}
