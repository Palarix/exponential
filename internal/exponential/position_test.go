package exponential

import (
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/model"
)

func keyed(id, key string) *model.Issue {
	return &model.Issue{ID: id, SortOrder: key}
}

func TestKeyBeside_BelowPicksGlobalNeighbor(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0"), keyed("b", "a2"), keyed("c", "a4")}
	key, err := keyBeside(issues, "", "a", true)
	if err != nil {
		t.Fatal(err)
	}
	if !(key > "a0" && key < "a2") {
		t.Errorf("key %q not between a0 and a2", key)
	}
}

func TestKeyBeside_AbovePicksGlobalNeighbor(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0"), keyed("b", "a2"), keyed("c", "a4")}
	key, err := keyBeside(issues, "", "c", false)
	if err != nil {
		t.Fatal(err)
	}
	if !(key > "a2" && key < "a4") {
		t.Errorf("key %q not between a2 and a4", key)
	}
}

func TestKeyBeside_OpenEnds(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0"), keyed("b", "a2")}
	below, err := keyBeside(issues, "", "b", true)
	if err != nil {
		t.Fatal(err)
	}
	if below <= "a2" {
		t.Errorf("below-last key %q not after a2", below)
	}
	above, err := keyBeside(issues, "", "a", false)
	if err != nil {
		t.Fatal(err)
	}
	if above >= "a0" {
		t.Errorf("above-first key %q not before a0", above)
	}
}

func TestKeyBeside_SkipsSelf(t *testing.T) {
	// x sits right after a; moving x below b must ignore x's own key.
	issues := []*model.Issue{keyed("a", "a0"), keyed("x", "a1"), keyed("b", "a2"), keyed("c", "a4")}
	key, err := keyBeside(issues, "x", "b", true)
	if err != nil {
		t.Fatal(err)
	}
	if !(key > "a2" && key < "a4") {
		t.Errorf("key %q not between a2 and a4", key)
	}
}

func TestKeyBeside_AlreadyInPositionKeepsKey(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0"), keyed("x", "a1"), keyed("b", "a2")}
	below, err := keyBeside(issues, "x", "a", true)
	if err != nil {
		t.Fatal(err)
	}
	if below != "a1" {
		t.Errorf("below: got %q, want unchanged a1", below)
	}
	above, err := keyBeside(issues, "x", "b", false)
	if err != nil {
		t.Fatal(err)
	}
	if above != "a1" {
		t.Errorf("above: got %q, want unchanged a1", above)
	}
}

func TestKeyBeside_DuplicateTargetKeySkipped(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0"), keyed("dup", "a0"), keyed("b", "a2")}
	key, err := keyBeside(issues, "", "a", true)
	if err != nil {
		t.Fatal(err)
	}
	if !(key > "a0" && key < "a2") {
		t.Errorf("key %q not between a0 and a2", key)
	}
}

func TestKeyBeside_TargetMissing(t *testing.T) {
	issues := []*model.Issue{keyed("a", "a0")}
	if _, err := keyBeside(issues, "", "zzz", true); err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestClient_PositionKey_Errors(t *testing.T) {
	c := setupClient(t)
	a, _ := c.AddIssue(model.CreatePayload{Title: "a"})

	if _, err := c.PositionKey(a.ID, a.ID, ""); err == nil || !strings.Contains(err.Error(), "itself") {
		t.Errorf("self target: got %v", err)
	}
	if _, err := c.PositionKey(a.ID, "", "test-nope"); err == nil {
		t.Error("expected error for missing target")
	}
	if _, err := c.PositionKey(a.ID, a.ID, a.ID); err == nil {
		t.Error("expected error for before and after together")
	}
	pos, err := c.PositionKey(a.ID, "", "")
	if err != nil || pos != nil {
		t.Errorf("no position: got %v, %v", pos, err)
	}
}

func TestClient_PositionKey_UpdateAcrossStatuses(t *testing.T) {
	c := setupClient(t)
	a, _ := c.AddIssue(model.CreatePayload{Title: "a", Status: "PLANNED"})
	b, _ := c.AddIssue(model.CreatePayload{Title: "b", Status: "PLANNED"})
	x, _ := c.AddIssue(model.CreatePayload{Title: "x"})

	pos, err := c.PositionKey(x.ID, "", a.ID[len(a.ID)-4:])
	if err != nil {
		t.Fatal(err)
	}
	if pos.Target != a.ID {
		t.Errorf("target: got %q want %q", pos.Target, a.ID)
	}
	status := "PLANNED"
	if _, err := c.UpdateIssue(x.ID, model.UpdatePayload{Status: &status, SortOrder: &pos.Key}, "update"); err != nil {
		t.Fatal(err)
	}

	planned, _ := c.ListIssues(FilterOptions{Statuses: []string{"PLANNED"}})
	var got []string
	for _, i := range planned {
		got = append(got, i.ID)
	}
	want := []string{a.ID, x.ID, b.ID}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order: got %v want %v", got, want)
	}
}

func TestPosition_Unchanged(t *testing.T) {
	p := &Position{Target: "t-1", Above: true}
	if got := p.Unchanged("t-2"); got != "t-2 is already before t-1" {
		t.Errorf("got %q", got)
	}
	p.Above = false
	if got := p.Unchanged("t-2"); got != "t-2 is already after t-1" {
		t.Errorf("got %q", got)
	}
}
