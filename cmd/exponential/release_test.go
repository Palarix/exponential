package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func releaseCLI(t *testing.T, dir string, args ...string) jsonio.ReleaseOutput {
	t.Helper()
	out, stderr, code := runXPO(t, dir, append([]string{"release"}, append(args, "--json")...)...)
	if code != 0 {
		t.Fatalf("release %v failed: %s %s", args, out, stderr)
	}
	var res jsonio.ReleaseOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unexpected output: %s", out)
	}
	return res
}

func TestReleaseCLI_Lifecycle(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)

	list := releaseCLI(t, dir, "list")
	if len(list.Releases) != 2 || !list.Releases[0].Placeholder {
		t.Fatalf("list = %+v", list.Releases)
	}

	created := releaseCLI(t, dir, "create", "--label", "1.5", "--goal", "Ship it")
	if created.Release == nil || created.Release.Ref != "#1" {
		t.Fatalf("create = %+v", created)
	}

	issue := addIssueCLI(t, dir, "A")
	if out, _, code := runXPO(t, dir, "update", issue, "--release", "1.5"); code != 0 {
		t.Fatalf("update --release failed: %s", out)
	}
	out, _, code := runXPO(t, dir, "list", "--release", "@current", "--json")
	if code != 0 || !strings.Contains(out, issue) || !strings.Contains(out, `"release": "1.5"`) {
		t.Fatalf("list --release @current: %s", out)
	}

	child, _, code := runXPO(t, dir, "add", "Child", "--parent", issue, "--release", "@next", "--force")
	if code != 0 || !strings.Contains(child, "Release: #2") {
		t.Errorf("add --release @next: %s", child)
	}

	releaseCLI(t, dir, "update", "#2", "--label", "1.6")
	releaseCLI(t, dir, "reorder", "1.6", "--before", "1.5")
	list = releaseCLI(t, dir, "list")
	if list.Releases[0].Name != "1.6" {
		t.Errorf("after reorder current = %+v", list.Releases[0])
	}

	done := releaseCLI(t, dir, "complete", "1.5", "--remove")
	if done.Release.State != "RELEASED" {
		t.Errorf("complete = %+v", done)
	}
	if out, _, code := runXPO(t, dir, "update", issue, "--release", "1.5"); code == 0 {
		t.Errorf("assigning to a RELEASED release must fail: %s", out)
	}
	if _, _, code := runXPO(t, dir, "release", "delete", "1.5"); code == 0 {
		t.Error("deleting a RELEASED release must fail")
	}
	releaseCLI(t, dir, "delete", "1.6")

	text, _, code := runXPO(t, dir, "release")
	if code != 0 || !strings.Contains(text, "Current") || !strings.Contains(text, "1.5") {
		t.Errorf("text list: %s", text)
	}
}
