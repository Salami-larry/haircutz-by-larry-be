package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr              string
	MongoURI              string
	CORSOrigins           []string
	JWTSecret             string
	SupabaseURL           string
	SupabaseServiceKey    string
	SupabaseStorageBucket string
	AppLogFile            string
	PaystackSecretKey     string
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPassword          string
	SMTPFrom              string
	AdminNotifyEmail      string
	ClientPublicURL       string
	PaystackCallbackURL   string
}

func (c Config) SupabaseConfigured() bool {
	return c.SupabaseURL != "" && c.SupabaseServiceKey != "" && c.SupabaseStorageBucket != ""
}

func (c Config) PaystackConfigured() bool {
	return c.PaystackSecretKey != ""
}

func (c Config) SMTPConfigured() bool {
	return c.SMTPHost != "" && c.SMTPFrom != "" && c.AdminNotifyEmail != ""
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:              getenv("HTTP_ADDR", ":8080"),
		MongoURI:              os.Getenv("MONGODB_URI"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		SupabaseURL:           os.Getenv("SUPABASE_URL"),
		SupabaseServiceKey:    os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		SupabaseStorageBucket: os.Getenv("SUPABASE_STORAGE_BUCKET"),
		AppLogFile:            os.Getenv("APP_LOG_FILE"),
		PaystackSecretKey:     os.Getenv("PAYSTACK_SECRET_KEY"),
		SMTPHost:              os.Getenv("SMTP_HOST"),
		SMTPPort:              getenv("SMTP_PORT", "587"),
		SMTPUser:              os.Getenv("SMTP_USER"),
		SMTPPassword:          os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:              os.Getenv("SMTP_FROM"),
		AdminNotifyEmail:      os.Getenv("ADMIN_NOTIFY_EMAIL"),
		ClientPublicURL:       getenv("CLIENT_PUBLIC_URL", "http://localhost:3000"),
		PaystackCallbackURL:   os.Getenv("PAYSTACK_CALLBACK_URL"),
	}

	if cfg.PaystackCallbackURL == "" {
		cfg.PaystackCallbackURL = strings.TrimSuffix(cfg.ClientPublicURL, "/") + "/book/success"
	}

	origins := os.Getenv("CORS_ORIGINS")
	if origins == "" {
		cfg.CORSOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	} else {
		for _, o := range strings.Split(origins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, o)
			}
		}
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
