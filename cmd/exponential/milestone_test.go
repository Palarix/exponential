package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func milestoneCLI(t *testing.T, dir string, args ...string) jsonio.MilestoneOutput {
	t.Helper()
	out, stderr, code := runXPO(t, dir, append([]string{"milestone"}, append(args, "--json")...)...)
	if code != 0 {
		t.Fatalf("milestone %v failed: %s %s", args, out, stderr)
	}
	var res jsonio.MilestoneOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unexpected output: %s", out)
	}
	return res
}

func TestMilestoneCLI_Lifecycle(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	list := milestoneCLI(t, dir, "list")
	if len(list.Milestones) != 2 || !list.Milestones[0].Placeholder {
		t.Fatalf("list = %+v", list.Milestones)
	}

	created := milestoneCLI(t, dir, "create", "--label", "1.5", "--goal", "Ship it")
	if created.Milestone == nil || created.Milestone.Ref != "#1" {
		t.Fatalf("create = %+v", created)
	}

	issue := addIssueCLI(t, dir, "A")
	if out, _, code := runXPO(t, dir, "update", issue, "--milestone", "1.5"); code != 0 {
		t.Fatalf("update --milestone failed: %s", out)
	}
	out, _, code := runXPO(t, dir, "list", "--milestone", "@current", "--json")
	if code != 0 || !strings.Contains(out, issue) || !strings.Contains(out, `"milestone": "1.5"`) {
		t.Fatalf("list --milestone @current: %s", out)
	}

	child, _, code := runXPO(t, dir, "add", "Child", "--parent", issue, "--milestone", "@next", "--force")
	if code != 0 || !strings.Contains(child, "Milestone: #2") {
		t.Errorf("add --milestone @next: %s", child)
	}

	milestoneCLI(t, dir, "update", "#2", "--label", "1.6")
	milestoneCLI(t, dir, "reorder", "1.6", "--before", "1.5")
	list = milestoneCLI(t, dir, "list")
	if list.Milestones[0].Name != "1.6" {
		t.Errorf("after reorder current = %+v", list.Milestones[0])
	}

	done := milestoneCLI(t, dir, "complete", "1.5", "--remove")
	if done.Milestone.State != "COMPLETED" {
		t.Errorf("complete = %+v", done)
	}
	if out, _, code := runXPO(t, dir, "update", issue, "--milestone", "1.5"); code == 0 {
		t.Errorf("assigning to a COMPLETED milestone must fail: %s", out)
	}
	if _, _, code := runXPO(t, dir, "milestone", "delete", "1.5"); code == 0 {
		t.Error("deleting a COMPLETED milestone must fail")
	}
	milestoneCLI(t, dir, "delete", "1.6")

	text, _, code := runXPO(t, dir, "milestone")
	if code != 0 || !strings.Contains(text, "Current") || !strings.Contains(text, "1.5") {
		t.Errorf("text list: %s", text)
	}
}

// A board with Release data from before 1.6: reads refuse it, doctor
// reports it, and doctor --fix migrates it once.
func TestDoctor_MigratesOldReleaseData(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	milestoneCLI(t, dir, "create", "--label", "1.5")
	issue := addIssueCLI(t, dir, "A")
	if out, _, code := runXPO(t, dir, "update", issue, "--milestone", "1.5"); code != 0 {
		t.Fatalf("update --milestone failed: %s", out)
	}

	db := filepath.Join(dir, ".xpo", "issues.db")
	data, _ := os.ReadFile(db)
	old := strings.NewReplacer(`"type":"MILESTONE_`, `"type":"RELEASE_`, `"id":"ms-`, `"id":"rel-`,
		`"milestone_id":"ms-`, `"release_id":"rel-`).Replace(string(data))
	os.WriteFile(db, []byte(old), 0644)

	out, stderr, code := runXPO(t, dir, "list")
	if code == 0 || !strings.Contains(out+stderr, "This board has Release data from before 1.6. Run `xpo doctor --fix` to migrate it.") {
		t.Fatalf("list on old data: code %d, %s %s", code, out, stderr)
	}

	out, _, _ = runXPO(t, dir, "doctor")
	if !strings.Contains(out, "Milestone data uses the old Release names") {
		t.Errorf("doctor does not report the old names:\n%s", out)
	}

	out, _, _ = runXPO(t, dir, "doctor", "--fix")
	if !strings.Contains(out, "Migrated Release data to Milestones") {
		t.Errorf("doctor --fix does not report the migration:\n%s", out)
	}
	if _, err := os.Stat(db + ".bak"); err != nil {
		t.Error("doctor --fix must write issues.db.bak")
	}
	migrated, _ := os.ReadFile(db)
	if string(migrated) != string(data) {
		t.Errorf("migration must restore the new names byte for byte")
	}

	out, _, code = runXPO(t, dir, "list", "--milestone", "1.5", "--json")
	if code != 0 || !strings.Contains(out, issue) {
		t.Fatalf("list after the migration: %s", out)
	}

	runXPO(t, dir, "doctor", "--fix")
	if again, _ := os.ReadFile(db); string(again) != string(migrated) {
		t.Error("a second doctor --fix must change nothing")
	}
	out, _, _ = runXPO(t, dir, "doctor")
	if !strings.Contains(out, "Milestone data uses the new names") {
		t.Errorf("doctor after the migration:\n%s", out)
	}
}
