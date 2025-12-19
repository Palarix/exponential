package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/beats/internal/model"
)

func ReadEvents() ([]model.Event, error) {
	f, err := os.Open(filepath.Join(".beats", "issues.db"))
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
