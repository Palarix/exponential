package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func runXPOStdin(t *testing.T, projectDir, stdin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Dir = projectDir
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok {
		return out.String(), e.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run xpo: %v", err)
	}
	return out.String(), 0
}

func addIssueCLI(t *testing.T, dir, title string) string {
	t.Helper()
	out, code := runXPOStdin(t, dir, `{"title":"`+title+`"}`, "add", "--json")
	if code != 0 {
		t.Fatalf("add failed: %s", out)
	}
	var res struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.ID == "" {
		t.Fatalf("unexpected add output: %s", out)
	}
	return res.ID
}

func showDepsCLI(t *testing.T, dir, id string) []json.RawMessage {
	t.Helper()
	out, _, code := runXPO(t, dir, "show", id, "--json")
	if code != 0 {
		t.Fatalf("show failed: %s", out)
	}
	var res struct {
		Dependencies []json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unexpected show output: %s", out)
	}
	return res.Dependencies
}

func TestUnlinkCLI_Positional(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	if out, _, code := runXPO(t, dir, "link", a, b, "-t", "blocks"); code != 0 {
		t.Fatalf("link failed: %s", out)
	}

	out, _, code := runXPO(t, dir, "unlink", b, a, "-t", "blocked_by")
	if code != 0 {
		t.Fatalf("unlink failed: %s", out)
	}
	if !strings.Contains(out, "Unlinked") {
		t.Errorf("unexpected output: %s", out)
	}
	if deps := showDepsCLI(t, dir, a); len(deps) != 0 {
		t.Errorf("A deps = %s, want none", deps)
	}
}

func TestUnlinkCLI_JSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	if out, _, code := runXPO(t, dir, "link", a, b, "-t", "depends_on"); code != 0 {
		t.Fatalf("link failed: %s", out)
	}

	payload := `{"source":"` + a + `","target":"` + b + `","type":"depends_on"}`
	out, code := runXPOStdin(t, dir, payload, "unlink", "--json")
	if code != 0 {
		t.Fatalf("unlink --json failed: %s", out)
	}
	var res jsonio.LinkOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.Source != a || res.Target != b || res.Kind != "depends_on" {
		t.Errorf("unexpected unlink output: %s", out)
	}

	out, code = runXPOStdin(t, dir, payload, "unlink", "--json")
	assertJSONError(t, out, code, "no link depends_on")
}
