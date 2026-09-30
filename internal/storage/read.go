package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/model"
)

// ReadEvents reads every event in issues.db. Readers don't take the events
// lock, so they can catch an append halfway through its write: a final line
// with no newline that doesn't parse is that torn append and is skipped.
func ReadEvents() ([]model.Event, error) {
	data, err := os.ReadFile(filepath.Join(XpoDir(), "issues.db"))
	if os.IsNotExist(err) {
		return []model.Event{}, nil
	}
	if err != nil {
		return nil, err
	}

	var events []model.Event
	for len(data) > 0 {
		line, rest, terminated := bytes.Cut(data, []byte("\n"))
		data = rest
		var evt model.Event
		if err := json.Unmarshal(line, &evt); err != nil {
			if !terminated {
				break
			}
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
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
