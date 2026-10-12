package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

func sampleDraft(id string, updated time.Time) model.IssueDraft {
	return model.IssueDraft{
		ID:        id,
		CreatedAt: updated,
		UpdatedAt: updated,
		CreatedBy: "Alice <alice@example.com>",
		Payload:   model.IssueDraftPayload{Title: "T " + id, Labels: []string{"feature"}},
	}
}

func TestNewDraftID_Format(t *testing.T) {
	id, err := NewDraftID()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidDraftID(id) {
		t.Errorf("generated ID %q is not valid", id)
	}
}

func TestValidDraftID(t *testing.T) {
	cases := map[string]bool{
		"draft-0123abcd":       true,
		"draft-0123ABCD":       false,
		"draft-0123abc":        false,
		"xpo-123abc":           false,
		"../draft-0123abcd":    false,
		"draft-0123abcd/../..": false,
		"":                     false,
	}
	for id, want := range cases {
		if got := ValidDraftID(id); got != want {
			t.Errorf("ValidDraftID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestListDrafts_MissingDirIsEmpty(t *testing.T) {
	setupXpoDir(t)
	drafts, err := ListDrafts()
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 0 {
		t.Errorf("expected no drafts, got %d", len(drafts))
	}
}

func TestWriteDraft_CreatesDirWithGitignore(t *testing.T) {
	setupXpoDir(t)
	if err := WriteDraft(sampleDraft("draft-00000001", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(".xpo", "drafts", ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore: %v", err)
	}
	if string(data) != "*\n" {
		t.Errorf(".gitignore content = %q, want %q", data, "*\n")
	}
	info, err := os.Stat(filepath.Join(".xpo", "drafts", "draft-00000001.json"))
	if err != nil {
		t.Fatalf("expected draft file: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("draft file mode = %v, want 0644", info.Mode().Perm())
	}
	raw, _ := os.ReadFile(filepath.Join(".xpo", "drafts", "draft-00000001.json"))
	if !strings.Contains(string(raw), "Alice <alice@example.com>") {
		t.Errorf("expected unescaped author in file, got %s", raw)
	}
}

func TestWriteDraft_RejectsInvalidID(t *testing.T) {
	setupXpoDir(t)
	if err := WriteDraft(sampleDraft("../evil", time.Now().UTC())); !errors.Is(err, ErrInvalidDraftID) {
		t.Errorf("expected ErrInvalidDraftID, got %v", err)
	}
}

func TestReadDraft_RoundTrip(t *testing.T) {
	setupXpoDir(t)
	in := sampleDraft("draft-00000002", time.Now().UTC().Truncate(time.Second))
	in.Payload.Description = "body"
	in.Payload.AdditionalLabels = []string{"backend"}
	in.Payload.Status = "PLANNED"
	in.Payload.Estimate = 3
	in.Payload.ParentID = "xpo-abc123"
	in.Payload.Assignee = "Bob <bob@example.com>"
	if err := WriteDraft(in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadDraft(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Payload.Title != in.Payload.Title || out.Payload.Description != "body" ||
		out.Payload.Status != "PLANNED" || out.Payload.Estimate != 3 ||
		out.Payload.ParentID != "xpo-abc123" || out.Payload.Assignee != in.Payload.Assignee ||
		len(out.Payload.AdditionalLabels) != 1 || out.CreatedBy != in.CreatedBy ||
		!out.UpdatedAt.Equal(in.UpdatedAt) {
		t.Errorf("round trip mismatch: got %+v", out)
	}
}

func TestReadDraft_NotFound(t *testing.T) {
	setupXpoDir(t)
	if _, err := ReadDraft("draft-deadbeef"); !errors.Is(err, ErrDraftNotFound) {
		t.Errorf("expected ErrDraftNotFound, got %v", err)
	}
}

func TestReadDraft_RejectsInvalidID(t *testing.T) {
	setupXpoDir(t)
	if _, err := ReadDraft("../issues"); !errors.Is(err, ErrInvalidDraftID) {
		t.Errorf("expected ErrInvalidDraftID, got %v", err)
	}
}

func TestListDrafts_NewestFirstAndSkipsBadFiles(t *testing.T) {
	setupXpoDir(t)
	base := time.Now().UTC()
	WriteDraft(sampleDraft("draft-00000001", base))
	WriteDraft(sampleDraft("draft-00000002", base.Add(time.Minute)))
	dir := filepath.Join(".xpo", "drafts")
	os.WriteFile(filepath.Join(dir, "draft-00000003.json"), []byte("{not json"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644)

	drafts, err := ListDrafts()
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 2 {
		t.Fatalf("expected 2 drafts, got %d", len(drafts))
	}
	if drafts[0].ID != "draft-00000002" || drafts[1].ID != "draft-00000001" {
		t.Errorf("wrong order: %s, %s", drafts[0].ID, drafts[1].ID)
	}
}

func TestDeleteDraft(t *testing.T) {
	setupXpoDir(t)
	WriteDraft(sampleDraft("draft-00000001", time.Now().UTC()))
	if err := DeleteDraft("draft-00000001"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDraft("draft-00000001"); !errors.Is(err, ErrDraftNotFound) {
		t.Errorf("expected draft gone, got %v", err)
	}
	if err := DeleteDraft("draft-00000001"); !errors.Is(err, ErrDraftNotFound) {
		t.Errorf("expected ErrDraftNotFound on second delete, got %v", err)
	}
}

func TestWriteDraft_LeavesNoTempFiles(t *testing.T) {
	setupXpoDir(t)
	WriteDraft(sampleDraft("draft-00000001", time.Now().UTC()))
	WriteDraft(sampleDraft("draft-00000001", time.Now().UTC()))
	entries, _ := os.ReadDir(filepath.Join(".xpo", "drafts"))
	for _, e := range entries {
		if e.Name() != ".gitignore" && e.Name() != "draft-00000001.json" {
			t.Errorf("unexpected file %q", e.Name())
		}
	}
}
