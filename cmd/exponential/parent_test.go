package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func parentCLI(t *testing.T, dir, id string) string {
	t.Helper()
	out, _, code := runXPO(t, dir, "show", id, "--json")
	if code != 0 {
		t.Fatalf("show failed: %s", out)
	}
	var res jsonio.ShowOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unexpected show output: %s", out)
	}
	return res.ParentID
}

// Bug xpo-328824: `update --json` ignored an explicit `"parent": null`.
func TestUpdateJSONNullParentClears(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	p := addIssueCLI(t, dir, "P")
	c := addIssueCLI(t, dir, "C")

	if out, code := runXPOStdin(t, dir, `{"parent":"`+p+`"}`, "update", c, "--json"); code != 0 {
		t.Fatalf("set parent failed: %s", out)
	}
	if got := parentCLI(t, dir, c); got != p {
		t.Fatalf("parent: got %q want %q", got, p)
	}
	if out, code := runXPOStdin(t, dir, `{"parent":null}`, "update", c, "--json"); code != 0 {
		t.Fatalf("clear parent with null failed: %s", out)
	}
	if got := parentCLI(t, dir, c); got != "" {
		t.Errorf("parent after null: got %q want empty", got)
	}
}
