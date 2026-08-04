package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/model"
)

func AppendEvent(event model.Event) error {
	path := filepath.Join(XpoDir(), "issues.db")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
