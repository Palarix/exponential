package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/palarix/exponential/internal/model"
)

var (
	// ErrInvalidDraftID is returned for IDs that do not match draftIDRe.
	ErrInvalidDraftID = errors.New("invalid draft ID")
	// ErrDraftNotFound is returned when no file exists for a draft ID.
	ErrDraftNotFound = errors.New("draft not found")
)

// draftIDRe is checked before every file access, so a draft ID can never
// name a path outside .xpo/drafts/.
var draftIDRe = regexp.MustCompile(`^draft-[0-9a-f]{8}$`)

// ValidDraftID reports whether id is a well-formed issue draft ID.
func ValidDraftID(id string) bool {
	return draftIDRe.MatchString(id)
}

// NewDraftID returns a random ID of the form draft-<8 hex>.
func NewDraftID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "draft-" + hex.EncodeToString(b), nil
}

// DraftsDir returns the absolute path to .xpo/drafts on the hub.
func DraftsDir() string {
	return filepath.Join(XpoDir(), "drafts")
}

func draftPath(id string) (string, error) {
	if !ValidDraftID(id) {
		return "", ErrInvalidDraftID
	}
	return filepath.Join(DraftsDir(), id+".json"), nil
}

// ensureDraftsDir creates .xpo/drafts/ with a .gitignore that keeps the
// directory's contents out of git: issue drafts are personal work in progress.
func ensureDraftsDir() error {
	dir := DraftsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	gi := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gi); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(gi, []byte("*\n"), 0644)
	}
	return nil
}

// WriteDraft creates or replaces the file for d.ID. It writes a temp file
// and renames it, so a reader never sees a partial file.
func WriteDraft(d model.IssueDraft) error {
	path, err := draftPath(d.ID)
	if err != nil {
		return err
	}
	if err := ensureDraftsDir(); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep "Name <email>" readable
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(DraftsDir(), ".tmp-"+d.ID+"-*")
	if err != nil {
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// ReadDraft returns the issue draft with the given ID.
func ReadDraft(id string) (*model.IssueDraft, error) {
	path, err := draftPath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrDraftNotFound
	}
	if err != nil {
		return nil, err
	}
	var d model.IssueDraft
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("draft %s: %w", id, err)
	}
	return &d, nil
}

// ListDrafts returns all issue drafts, newest UpdatedAt first. A missing
// directory yields an empty list. Unreadable or corrupt files are skipped
// with a warning so one bad file does not hide the others.
func ListDrafts() ([]*model.IssueDraft, error) {
	entries, err := os.ReadDir(DraftsDir())
	if errors.Is(err, os.ErrNotExist) {
		return []*model.IssueDraft{}, nil
	}
	if err != nil {
		return nil, err
	}
	drafts := []*model.IssueDraft{}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if e.IsDir() || !ok || !ValidDraftID(id) {
			continue
		}
		d, err := ReadDraft(id)
		if err != nil {
			log.Printf("skipping issue draft %s: %v", e.Name(), err)
			continue
		}
		drafts = append(drafts, d)
	}
	sort.SliceStable(drafts, func(i, j int) bool {
		return drafts[i].UpdatedAt.After(drafts[j].UpdatedAt)
	})
	return drafts, nil
}

// DeleteDraft removes the file for id.
func DeleteDraft(id string) error {
	path, err := draftPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return ErrDraftNotFound
	} else if err != nil {
		return err
	}
	return nil
}
