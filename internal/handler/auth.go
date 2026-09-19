package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/middleware"
)

type AuthHandler struct {
	ctrl *controller.AuthController
}

func NewAuthHandler(ctrl *controller.AuthController) *AuthHandler {
	return &AuthHandler{ctrl: ctrl}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.ctrl.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, controller.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		case errors.Is(err, controller.ErrNotAdminRole):
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     result.Token,
		"expiresAt": result.ExpiresAt,
		"role":      result.Role,
		"email":     result.Email,
	})
}

// Me returns the authenticated admin claims (Phase 1 smoke / session check).
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"adminId": c.GetString(middleware.ContextAdminIDKey),
		"email":   c.GetString(middleware.ContextAdminEmail),
		"role":    c.GetString(middleware.ContextAdminRole),
	})
}
