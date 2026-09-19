package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"haircutz/backend/internal/auth"
	"haircutz/backend/internal/config"
	"haircutz/backend/internal/controller"
	"haircutz/backend/internal/database"
	"haircutz/backend/internal/handler"
	"haircutz/backend/internal/jobs"
	"haircutz/backend/internal/logging"
	"haircutz/backend/internal/mail"
	"haircutz/backend/internal/middleware"
	"haircutz/backend/internal/paystack"
	"haircutz/backend/internal/repository"
	"haircutz/backend/internal/route"
	"haircutz/backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log, logCloser, err := logging.New(cfg.AppLogFile)
	if err != nil {
		slog.Error("logger setup failed", "err", err)
		os.Exit(1)
	}
	defer logCloser.Close()

	if cfg.AppLogFile != "" {
		log.Info("file logging enabled", "path", cfg.AppLogFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	mongo, err := database.Connect(ctx, cfg.MongoURI)
	cancel()
	if err != nil {
		log.Error("mongodb connect failed", "err", err)
		os.Exit(1)
	}

	log.Info("mongodb connected",
		"database", mongo.Database.Name(),
		"cors", middleware.AllowedOriginList(cfg.CORSOrigins),
	)

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := repository.NewAdminRepository(mongo.Database).EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("admin indexes failed", "err", err)
		os.Exit(1)
	}
	if err := repository.NewHairstyleRepository(mongo.Database).EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("hairstyle indexes failed", "err", err)
		os.Exit(1)
	}
	appointmentRepo := repository.NewAppointmentRepository(mongo.Database)
	if err := appointmentRepo.EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("appointment indexes failed", "err", err)
		os.Exit(1)
	}
	indexCancel()

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		log.Error("jwt setup failed", "err", err)
		os.Exit(1)
	}

	var storageClient *storage.Client
	var uploadHandler *handler.UploadHandler
	var hairstyleImages controller.HairstyleMediaDeleter
	if cfg.SupabaseConfigured() {
		var err error
		storageClient, err = storage.NewClient(storage.Config{
			SupabaseURL:    cfg.SupabaseURL,
			ServiceRoleKey: cfg.SupabaseServiceKey,
			Bucket:         cfg.SupabaseStorageBucket,
		})
		if err != nil {
			log.Error("supabase storage setup failed", "err", err)
			os.Exit(1)
		}
		hairstyleImages = storageClient
		uploadHandler = handler.NewUploadHandler(controller.NewUploadController(storageClient), log)
		log.Info("supabase storage enabled", "bucket", cfg.SupabaseStorageBucket)
	} else {
		log.Info("supabase storage not configured; POST /admin/uploads will return 503")
		uploadHandler = handler.NewUploadHandler(controller.NewUploadController(nil), log)
	}

	var psClient *paystack.Client
	if cfg.PaystackConfigured() {
		psClient = paystack.NewClient(cfg.PaystackSecretKey, cfg.PaystackCallbackURL)
		log.Info("paystack enabled", "callbackURL", cfg.PaystackCallbackURL)
	} else {
		log.Info("paystack not configured; payments and webhooks will return 503")
	}

	var mailer *mail.Sender
	if cfg.SMTPConfigured() {
		mailer = mail.NewSender(mail.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		})
		log.Info("smtp enabled", "from", cfg.SMTPFrom)
	} else {
		log.Info("smtp not configured; payment emails will be skipped")
	}

	hairstyleRepo := repository.NewHairstyleRepository(mongo.Database)
	appointmentCtrl, err := controller.NewAppointmentController(
		appointmentRepo,
		hairstyleRepo,
		mailer,
		cfg.ClientPublicURL,
		log,
	)
	if err != nil {
		log.Error("appointment controller setup failed", "err", err)
		os.Exit(1)
	}
	appointmentHandler := handler.NewAppointmentHandler(appointmentCtrl)
	hairstyleDeleteGuard := controller.NewAppointmentDeleteGuard(appointmentRepo)

	paymentCtrl := controller.NewPaymentController(
		appointmentRepo,
		psClient,
		mailer,
		cfg.AdminNotifyEmail,
		cfg.ClientPublicURL,
		log,
	)
	paymentHandler := handler.NewPaymentHandler(paymentCtrl, cfg.PaystackSecretKey)

	jobCtx, jobCancel := context.WithCancel(context.Background())
	defer jobCancel()
	go jobs.RunAbandonStaleBooked(jobCtx, appointmentRepo, log)
	log.Info("abandon hold job started", "ttl", jobs.BookedHoldTTL.String(), "interval", jobs.AbandonHoldInterval.String())

	router := route.NewRouter(
		cfg,
		mongo,
		tokens,
		uploadHandler,
		hairstyleImages,
		appointmentHandler,
		paymentHandler,
		hairstyleDeleteGuard,
		log,
	)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	jobCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "err", err)
	}

	if err := mongo.Close(shutdownCtx); err != nil {
		log.Error("mongodb disconnect failed", "err", err)
	}

	log.Info("shutdown complete")
}
