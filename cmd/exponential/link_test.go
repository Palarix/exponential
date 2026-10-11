package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func TestLinkCLI_SameErrorsAcrossPositionalAndJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	if out, _, code := runXPO(t, dir, "link", a, b, "-t", "blocks"); code != 0 {
		t.Fatalf("link failed: %s", out)
	}

	cases := []struct{ name, source, target, kind, want string }{
		{"already exists", a, b, "blocks", "link blocks " + b + " already exists on " + a},
		{"self", a, a, "blocks", "cannot link an issue to itself"},
		{"conflict", a, b, "relates_to", a + " is already linked to " + b + " (blocks); set overwrite to replace it with relates_to"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _, code := runXPO(t, dir, "link", tc.source, tc.target, "-t", tc.kind)
			if code == 0 || !strings.Contains(out, tc.want) {
				t.Errorf("positional: code=%d out=%q, want %q", code, out, tc.want)
			}
			payload := `{"source":"` + tc.source + `","target":"` + tc.target + `","type":"` + tc.kind + `"}`
			out, code = runXPOStdin(t, dir, payload, "link", "--json")
			assertJSONError(t, out, code, tc.want)
		})
	}
}

func TestLinkCLI_OverwritePositional(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	if out, _, code := runXPO(t, dir, "link", a, b, "-t", "blocks"); code != 0 {
		t.Fatalf("link failed: %s", out)
	}

	out, _, code := runXPO(t, dir, "link", b, a, "-t", "relates_to", "--overwrite")
	if code != 0 {
		t.Fatalf("link --overwrite failed: %s", out)
	}
	if !strings.Contains(out, "(replaced blocked_by)") {
		t.Errorf("unexpected output: %s", out)
	}
	if deps := showDepsCLI(t, dir, a); len(deps) != 1 || !strings.Contains(string(deps[0]), "relates_to") {
		t.Errorf("A deps = %s, want only relates_to", deps)
	}
}

func TestLinkCLI_OverwriteJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	if out, _, code := runXPO(t, dir, "link", a, b, "-t", "depends_on"); code != 0 {
		t.Fatalf("link failed: %s", out)
	}

	payload := `{"source":"` + a + `","target":"` + b + `","type":"relates_to","overwrite":true}`
	out, code := runXPOStdin(t, dir, payload, "link", "--json")
	if code != 0 {
		t.Fatalf("link --json overwrite failed: %s", out)
	}
	var res jsonio.LinkOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.Kind != "relates_to" ||
		len(res.Replaced) != 1 || res.Replaced[0].Kind != "depends_on" {
		t.Errorf("unexpected link output: %s", out)
	}
}
