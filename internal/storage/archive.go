package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kuyio/beats/internal/model"
)

// ArchiveEvents moves the specified events to archive.db and rewrites issues.db with the active events.
func ArchiveEvents(active []model.Event, archived []model.Event) error {
	issuesPath := filepath.Join(".beats", "issues.db")
	archivePath := filepath.Join(".beats", "archive.db")

	// 1. Backup existing issues.db
	backupPath := fmt.Sprintf("%s.%s.bak", issuesPath, time.Now().Format("20060102150405"))
	if err := copyFile(issuesPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup issues db: %w", err)
	}

	// 2. Append archived events to archive.db
	archiveFile, err := os.OpenFile(archivePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open archive db: %w", err)
	}
	defer archiveFile.Close()

	for _, evt := range archived {
		bytes, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.ID, err)
		}
		if _, err := archiveFile.Write(bytes); err != nil {
			return fmt.Errorf("failed to write to archive db: %w", err)
		}
		if _, err := archiveFile.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to archive db: %w", err)
		}
	}

	// 3. Rewrite issues.db with active events
	// Use O_TRUNC to clear the file
	issuesFile, err := os.OpenFile(issuesPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open issues db for rewrite: %w", err)
	}
	defer issuesFile.Close()

	for _, evt := range active {
		bytes, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", evt.ID, err)
		}
		if _, err := issuesFile.Write(bytes); err != nil {
			return fmt.Errorf("failed to write to issues db: %w", err)
		}
		if _, err := issuesFile.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to issues db: %w", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
