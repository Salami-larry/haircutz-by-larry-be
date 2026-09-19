package route

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/config"
	"haircutz/backend/internal/database"
	"haircutz/backend/internal/handler"
	"haircutz/backend/internal/middleware"
)

func NewRouter(cfg config.Config, mongo *database.Mongo, log *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogMiddleware(log))
	r.Use(middleware.CORS(cfg.CORSOrigins))

	health := handler.NewHealthHandler(mongo)
	r.GET("/health", health.Liveness)
	r.GET("/ready", health.Readiness)

	return r
}

func requestLogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		c.Next()
		if path == "/health" || path == "/ready" {
			return
		}
		log.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
		)
	}
}
