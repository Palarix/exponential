package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type OrderViolationKind string

const (
	// OrderViolationOutOfOrder: the line's created_at is earlier than the line before it.
	OrderViolationOutOfOrder OrderViolationKind = "out_of_order"
	// OrderViolationTie: the line's created_at equals the line before it.
	OrderViolationTie OrderViolationKind = "tie"
)

// EventOrderViolation is a line in issues.db whose created_at does not
// strictly follow the line before it.
type EventOrderViolation struct {
	Line int
	Kind OrderViolationKind
}

// CheckEventOrder verifies that created_at strictly increases in file order
// across issues.db. File order is authoritative for projection; timestamps
// that disagree with it mislead any consumer that orders events by time.
func CheckEventOrder() ([]EventOrderViolation, error) {
	f, err := os.Open(filepath.Join(XpoDir(), "issues.db"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var violations []EventOrderViolation
	var prev time.Time
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		var evt struct {
			CreatedAt time.Time `json:"created_at"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return nil, err
		}
		if line > 1 {
			switch {
			case evt.CreatedAt.Before(prev):
				violations = append(violations, EventOrderViolation{Line: line, Kind: OrderViolationOutOfOrder})
			case evt.CreatedAt.Equal(prev):
				violations = append(violations, EventOrderViolation{Line: line, Kind: OrderViolationTie})
			}
		}
		prev = evt.CreatedAt
	}
	return violations, scanner.Err()
}
