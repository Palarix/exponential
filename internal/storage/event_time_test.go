package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLastEventTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "issues.db")
	t1 := `{"id":"a","created_at":"2026-10-09T12:00:00Z"}`
	t2 := `{"id":"b","payload":{"description":"` + strings.Repeat("x", 40<<10) + `"},"created_at":"2026-10-09T13:00:00.5Z"}`
	want2 := time.Date(2026, 10, 9, 13, 0, 0, 5e8, time.UTC)

	cases := []struct {
		name, content string
		want          time.Time
	}{
		{"empty", "", time.Time{}},
		{"one line", t1 + "\n", time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)},
		{"last line longer than first chunk", t1 + "\n" + t2 + "\n", want2},
		{"torn final line skipped", t1 + "\n" + t2 + "\n" + `{"id":"c","crea`, want2},
		{"only a torn line", `{"id":"c","crea`, time.Time{}},
	}
	for _, c := range cases {
		os.WriteFile(path, []byte(c.content), 0644)
		got, err := lastEventTime(path)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}

	if got, err := lastEventTime(filepath.Join(dir, "missing.db")); err != nil || !got.IsZero() {
		t.Errorf("missing file: got %s, %v; want zero, nil", got, err)
	}
}
