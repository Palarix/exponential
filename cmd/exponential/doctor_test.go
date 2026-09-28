package main

import (
	"testing"

	"github.com/palarix/exponential/internal/storage"
)

func TestDescribeOrderViolations(t *testing.T) {
	var vs []storage.EventOrderViolation
	for i := 0; i < 12; i++ {
		kind := storage.OrderViolationOutOfOrder
		if i%3 == 0 {
			kind = storage.OrderViolationTie
		}
		vs = append(vs, storage.EventOrderViolation{Line: 10 + i, Kind: kind})
	}

	summary, lines := describeOrderViolations(vs)
	if want := "Issues database has 8 out-of-order and 4 tied timestamps"; summary != want {
		t.Errorf("summary = %q, want %q", summary, want)
	}
	if want := "lines 10, 11, 12, 13, 14, 15, 16, 17, 18, 19 …and 2 more"; lines != want {
		t.Errorf("lines = %q, want %q", lines, want)
	}
}

func TestDescribeOrderViolationsFew(t *testing.T) {
	summary, lines := describeOrderViolations([]storage.EventOrderViolation{
		{Line: 3, Kind: storage.OrderViolationOutOfOrder},
	})
	if want := "Issues database has 1 out-of-order and 0 tied timestamps"; summary != want {
		t.Errorf("summary = %q, want %q", summary, want)
	}
	if want := "line 3"; lines != want {
		t.Errorf("lines = %q, want %q", lines, want)
	}
}
