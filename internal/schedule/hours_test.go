package schedule

import (
	"testing"
	"time"

	"haircutz/backend/internal/model"
)

func TestWindowForDay(t *testing.T) {
	loc, err := LoadLocation()
	if err != nil {
		t.Fatal(err)
	}

	monday := time.Date(2026, 9, 14, 12, 0, 0, 0, loc) // Monday
	w, ok, err := WindowForDay(monday, model.ServiceWalkIn, loc)
	if err != nil || !ok {
		t.Fatalf("walk-in monday: ok=%v err=%v", ok, err)
	}
	if w.Open.Hour() != 9 || w.Close.Hour() != 22 {
		t.Fatalf("walk-in hours = %v–%v", w.Open, w.Close)
	}

	home, ok, err := WindowForDay(monday, model.ServiceHomeService, loc)
	if err != nil || !ok {
		t.Fatalf("home monday: ok=%v err=%v", ok, err)
	}
	if home.Open.Hour() != 10 || home.Close.Hour() != 17 {
		t.Fatalf("home hours = %v–%v", home.Open, home.Close)
	}

	sunday := time.Date(2026, 9, 13, 12, 0, 0, 0, loc)
	sw, ok, err := WindowForDay(sunday, model.ServiceWalkIn, loc)
	if err != nil || !ok {
		t.Fatalf("walk-in sunday: ok=%v err=%v", ok, err)
	}
	if sw.Open.Hour() != 12 {
		t.Fatalf("sunday walk-in open = %v", sw.Open)
	}
	_, ok, err = WindowForDay(sunday, model.ServiceHomeService, loc)
	if err != nil || ok {
		t.Fatalf("home sunday should be closed, ok=%v err=%v", ok, err)
	}
}

func TestAvailableStartsWithBusy(t *testing.T) {
	loc, err := LoadLocation()
	if err != nil {
		t.Fatal(err)
	}
	open := time.Date(2026, 9, 14, 9, 0, 0, 0, loc)
	close := time.Date(2026, 9, 14, 12, 0, 0, 0, loc)
	window := DayWindow{Open: open, Close: close}

	busy := []BusyInterval{
		{
			Start: time.Date(2026, 9, 14, 10, 0, 0, 0, loc),
			End:   time.Date(2026, 9, 14, 11, 0, 0, 0, loc),
		},
	}
	starts := AvailableStarts(window, 45*time.Minute, busy, nil)
	// 9:00, 9:15 fit before 10:00; 11:00, 11:15 fit before 12:00
	want := []string{"09:00", "09:15", "11:00", "11:15"}
	if len(starts) != len(want) {
		t.Fatalf("got %d starts %v, want %v", len(starts), formatStarts(starts), want)
	}
	for i, s := range starts {
		got := s.Format("15:04")
		if got != want[i] {
			t.Fatalf("start[%d]=%s want %s", i, got, want[i])
		}
	}
}

func TestAvailableStartsDropsPast(t *testing.T) {
	loc, err := LoadLocation()
	if err != nil {
		t.Fatal(err)
	}
	open := time.Date(2026, 9, 14, 9, 0, 0, 0, loc)
	close := time.Date(2026, 9, 14, 10, 0, 0, 0, loc)
	window := DayWindow{Open: open, Close: close}
	now := time.Date(2026, 9, 14, 9, 20, 0, 0, loc)
	starts := AvailableStarts(window, 30*time.Minute, nil, &now)
	// latest start 9:30; past relative to 9:20 dropped → only 9:30
	if len(starts) != 1 || starts[0].Format("15:04") != "09:30" {
		t.Fatalf("got %v", formatStarts(starts))
	}
}

func formatStarts(starts []time.Time) []string {
	out := make([]string, len(starts))
	for i, s := range starts {
		out[i] = s.Format("15:04")
	}
	return out
}

func TestOverlaps(t *testing.T) {
	a := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	c := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)
	d := time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC)
	if !Overlaps(a, b, c, d) {
		t.Fatal("expected overlap")
	}
	if Overlaps(a, b, b, d) {
		t.Fatal("touching end should not overlap")
	}
}
