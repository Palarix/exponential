package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/beats/internal/model"
)

func AppendEvent(event model.Event) error {
	f, err := os.OpenFile(filepath.Join(".beats", "issues.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := f.Write(bytes); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}

	return nil
}
