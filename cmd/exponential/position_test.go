package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/jsonio"
)

func listOrderCLI(t *testing.T, dir string) string {
	t.Helper()
	out, _, code := runXPO(t, dir, "list", "--json")
	if code != 0 {
		t.Fatalf("list failed: %s", out)
	}
	var res jsonio.ListOutput
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unexpected list output: %s", out)
	}
	ids := make([]string, 0, len(res.Issues))
	for _, i := range res.Issues {
		ids = append(ids, i.ID)
	}
	return strings.Join(ids, ",")
}

func TestPositionCLI_UpdateFlagsAndJSON(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")
	c := addIssueCLI(t, dir, "C")

	if out, _, code := runXPO(t, dir, "update", c, "--before", a); code != 0 {
		t.Fatalf("update --before failed: %s", out)
	}
	if got, want := listOrderCLI(t, dir), strings.Join([]string{c, a, b}, ","); got != want {
		t.Errorf("after --before: got %s want %s", got, want)
	}

	if out, code := runXPOStdin(t, dir, `{"after":"`+b+`"}`, "update", c, "--json"); code != 0 {
		t.Fatalf("update --json after failed: %s", out)
	}
	if got, want := listOrderCLI(t, dir), strings.Join([]string{a, b, c}, ","); got != want {
		t.Errorf("after json after: got %s want %s", got, want)
	}

	out, _, code := runXPO(t, dir, "update", c, "--before", a, "--after", b)
	if code == 0 || !strings.Contains(out, "mutually exclusive") {
		t.Errorf("both flags: code=%d out=%q", code, out)
	}
}

func TestPositionCLI_Add(t *testing.T) {
	dir := setupProject(t)
	defer os.RemoveAll(dir)
	a := addIssueCLI(t, dir, "A")
	b := addIssueCLI(t, dir, "B")

	out, code := runXPOStdin(t, dir, `{"title":"X","after":"`+a+`"}`, "add", "--json")
	if code != 0 {
		t.Fatalf("add --json after failed: %s", out)
	}
	var x jsonio.AddOutput
	if err := json.Unmarshal([]byte(out), &x); err != nil {
		t.Fatalf("unexpected add output: %s", out)
	}

	out, _, code = runXPO(t, dir, "add", "Y", "--before", a, "--force")
	if code != 0 {
		t.Fatalf("add --before failed: %s", out)
	}
	y := strings.TrimSuffix(strings.Fields(out)[1], ":")

	if got, want := listOrderCLI(t, dir), strings.Join([]string{y, a, x.ID, b}, ","); got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
