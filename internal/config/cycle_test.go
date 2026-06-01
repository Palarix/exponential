package config

import (
	"testing"
	"time"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCycleForDate_TwoWeek(t *testing.T) {
	cfg := CycleConfig{
		Enabled:    true,
		Duration:   "2w",
		StartDay:   "monday",
		AnchorDate: "2026-05-25", // a Sunday — but anchor is taken as-is
	}

	// Use a fixed anchor to avoid time.Now() dependency
	anchor := date("2026-05-25")
	durationDays := 14

	tests := []struct {
		name       string
		date       string
		wantID     string
		wantNumber int
		wantStart  string
		wantEnd    string
	}{
		{
			name:       "anchor date itself",
			date:       "2026-05-25",
			wantID:     "2026-05-25",
			wantNumber: 1,
			wantStart:  "2026-05-25",
			wantEnd:    "2026-06-07",
		},
		{
			name:       "mid first cycle",
			date:       "2026-06-01",
			wantID:     "2026-05-25",
			wantNumber: 1,
			wantStart:  "2026-05-25",
			wantEnd:    "2026-06-07",
		},
		{
			name:       "last day of first cycle",
			date:       "2026-06-07",
			wantID:     "2026-05-25",
			wantNumber: 1,
			wantStart:  "2026-05-25",
			wantEnd:    "2026-06-07",
		},
		{
			name:       "first day of second cycle",
			date:       "2026-06-08",
			wantID:     "2026-06-08",
			wantNumber: 2,
			wantStart:  "2026-06-08",
			wantEnd:    "2026-06-21",
		},
		{
			name:       "before anchor",
			date:       "2026-05-20",
			wantID:     "2026-05-11",
			wantNumber: 0,
			wantStart:  "2026-05-11",
			wantEnd:    "2026-05-24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cycleForDateFromAnchor(anchor, durationDays, date(tt.date))
			if c.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", c.ID, tt.wantID)
			}
			if c.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", c.Number, tt.wantNumber)
			}
			if c.Start.Format("2006-01-02") != tt.wantStart {
				t.Errorf("Start = %s, want %s", c.Start.Format("2006-01-02"), tt.wantStart)
			}
			if c.End.Format("2006-01-02") != tt.wantEnd {
				t.Errorf("End = %s, want %s", c.End.Format("2006-01-02"), tt.wantEnd)
			}
		})
	}

	_ = cfg // cfg used above for documentation; direct anchor tested via cycleForDateFromAnchor
}

func TestCycleForDate_OneWeek(t *testing.T) {
	anchor := date("2026-06-01") // Monday
	durationDays := 7

	c := cycleForDateFromAnchor(anchor, durationDays, date("2026-06-04"))
	if c.ID != "2026-06-01" {
		t.Errorf("ID = %q, want %q", c.ID, "2026-06-01")
	}
	if c.Start.Format("2006-01-02") != "2026-06-01" {
		t.Errorf("Start = %s, want 2026-06-01", c.Start.Format("2006-01-02"))
	}
	if c.End.Format("2006-01-02") != "2026-06-07" {
		t.Errorf("End = %s, want 2026-06-07", c.End.Format("2006-01-02"))
	}
}

func TestCycleContains(t *testing.T) {
	anchor := date("2026-06-01")
	c := cycleForDateFromAnchor(anchor, 14, date("2026-06-05"))

	if !c.Contains(date("2026-06-01")) {
		t.Error("should contain start date")
	}
	if !c.Contains(date("2026-06-14")) {
		t.Error("should contain end date")
	}
	if c.Contains(date("2026-05-31")) {
		t.Error("should not contain day before start")
	}
	if c.Contains(date("2026-06-15")) {
		t.Error("should not contain day after end")
	}
}

func TestIsPastCycle(t *testing.T) {
	cfg := CycleConfig{
		Enabled:    true,
		Duration:   "2w",
		StartDay:   "monday",
		AnchorDate: "2026-05-25",
	}

	now := date("2026-06-10") // falls in cycle 2 (Jun 8 - Jun 21)

	if !cfg.IsPastCycle("2026-05-25", now) {
		t.Error("cycle starting 2026-05-25 should be past")
	}
	if cfg.IsPastCycle("2026-06-08", now) {
		t.Error("current cycle should not be past")
	}
	if cfg.IsPastCycle("2026-06-22", now) {
		t.Error("future cycle should not be past")
	}
}

func TestEnumerateCycles(t *testing.T) {
	cfg := CycleConfig{
		Enabled:    true,
		Duration:   "2w",
		StartDay:   "monday",
		AnchorDate: "2026-05-25",
	}

	now := date("2026-06-10")
	cycles := cfg.EnumerateCycles(now, 1, 2)

	if len(cycles) != 4 {
		t.Fatalf("expected 4 cycles, got %d", len(cycles))
	}

	// Previous cycle
	if cycles[0].ID != "2026-05-25" {
		t.Errorf("past cycle ID = %q, want %q", cycles[0].ID, "2026-05-25")
	}
	// Current cycle
	if cycles[1].ID != "2026-06-08" {
		t.Errorf("current cycle ID = %q, want %q", cycles[1].ID, "2026-06-08")
	}
	// Future cycles
	if cycles[2].ID != "2026-06-22" {
		t.Errorf("future cycle 1 ID = %q, want %q", cycles[2].ID, "2026-06-22")
	}
	if cycles[3].ID != "2026-07-06" {
		t.Errorf("future cycle 2 ID = %q, want %q", cycles[3].ID, "2026-07-06")
	}
}

func TestCycleForID(t *testing.T) {
	cfg := CycleConfig{
		Enabled:    true,
		Duration:   "2w",
		StartDay:   "monday",
		AnchorDate: "2026-05-25",
	}

	c, err := cfg.CycleForID("2026-06-08")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Number != 2 {
		t.Errorf("Number = %d, want 2", c.Number)
	}

	_, err = cfg.CycleForID("not-a-date")
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestYearBoundary(t *testing.T) {
	anchor := date("2026-12-28") // Monday
	c := cycleForDateFromAnchor(anchor, 14, date("2027-01-05"))

	if c.ID != "2026-12-28" {
		t.Errorf("ID = %q, want %q", c.ID, "2026-12-28")
	}
	if c.End.Format("2006-01-02") != "2027-01-10" {
		t.Errorf("End = %s, want 2027-01-10", c.End.Format("2006-01-02"))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CycleConfig
		wantErr bool
	}{
		{"disabled", CycleConfig{Enabled: false}, false},
		{"valid", CycleConfig{Enabled: true, Duration: "2w", StartDay: "monday"}, false},
		{"with anchor", CycleConfig{Enabled: true, Duration: "1w", StartDay: "friday", AnchorDate: "2026-01-02"}, false},
		{"bad duration", CycleConfig{Enabled: true, Duration: "5w", StartDay: "monday"}, true},
		{"bad day", CycleConfig{Enabled: true, Duration: "2w", StartDay: "funday"}, true},
		{"bad anchor", CycleConfig{Enabled: true, Duration: "2w", StartDay: "monday", AnchorDate: "nope"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseCycleDurationDays(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"1w", 7, false},
		{"2w", 14, false},
		{"3w", 21, false},
		{"4w", 28, false},
		{"5w", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		got, err := parseCycleDurationDays(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseCycleDurationDays(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("parseCycleDurationDays(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
