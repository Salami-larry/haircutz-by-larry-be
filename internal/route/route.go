package route

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"haircutz/backend/internal/auth"
	"haircutz/backend/internal/config"
	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/database"
	"haircutz/backend/internal/handler"
	"haircutz/backend/internal/middleware"
	"haircutz/backend/internal/repository"
)

func NewRouter(
	cfg config.Config,
	mongo *database.Mongo,
	tokens *auth.TokenIssuer,
	uploadHandler *handler.UploadHandler,
	hairstyleImages controller.HairstyleMediaDeleter,
	appointmentHandler *handler.AppointmentHandler,
	hairstyleDeleteGuard controller.HairstyleDeleteGuard,
	log *slog.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogMiddleware(log))
	r.Use(middleware.CORS(cfg.CORSOrigins))

	health := handler.NewHealthHandler(mongo)
	r.GET("/health", health.Liveness)
	r.GET("/ready", health.Readiness)

	adminRepo := repository.NewAdminRepository(mongo.Database)
	authCtrl := controller.NewAuthController(adminRepo, tokens)
	authHandler := handler.NewAuthHandler(authCtrl)

	hairstyleRepo := repository.NewHairstyleRepository(mongo.Database)
	hairstyleCtrl := controller.NewHairstyleController(hairstyleRepo, hairstyleImages, hairstyleDeleteGuard, log)
	hairstyleHandler := handler.NewHairstyleHandler(hairstyleCtrl)

	admin := r.Group("/api/v1/admin")
	admin.POST("/login", authHandler.Login)

	protected := admin.Group("")
	protected.Use(middleware.RequireAdminRole(tokens))
	protected.GET("/me", authHandler.Me)
	protected.POST("/hairstyles", hairstyleHandler.Create)
	protected.GET("/hairstyles", hairstyleHandler.List)
	protected.GET("/hairstyles/:id", hairstyleHandler.Get)
	protected.PUT("/hairstyles/:id", hairstyleHandler.Update)
	protected.DELETE("/hairstyles/:id", hairstyleHandler.Delete)
	protected.POST("/uploads", uploadHandler.UploadHairstyleImage)
	protected.POST("/uploads/video", uploadHandler.UploadHairstyleVideo)
	protected.GET("/appointments", appointmentHandler.ListAdmin)
	protected.GET("/appointments/:id", appointmentHandler.GetAdmin)

	v1 := r.Group("/api/v1")
	v1.GET("/availability", appointmentHandler.Availability)
	v1.POST("/appointments", appointmentHandler.Create)

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
