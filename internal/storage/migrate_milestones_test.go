package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Lines as the pre-1.6 build wrote them.
const (
	oldCreate     = `{"id":"rel-a1","type":"RELEASE_CREATE","payload":{"number":1,"label":"1.0"},"created_at":"2026-10-01T10:00:00.000001Z","created_by":"t"}`
	oldIssue      = `{"id":"x-1","type":"CREATE","payload":{"title":"a","release_id":"rel-a1"},"created_at":"2026-10-01T10:00:01.000001Z","created_by":"t"}`
	oldMove       = `{"id":"x-1","type":"UPDATE","payload":{"release_id":"rel-b2","dependencies":null,"labels":null},"created_at":"2026-10-01T10:00:02.000001Z","created_by":"t","cause":"rel-a1"}`
	oldRemove     = `{"id":"x-2","type":"UPDATE","payload":{"release_id":"","children_moved":{"count":2,"release_id":"rel-b2"},"dependencies":null,"labels":null},"created_at":"2026-10-01T10:00:03.000001Z","created_by":"t"}`
	oldComplete   = `{"id":"rel-a1","type":"RELEASE_COMPLETE","payload":{"open_issues":"move_to_next","moved":1,"moved_to":"rel-b2","stats":{"counts":{"DONE":1},"leaf_ids":["x-1"]}},"created_at":"2026-10-01T10:00:04.000001Z","created_by":"t"}`
	plainComment  = `{"id":"x-1","type":"COMMENT","payload":{"id":"cmt-1","text":"we sent \"release_id\":\"rel-a1\" and \"type\":\"RELEASE_CREATE\" and rel-a1"},"created_at":"2026-10-01T10:00:05.000001Z","created_by":"t"}`
	plainTitleRel = `{"id":"x-3","type":"CREATE","payload":{"title":"rel- RELEASE_ moved_to"},"created_at":"2026-10-01T10:00:06.000001Z","created_by":"t"}`

	newCreate   = `{"id":"ms-a1","type":"MILESTONE_CREATE","payload":{"number":1,"label":"1.0"},"created_at":"2026-10-01T10:00:00.000001Z","created_by":"t"}`
	newIssue    = `{"id":"x-1","type":"CREATE","payload":{"title":"a","milestone_id":"ms-a1"},"created_at":"2026-10-01T10:00:01.000001Z","created_by":"t"}`
	newMove     = `{"id":"x-1","type":"UPDATE","payload":{"milestone_id":"ms-b2","dependencies":null,"labels":null},"created_at":"2026-10-01T10:00:02.000001Z","created_by":"t","cause":"ms-a1"}`
	newRemove   = `{"id":"x-2","type":"UPDATE","payload":{"milestone_id":"","children_moved":{"count":2,"milestone_id":"ms-b2"},"dependencies":null,"labels":null},"created_at":"2026-10-01T10:00:03.000001Z","created_by":"t"}`
	newComplete = `{"id":"ms-a1","type":"MILESTONE_COMPLETE","payload":{"open_issues":"move_to_next","moved":1,"moved_to":"ms-b2","stats":{"counts":{"DONE":1},"leaf_ids":["x-1"]}},"created_at":"2026-10-01T10:00:04.000001Z","created_by":"t"}`
)

