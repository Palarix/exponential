package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/model"
)

func ReadEvents() ([]model.Event, error) {
	f, err := os.Open(filepath.Join(XpoDir(), "issues.db"))
	if os.IsNotExist(err) {
		return []model.Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []model.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, scanner.Err()
}

// ValidateEvents reads issues.db and checks every line parses as valid JSON.
// On success it returns (eventCount, nil). On failure it returns (lineNumber, error).
func ValidateEvents() (int, error) {
	f, err := os.Open(filepath.Join(XpoDir(), "issues.db"))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return line, err
		}
	}
	if err := scanner.Err(); err != nil {
		return line, err
	}
	return line, nil
}

// ReadArchivedEvents reads events from the archive database
func ReadArchivedEvents() ([]model.Event, error) {
	f, err := os.Open(filepath.Join(XpoDir(), "archive.db"))
	if os.IsNotExist(err) {
		return []model.Event{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []model.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, scanner.Err()
}
