package schedule

import (
	"fmt"
	"time"

	"haircutz/backend/internal/model"
)

const (
	LocationName = "Africa/Lagos"
	SlotStep     = 15 * time.Minute
)

// DayWindow is the open interval [Open, Close) in the shop timezone.
type DayWindow struct {
	Open  time.Time
	Close time.Time
}

func LoadLocation() (*time.Location, error) {
	loc, err := time.LoadLocation(LocationName)
	if err != nil {
		return nil, fmt.Errorf("load timezone %s: %w", LocationName, err)
	}
	return loc, nil
}

// WindowForDay returns shop hours for the given calendar date and service type in loc.
// ok=false means closed (e.g. home service on Sunday).
func WindowForDay(date time.Time, service model.ServiceType, loc *time.Location) (DayWindow, bool, error) {
	if loc == nil {
		return DayWindow{}, false, fmt.Errorf("timezone is required")
	}
	if !service.Valid() {
		return DayWindow{}, false, fmt.Errorf("invalid service type")
	}

	y, m, d := date.In(loc).Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, loc)
	weekday := dayStart.Weekday()

	var openHour, openMin, closeHour, closeMin int

	switch service {
	case model.ServiceWalkIn:
		switch weekday {
		case time.Sunday:
			openHour, openMin = 12, 0
			closeHour, closeMin = 22, 0
		default: // Mon–Sat
			openHour, openMin = 9, 0
			closeHour, closeMin = 22, 0
		}
	case model.ServiceHomeService:
		if weekday == time.Sunday {
			return DayWindow{}, false, nil
		}
		openHour, openMin = 10, 0
		closeHour, closeMin = 17, 0
	}

	open := time.Date(y, m, d, openHour, openMin, 0, 0, loc)
	close := time.Date(y, m, d, closeHour, closeMin, 0, 0, loc)
	return DayWindow{Open: open, Close: close}, true, nil
}

type BusyInterval struct {
	Start time.Time
	End   time.Time
}

// AvailableStarts returns slot start times (in loc) every SlotStep inside free gaps.
// Duration must fit entirely before Close / next busy block.
// If now is set and the day is today, starts in the past relative to now are dropped.
func AvailableStarts(
	window DayWindow,
	duration time.Duration,
	busy []BusyInterval,
	now *time.Time,
) []time.Time {
	if duration <= 0 || !window.Close.After(window.Open) {
		return nil
	}

	merged := mergeBusy(busy, window.Open, window.Close)
	gaps := freeGaps(window.Open, window.Close, merged)

	var starts []time.Time
	for _, gap := range gaps {
		latestStart := gap.End.Add(-duration)
		if latestStart.Before(gap.Start) {
			continue
		}
		for t := alignUp(gap.Start, SlotStep); !t.After(latestStart); t = t.Add(SlotStep) {
			if now != nil && !t.After(*now) {
				continue
			}
			starts = append(starts, t)
		}
	}
	return starts
}

func mergeBusy(busy []BusyInterval, open, close time.Time) []BusyInterval {
	clipped := make([]BusyInterval, 0, len(busy))
	for _, b := range busy {
		start := b.Start
		end := b.End
		if end.Before(open) || !start.Before(close) {
			continue
		}
		if start.Before(open) {
			start = open
		}
		if end.After(close) {
			end = close
		}
		if end.After(start) {
			clipped = append(clipped, BusyInterval{Start: start, End: end})
		}
	}
	if len(clipped) == 0 {
		return nil
	}

	// Sort by start
	for i := 0; i < len(clipped); i++ {
		for j := i + 1; j < len(clipped); j++ {
			if clipped[j].Start.Before(clipped[i].Start) {
				clipped[i], clipped[j] = clipped[j], clipped[i]
			}
		}
	}

	out := []BusyInterval{clipped[0]}
	for _, b := range clipped[1:] {
		last := &out[len(out)-1]
		if !b.Start.After(last.End) {
			if b.End.After(last.End) {
				last.End = b.End
			}
			continue
		}
		out = append(out, b)
	}
	return out
}

func freeGaps(open, close time.Time, busy []BusyInterval) []BusyInterval {
	cursor := open
	gaps := make([]BusyInterval, 0, len(busy)+1)
	for _, b := range busy {
		if b.Start.After(cursor) {
			gaps = append(gaps, BusyInterval{Start: cursor, End: b.Start})
		}
		if b.End.After(cursor) {
			cursor = b.End
		}
	}
	if close.After(cursor) {
		gaps = append(gaps, BusyInterval{Start: cursor, End: close})
	}
	return gaps
}

// alignUp rounds t up to the next SlotStep boundary in its location (or leaves it if already aligned).
func alignUp(t time.Time, step time.Duration) time.Time {
	if step <= 0 {
		return t
	}
	loc := t.Location()
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	elapsed := t.Sub(midnight)
	rem := elapsed % step
	if rem == 0 {
		return t
	}
	return t.Add(step - rem)
}

// Overlaps reports whether [aStart,aEnd) overlaps [bStart,bEnd).
func Overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}
