package controller

import (
	"testing"
)

func TestNormalizeAndValidateHairstyleInput(t *testing.T) {
	valid := HairstyleInput{
		Name:                 "Fade",
		Description:          "Classic fade",
		WalkInPriceKobo:      500000,
		HomeServicePriceKobo: 800000,
		DurationMinutes:      45,
		ImageURLs:            []string{"https://example.com/a.jpg"},
		Active:               true,
	}

	out, err := normalizeAndValidateHairstyleInput(valid)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Name != "Fade" || len(out.ImageURLs) != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}

	noImage := valid
	noImage.ImageURLs = nil
	if _, err := normalizeAndValidateHairstyleInput(noImage); err == nil {
		t.Fatal("expected error for missing images")
	}

	tooMany := valid
	tooMany.ImageURLs = []string{"a", "b", "c", "d"}
	if _, err := normalizeAndValidateHairstyleInput(tooMany); err == nil {
		t.Fatal("expected error for too many images")
	}

	badDuration := valid
	badDuration.DurationMinutes = 0
	if _, err := normalizeAndValidateHairstyleInput(badDuration); err == nil {
		t.Fatal("expected error for duration")
	}
}

func TestMediaURLsToDelete(t *testing.T) {
	prev := []string{"a", "b", "c"}
	curr := []string{"b", "c"}
	got := mediaURLsToDelete(prev, curr)
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("got %v", got)
	}
}
