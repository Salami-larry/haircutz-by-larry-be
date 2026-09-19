package model

import (
	"strings"
	"testing"
	"time"
)

func TestNewTrackingNumber(t *testing.T) {
	n := NewTrackingNumber(time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC))
	if !strings.HasPrefix(n, "HBL-20260920-") {
		t.Fatalf("got %q", n)
	}
	if len(n) != len("HBL-20260920-XXXXXX") {
		t.Fatalf("length %d for %q", len(n), n)
	}
}
