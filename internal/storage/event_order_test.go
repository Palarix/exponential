package storage

import (
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func TestCheckEventOrderClean(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	writeEvents(t, []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: base},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Title: strptr("B")}, CreatedAt: base.Add(time.Nanosecond)},
	})

	violations, err := CheckEventOrder()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %+v", violations)
	}
}

func TestCheckEventOrderReportsOutOfOrderAndTies(t *testing.T) {
	setupTestRepo(t)
	base := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	// May 26 pattern: a collapsed UPDATE carries a later timestamp than
	// the DOING/DONE lines that follow it.
	writeEvents(t, []model.Event{
		{ID: "a", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "A"}, CreatedAt: base},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strptr("PLANNED")}, CreatedAt: base.Add(48 * time.Minute)},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strptr("DOING")}, CreatedAt: base.Add(4 * time.Minute)},
		{ID: "a", Type: model.EventTypeUpdate, Payload: model.UpdatePayload{Status: strptr("DONE")}, CreatedAt: base.Add(17 * time.Minute)},
		{ID: "b", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "B"}, CreatedAt: base.Add(50 * time.Minute)},
		{ID: "c", Type: model.EventTypeCreate, Payload: model.CreatePayload{Title: "C"}, CreatedAt: base.Add(50 * time.Minute)},
	})

	violations, err := CheckEventOrder()
	if err != nil {
		t.Fatal(err)
	}
	want := []EventOrderViolation{
		{Line: 3, Kind: OrderViolationOutOfOrder},
		{Line: 6, Kind: OrderViolationTie},
	}
	if len(violations) != len(want) {
		t.Fatalf("violations = %+v, want %+v", violations, want)
	}
	for i := range want {
		if violations[i] != want[i] {
			t.Errorf("violation %d = %+v, want %+v", i, violations[i], want[i])
		}
	}
}
