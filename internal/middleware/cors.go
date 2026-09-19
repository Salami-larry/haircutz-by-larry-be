package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					c.AbortWithStatus(http.StatusNoContent)
					return
				}
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// AllowedOriginList returns origins for logging (comma-separated).
func AllowedOriginList(origins []string) string {
	return strings.Join(origins, ", ")
}
