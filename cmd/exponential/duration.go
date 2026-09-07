package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseDuration parses human-friendly duration strings like "30m", "24h",
// "7d", "2w". Go's time.ParseDuration handles m/s/h; this adds d and w.
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	suffix := s[len(s)-1]
	numStr := s[:len(s)-1]

	switch suffix {
	case 'd':
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
		}
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}
