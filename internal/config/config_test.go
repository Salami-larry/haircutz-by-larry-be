package config

import "testing"

func TestPaystackCallbackDefaultsToBookSuccess(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/test")
	t.Setenv("CLIENT_PUBLIC_URL", "http://localhost:3000")
	t.Setenv("PAYSTACK_CALLBACK_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PaystackCallbackURL != "http://localhost:3000/book/success" {
		t.Fatalf("got %q", cfg.PaystackCallbackURL)
	}
}

func TestPaystackCallbackAppendsPathWhenOriginOnly(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/test")
	t.Setenv("PAYSTACK_CALLBACK_URL", "http://localhost:3000")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PaystackCallbackURL != "http://localhost:3000/book/success" {
		t.Fatalf("got %q", cfg.PaystackCallbackURL)
	}
}
