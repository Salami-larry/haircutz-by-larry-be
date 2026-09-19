package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/auth"
	"haircutz/backend/internal/model"
)

const (
	ContextAdminIDKey = "adminId"
	ContextAdminRole  = "adminRole"
	ContextAdminEmail = "adminEmail"
)

// RequireAdminRole validates JWT and requires role admin (from token claims, set at login from DB).
func RequireAdminRole(tokens *auth.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokens == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "auth is not configured",
			})
			return
		}

		header := c.GetHeader("Authorization")
		tokenString, ok := parseBearer(header)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		claims, err := tokens.Parse(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		if claims.Role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Set(ContextAdminIDKey, claims.AdminID)
		c.Set(ContextAdminRole, string(claims.Role))
		c.Set(ContextAdminEmail, claims.Email)
		c.Next()
	}
}

func parseBearer(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}