func writeLines(t *testing.T, name string, lines ...string) {
	t.Helper()
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(XpoDir(), name), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(XpoDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMigrateMilestones_RewritesOnlyTheNamedKeys(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", oldCreate, oldIssue, oldMove, oldRemove, oldComplete, plainComment, plainTitleRel)
	before := readFile(t, "issues.db")

	res, err := MigrateMilestones()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{newCreate, newIssue, newMove, newRemove, newComplete, plainComment, plainTitleRel}, "\n") + "\n"
	if got := readFile(t, "issues.db"); got != want {
		t.Errorf("issues.db =\n%s\nwant\n%s", got, want)
	}
	if res.Lines != 5 || len(res.Files) != 1 {
		t.Errorf("result = %+v, want 5 lines in 1 file", res)
	}
	if got := readFile(t, "issues.db.bak"); got != before {
		t.Error("issues.db.bak must hold the file as it was")
	}
}

func TestMigrateMilestones_SecondRunChangesNothing(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", oldCreate, oldIssue)
	if _, err := MigrateMilestones(); err != nil {
		t.Fatal(err)
	}
	after := readFile(t, "issues.db")
	os.Remove(filepath.Join(XpoDir(), "issues.db.bak"))

	res, err := MigrateMilestones()
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines != 0 || len(res.Files) != 0 {
		t.Errorf("second run result = %+v, want nothing", res)
	}
	if readFile(t, "issues.db") != after {
		t.Error("second run changed issues.db")
	}
	if _, err := os.Stat(filepath.Join(XpoDir(), "issues.db.bak")); !os.IsNotExist(err) {
		t.Error("second run must not write a .bak")
	}
}

// A branch from before the migration merges into a migrated main:
// merge=union keeps both copies of the lines it shares with main.
func TestMigrateMilestones_UnionMergeDuplicatesAreRemoved(t *testing.T) {
	setupXpoDir(t)
	branchOnly := `{"id":"x-9","type":"CREATE","payload":{"title":"b","release_id":"rel-a1"},"created_at":"2026-10-02T10:00:00.000001Z","created_by":"t"}`
	writeLines(t, "issues.db", newCreate, newIssue, oldCreate, oldIssue, branchOnly, plainComment, plainComment)

	if !mustHasOld(t) {
		t.Fatal("old lines from the branch must be reported")
	}
	if _, err := MigrateMilestones(); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{newCreate, newIssue,
		`{"id":"x-9","type":"CREATE","payload":{"title":"b","milestone_id":"ms-a1"},"created_at":"2026-10-02T10:00:00.000001Z","created_by":"t"}`,
		plainComment, plainComment}, "\n") + "\n"
	if got := readFile(t, "issues.db"); got != want {
		t.Errorf("issues.db =\n%s\nwant\n%s", got, want)
	}
}

func TestMigrateMilestones_ArchiveAndDrafts(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", newCreate)
	writeLines(t, "archive.db", oldIssue)
	os.MkdirAll(DraftsDir(), 0755)
	draft := filepath.Join(DraftsDir(), "draft-0000000a.json")
	os.WriteFile(draft, []byte(`{"id":"draft-0000000a","payload":{"title":"the release plan","release":"@next"}}`), 0644)

	if !mustHasOld(t) {
		t.Fatal("old archive and draft data must be reported")
	}
	res, err := MigrateMilestones()
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, "archive.db"); got != newIssue+"\n" {
		t.Errorf("archive.db = %s", got)
	}
	data, _ := os.ReadFile(draft)
	if string(data) != `{"id":"draft-0000000a","payload":{"title":"the release plan","milestone":"@next"}}` {
		t.Errorf("draft = %s", data)
	}
	if _, err := os.Stat(draft + ".bak"); err != nil {
		t.Error("the draft needs a .bak")
	}
	if len(res.Files) != 2 {
		t.Errorf("files = %v, want archive.db and the draft", res.Files)
	}
	if mustHasOld(t) {
		t.Error("nothing old should be left")
	}
}

func TestMigrateMilestones_NoArchiveIsFine(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", oldIssue)
	if _, err := MigrateMilestones(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(XpoDir(), "archive.db")); !os.IsNotExist(err) {
		t.Error("the migration must not create archive.db")
	}
}

func TestMigrateMilestones_OverwritesAnOldBak(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", oldIssue)
	writeLines(t, "issues.db.bak", "stale")
	if _, err := MigrateMilestones(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, "issues.db.bak"); got != oldIssue+"\n" {
		t.Errorf("issues.db.bak = %s", got)
	}
}

func TestReadEvents_OldReleaseDataFails(t *testing.T) {
	setupXpoDir(t)
	writeLines(t, "issues.db", newCreate, oldIssue)
	if _, err := ReadEvents(); !errors.Is(err, ErrOldReleaseData) {
		t.Errorf("ReadEvents err = %v, want ErrOldReleaseData", err)
	}
	if ErrOldReleaseData.Error() != "This board has Release data from before 1.6. Run `xpo doctor --fix` to migrate it." {
		t.Errorf("message = %q", ErrOldReleaseData.Error())
	}

	writeLines(t, "issues.db", newCreate, plainComment, plainTitleRel)
	writeLines(t, "archive.db", oldIssue)
	if _, err := ReadEvents(); err != nil {
		t.Errorf("free text with old names must read: %v", err)
	}
	if _, err := ReadArchivedEvents(); !errors.Is(err, ErrOldReleaseData) {
		t.Errorf("ReadArchivedEvents err = %v, want ErrOldReleaseData", err)
	}
}

func mustHasOld(t *testing.T) bool {
	t.Helper()
	old, err := HasOldReleaseData()
	if err != nil {
		t.Fatal(err)
	}
	return old
}
