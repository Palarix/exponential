package config

import (
	"fmt"
	"strings"
	"time"
)

type Cycle struct {
	ID     string    // start date as YYYY-MM-DD
	Number int       // sequential, 1-based from anchor
	Start  time.Time // first day of cycle (inclusive)
	End    time.Time // last day of cycle (inclusive)
}

func (c Cycle) Contains(date time.Time) bool {
	d := truncateToDay(date)
	return !d.Before(c.Start) && !d.After(c.End)
}

func (c Cycle) IsBefore(other Cycle) bool {
	return c.Start.Before(other.Start)
}

func parseCycleDurationDays(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1w":
		return 7, nil
	case "2w":
		return 14, nil
	case "3w":
		return 21, nil
	case "4w":
		return 28, nil
	}
	return 0, fmt.Errorf("invalid cycle duration %q (allowed: 1w, 2w, 3w, 4w)", s)
}

var weekdayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

func parseStartDay(s string) (time.Weekday, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if wd, ok := weekdayNames[s]; ok {
		return wd, nil
	}
	return 0, fmt.Errorf("invalid start day %q", s)
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (cfg CycleConfig) DurationDays() int {
	d, _ := parseCycleDurationDays(cfg.Duration)
	return d
}

func (cfg CycleConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}
	if _, err := parseCycleDurationDays(cfg.Duration); err != nil {
		return err
	}
	if _, err := parseStartDay(cfg.StartDay); err != nil {
		return err
	}
	if cfg.AnchorDate != "" {
		if _, err := time.Parse("2006-01-02", cfg.AnchorDate); err != nil {
			return fmt.Errorf("invalid anchor_date %q (expected YYYY-MM-DD)", cfg.AnchorDate)
		}
	}
	return nil
}

func (cfg CycleConfig) anchor() time.Time {
	startDay, _ := parseStartDay(cfg.StartDay)

	if cfg.AnchorDate != "" {
		t, _ := time.Parse("2006-01-02", cfg.AnchorDate)
		return truncateToDay(t)
	}

	now := truncateToDay(time.Now())
	offset := (int(now.Weekday()) - int(startDay) + 7) % 7
	return now.AddDate(0, 0, -offset)
}

func (cfg CycleConfig) cycleForDate(date time.Time) Cycle {
	anchor := cfg.anchor()
	durationDays, _ := parseCycleDurationDays(cfg.Duration)
	return cycleForDateFromAnchor(anchor, durationDays, date)
}

func cycleForDateFromAnchor(anchor time.Time, durationDays int, date time.Time) Cycle {
	date = truncateToDay(date)
	delta := int(date.Sub(anchor).Hours() / 24)

	n := delta / durationDays
	if delta < 0 && delta%durationDays != 0 {
		n--
	}

	start := anchor.AddDate(0, 0, n*durationDays)
	end := start.AddDate(0, 0, durationDays-1)

	return Cycle{
		ID:     start.Format("2006-01-02"),
		Number: n + 1,
		Start:  start,
		End:    end,
	}
}

func (cfg CycleConfig) CurrentCycle() Cycle {
	return cfg.cycleForDate(time.Now())
}

func (cfg CycleConfig) CycleForDate(date time.Time) Cycle {
	return cfg.cycleForDate(date)
}

func (cfg CycleConfig) CycleForID(id string) (Cycle, error) {
	t, err := time.Parse("2006-01-02", id)
	if err != nil {
		return Cycle{}, fmt.Errorf("invalid cycle ID %q (expected YYYY-MM-DD)", id)
	}
	return cfg.cycleForDate(t), nil
}

func (cfg CycleConfig) IsPastCycle(cycleID string, now time.Time) bool {
	current := cfg.cycleForDate(now)
	cycle, err := cfg.CycleForID(cycleID)
	if err != nil {
		return false
	}
	return cycle.IsBefore(current)
}

func (cfg CycleConfig) EnumerateCycles(now time.Time, pastCount, futureCount int) []Cycle {
	current := cfg.cycleForDate(now)
	durationDays, _ := parseCycleDurationDays(cfg.Duration)

	total := pastCount + 1 + futureCount
	cycles := make([]Cycle, 0, total)

	startOffset := -pastCount
	for i := 0; i < total; i++ {
		offset := startOffset + i
		start := current.Start.AddDate(0, 0, offset*durationDays)
		end := start.AddDate(0, 0, durationDays-1)
		cycles = append(cycles, Cycle{
			ID:     start.Format("2006-01-02"),
			Number: i + 1,
			Start:  start,
			End:    end,
		})
	}

	return cycles
}
