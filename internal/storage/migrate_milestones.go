package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ErrOldReleaseData is returned by the event readers when the log still has
// Milestone data under the names it had before 1.6, when the feature was
// called Releases.
var ErrOldReleaseData = errors.New("This board has Release data from before 1.6. Run `xpo doctor --fix` to migrate it.")

// The old names are matched as JSON keys with their quotes. Free text holds
// them escaped (\"release_id\"), so titles, descriptions and comments never
// match.
var (
	oldTypeKey     = []byte(`"type":"RELEASE_`)
	oldIDKey       = []byte(`"id":"rel-`)
	oldCauseKey    = []byte(`"cause":"rel-`)
	oldRefKey      = []byte(`"release_id":`)
	oldRefValueKey = []byte(`"release_id":"rel-`)
	oldMovedToKey  = []byte(`"moved_to":"rel-`)
	oldDraftKey    = []byte(`"release":`)
	newTypeKey     = []byte(`"type":"MILESTONE_`)
	newIDKey       = []byte(`"id":"ms-`)
	newCauseKey    = []byte(`"cause":"ms-`)
	newRefKey      = []byte(`"milestone_id":`)
	newRefValueKey = []byte(`"milestone_id":"ms-`)
	newMovedToKey  = []byte(`"moved_to":"ms-`)
	newDraftKey    = []byte(`"milestone":`)
)

// MigrationResult reports what MigrateMilestones changed.
type MigrationResult struct {
	Files      []string // files rewritten, relative to .xpo
	Lines      int      // event lines rewritten
	Duplicates int      // lines dropped as exact copies of another line
}

// lineHasOldReleaseData reports whether one event line uses the old names.
func lineHasOldReleaseData(line []byte) bool {
	return bytes.Contains(line, oldTypeKey) || bytes.Contains(line, oldIDKey) ||
		bytes.Contains(line, oldCauseKey) || bytes.Contains(line, oldRefKey)
}

// migrateLine rewrites the old names in one event line. All other bytes stay
// as they are, so the git diff shows only the renamed keys and IDs.
func migrateLine(line []byte) []byte {
	milestoneEvent := bytes.Contains(line, oldTypeKey)
	line = bytes.ReplaceAll(line, oldTypeKey, newTypeKey)
	line = bytes.ReplaceAll(line, oldIDKey, newIDKey)
	line = bytes.ReplaceAll(line, oldCauseKey, newCauseKey)
	line = bytes.ReplaceAll(line, oldRefValueKey, newRefValueKey)
	line = bytes.ReplaceAll(line, oldRefKey, newRefKey)
	if milestoneEvent {
		line = bytes.ReplaceAll(line, oldMovedToKey, newMovedToKey)
	}
	return line
}

// HasOldReleaseData reports whether issues.db, archive.db or an issue draft
// still uses the names from before the Milestones rename.
func HasOldReleaseData() (bool, error) {
	for _, name := range []string{"issues.db", "archive.db"} {
		data, err := os.ReadFile(filepath.Join(XpoDir(), name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		for _, line := range bytes.Split(data, []byte("\n")) {
			if lineHasOldReleaseData(line) {
				return true, nil
			}
		}
	}
	drafts, err := draftFiles()
	if err != nil {
		return false, err
	}
	for _, path := range drafts {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		if bytes.Contains(data, oldDraftKey) {
			return true, nil
		}
	}
	return false, nil
}

// MigrateMilestones rewrites the Release names in issues.db, archive.db and
// the issue drafts to the Milestone names, under the events lock. Each
// changed file is first copied to <file>.bak. A rewritten line that is now
// an exact copy of another line (a pre-migration branch merged with
// merge=union) is dropped; the first copy stays. Running it again on a
// migrated board changes nothing.
func MigrateMilestones() (MigrationResult, error) {
	var result MigrationResult
	err := withEventsLock(func() error {
		for _, name := range []string{"issues.db", "archive.db"} {
			lines, dups, err := migrateLog(filepath.Join(XpoDir(), name))
			if err != nil {
				return fmt.Errorf("failed to migrate %s: %w", name, err)
			}
			if lines > 0 {
				result.Files = append(result.Files, name)
				result.Lines += lines
				result.Duplicates += dups
			}
		}
		drafts, err := draftFiles()
		if err != nil {
			return err
		}
		for _, path := range drafts {
			changed, err := migrateDraft(path)
			if err != nil {
				return fmt.Errorf("failed to migrate %s: %w", filepath.Base(path), err)
			}
			if changed {
				result.Files = append(result.Files, filepath.Join("drafts", filepath.Base(path)))
			}
		}
		return nil
	})
	return result, err
}

// migrateLog rewrites one event log. Callers hold the events lock.
func migrateLog(path string) (lines, dups int, err error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}

	all := bytes.SplitAfter(data, []byte("\n"))
	rewritten := make(map[string]bool)
	for i, line := range all {
		if lineHasOldReleaseData(line) {
			all[i] = migrateLine(line)
			rewritten[string(all[i])] = true
			lines++
		}
	}
	if lines == 0 {
		return 0, 0, nil
	}

	var out bytes.Buffer
	seen := make(map[string]bool)
	for _, line := range all {
		key := string(line)
		if rewritten[key] {
			if seen[key] {
				dups++
				continue
			}
			seen[key] = true
		}
		out.Write(line)
	}

	if err := copyFile(path, path+".bak"); err != nil {
		return 0, 0, fmt.Errorf("failed to back up: %w", err)
	}
	if err := rewriteFile(path, out.Bytes(), nil); err != nil {
		return 0, 0, err
	}
	return lines, dups, nil
}

func migrateDraft(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if !bytes.Contains(data, oldDraftKey) {
		return false, nil
	}
	if err := copyFile(path, path+".bak"); err != nil {
		return false, fmt.Errorf("failed to back up: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, bytes.ReplaceAll(data, oldDraftKey, newDraftKey), 0644); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, os.Rename(tmp, path)
}

func draftFiles() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(DraftsDir(), "*.json"))
	sort.Strings(paths)
	return paths, err
}
