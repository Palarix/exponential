package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/model"
)

// ArchiveResult reports which requested issues were archived and which were
// skipped because they were no longer deleted or terminal.
type ArchiveResult struct {
	Archived []string
	Skipped  []string
}

// ArchiveEvents moves every event of the given issues from issues.db to
// archive.db. It re-reads the log under the events lock, so events written
// since the caller chose the IDs are kept, and it re-checks each issue:
// one that is no longer deleted or terminal (reopened since the preview)
// is skipped and stays in issues.db.
func ArchiveEvents(issueIDs map[string]bool) (ArchiveResult, error) {
	var result ArchiveResult
	err := withEventsLock(func() error {
		var err error
		result, err = archiveEvents(issueIDs)
		return err
	})
	return result, err
}

func archiveEvents(issueIDs map[string]bool) (ArchiveResult, error) {
	var result ArchiveResult
	issuesPath := filepath.Join(XpoDir(), "issues.db")
	archivePath := filepath.Join(XpoDir(), "archive.db")

	events, err := ReadEvents()
	if err != nil {
		return result, fmt.Errorf("failed to read issues db: %w", err)
	}

	state := make(map[string]*model.Issue)
	for _, evt := range events {
		applyStateEvent(state, evt)
	}
	archive := make(map[string]bool)
	for id := range issueIDs {
		issue, ok := state[id]
		if ok && (issue.Deleted || model.IsTerminal(issue.Status)) {
			archive[id] = true
			result.Archived = append(result.Archived, id)
		} else {
			result.Skipped = append(result.Skipped, id)
		}
	}
	sort.Strings(result.Archived)
	sort.Strings(result.Skipped)

	var active, archived []model.Event
	for _, evt := range events {
		if archive[evt.ID] {
			archived = append(archived, evt)
		} else {
			active = append(active, evt)
		}
	}

	// 1. Backup existing issues.db
	backupPath := fmt.Sprintf("%s.%s.bak", issuesPath, time.Now().Format("20060102150405"))
	if err := copyFile(issuesPath, backupPath); err != nil {
		return result, fmt.Errorf("failed to backup issues db: %w", err)
	}

	// 2. Append archived events to archive.db
	archiveFile, err := os.OpenFile(archivePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return result, fmt.Errorf("failed to open archive db: %w", err)
	}
	defer archiveFile.Close()

	for _, evt := range archived {
		bytes, err := json.Marshal(evt)
		if err != nil {
			return result, fmt.Errorf("failed to marshal event %s: %w", evt.ID, err)
		}
		bytes = append(bytes, '\n')
		if _, err := archiveFile.Write(bytes); err != nil {
			return result, fmt.Errorf("failed to write to archive db: %w", err)
		}
	}

	// 3. Replace issues.db with the active events
	if err := rewriteFile(issuesPath, nil, active); err != nil {
		return result, fmt.Errorf("failed to rewrite issues db: %w", err)
	}

	return result, nil
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
