package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

// ArchiveEvents moves the specified events to archive.db and rewrites issues.db with the active events.
func ArchiveEvents(active []model.Event, archived []model.Event) error {
	store := RefStore()

	// Append archived events to archive.db
	existingArchive, err := store.ReadFile("archive.db")
	if err != nil {
		return fmt.Errorf("failed to read archive.db: %w", err)
	}

	var archiveLines strings.Builder
	archiveLines.WriteString(existingArchive)
	for _, evt := range archived {
		b, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.ID, err)
		}
		archiveLines.Write(b)
		archiveLines.WriteString("\n")
	}

	if err := store.WriteFile("archive.db", archiveLines.String(), "xpo: archive events"); err != nil {
		return fmt.Errorf("failed to write archive.db: %w", err)
	}

	// Rewrite issues.db with active events
	var activeLines strings.Builder
	for _, evt := range active {
		b, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.ID, err)
		}
		activeLines.Write(b)
		activeLines.WriteString("\n")
	}

	if err := store.WriteFile("issues.db", activeLines.String(), "xpo: rewrite after archive"); err != nil {
		return fmt.Errorf("failed to rewrite issues.db: %w", err)
	}

	return nil
}
