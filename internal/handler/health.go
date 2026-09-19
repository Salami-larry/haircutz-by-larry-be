package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/database"
)

type HealthHandler struct {
	mongo *database.Mongo
}

func NewHealthHandler(mongo *database.Mongo) *HealthHandler {
	return &HealthHandler{mongo: mongo}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "haircutz-api",
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	if h.mongo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"reason": "database not configured",
		})
		return
	}

	if err := h.mongo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"reason": "database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
