package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Bug xpo-bbfcf9: an explicit null on assignee, cycle_id, priority or
// story_points decoded to nil, the same as an omitted field, so agents could
// not clear them. Goes through the protocol so the raw arguments reach the
// handler.
func TestEndToEndUpdateNullClearsScalarFields(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".xpo", "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{
		Prefix:           "e2e-",
		User:             "x",
		EstimationSystem: "fibonacci",
		Version:          2,
		Cycles:           config.CycleConfig{Enabled: true, Duration: "2w", StartDay: "monday"},
	}
	cycleID := cfg.Cycles.CurrentCycle().ID

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srvSess, _ := newStdioServer(cfg).Connect(ctx, st, nil)
	defer srvSess.Close()
	cliSess, _ := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil).Connect(ctx, ct, nil)
	defer cliSess.Close()

	call := func(args map[string]interface{}) *mcp.CallToolResult {
		t.Helper()
		raw, _ := json.Marshal(args)
		res, err := cliSess.CallTool(ctx, &mcp.CallToolParams{Name: "update", Arguments: json.RawMessage(raw)})
		if err != nil {
			t.Fatalf("CallTool update: %v", err)
		}
		return res
	}
	addRaw, _ := json.Marshal(map[string]interface{}{"title": "Issue", "description": "Body", "labels": []string{"bug"}})
	addRes, err := cliSess.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: json.RawMessage(addRaw)})
	if err != nil || addRes.IsError {
		t.Fatalf("add failed: %v %s", err, textOf(addRes))
	}
	var added jsonio.AddOutput
	if err := remarshal(addRes.StructuredContent, &added); err != nil {
		t.Fatal(err)
	}
	id := added.ID

	issue := func() *model.Issue {
		t.Helper()
		i, err := exponential.NewClient(cfg).GetIssue(id)
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	set := map[string]interface{}{
		"id":           id,
		"assignee":     "Alice <alice@example.com>",
		"cycle_id":     cycleID,
		"priority":     2,
		"story_points": 3,
	}
	type fieldCase struct {
		field   string
		cleared func(*model.Issue) bool
	}
	fields := []fieldCase{
		{"assignee", func(i *model.Issue) bool { return i.Assignee == "" }},
		{"cycle_id", func(i *model.Issue) bool { return i.CycleID == "" }},
		{"priority", func(i *model.Issue) bool { return i.Priority == 0 }},
		{"story_points", func(i *model.Issue) bool { return i.Estimate == 0 }},
	}

	for _, f := range fields {
		for _, withOther := range []bool{false, true} {
			name := f.field
			if withOther {
				name += " with another field"
			}
			t.Run(name, func(t *testing.T) {
				if res := call(set); res.IsError {
					t.Fatalf("set fields: %s", textOf(res))
				}
				if f.cleared(issue()) {
					t.Fatalf("%s not set before clear", f.field)
				}
				args := map[string]interface{}{"id": id, f.field: nil}
				if withOther {
					args["title"] = "Renamed " + f.field
				}
				if res := call(args); res.IsError {
					t.Fatalf("clear %s: %s", f.field, textOf(res))
				}
				got := issue()
				if !f.cleared(got) {
					t.Errorf("%s not cleared: %+v", f.field, got)
				}
				if withOther && got.Title != "Renamed "+f.field {
					t.Errorf("title: got %q", got.Title)
				}
			})
		}
	}

	t.Run("omitted fields are unchanged", func(t *testing.T) {
		call(set)
		if res := call(map[string]interface{}{"id": id, "title": "Only title"}); res.IsError {
			t.Fatalf("update title: %s", textOf(res))
		}
		got := issue()
		if got.Assignee == "" || got.CycleID == "" || got.Priority == 0 || got.Estimate == 0 {
			t.Errorf("omitted fields changed: %+v", got)
		}
	})

	// description, labels and links keep null as "unchanged": a client that
	// sends null for every unused field must not erase content.
	t.Run("null leaves description labels and links unchanged", func(t *testing.T) {
		other, _ := json.Marshal(map[string]interface{}{"title": "Other"})
		otherRes, _ := cliSess.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: json.RawMessage(other)})
		var o jsonio.AddOutput
		remarshal(otherRes.StructuredContent, &o)
		if res := call(map[string]interface{}{"id": id, "links": []map[string]string{{"target": o.ID, "type": "relates_to"}}}); res.IsError {
			t.Fatalf("set links: %s", textOf(res))
		}
		args := map[string]interface{}{"id": id, "title": "Nulls", "description": nil, "labels": nil, "links": nil}
		if res := call(args); res.IsError {
			t.Fatalf("update with nulls: %s", textOf(res))
		}
		got := issue()
		if got.Description != "Body" {
			t.Errorf("description: got %q want Body", got.Description)
		}
		if len(got.Labels) != 1 || got.Labels[0] != "bug" {
			t.Errorf("labels: got %v want [bug]", got.Labels)
		}
		if len(got.Links) != 1 {
			t.Errorf("links: got %v want one link", got.Links)
		}
	})
}

// Bug xpo-bbfcf9: the nullable fields did not say how to clear them.
func TestUpdateSchemaDocumentsClearingScalarFields(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".xpo", "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{Prefix: "e2e-", User: "x", EstimationSystem: "fibonacci", Version: 2}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srvSess, _ := newStdioServer(cfg).Connect(ctx, st, nil)
	defer srvSess.Close()
	cliSess, _ := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil).Connect(ctx, ct, nil)
	defer cliSess.Close()

	res, err := cliSess.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "update" {
			continue
		}
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]struct {
				Description string `json:"description"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"assignee", "cycle_id", "priority", "story_points"} {
			desc := schema.Properties[field].Description
			if !containsAll(desc, "null", "clear") && !containsAll(desc, "null", "unassign") && !containsAll(desc, "null", "remove") {
				t.Errorf("%s description should explain clearing with null, got %q", field, desc)
			}
		}
		return
	}
	t.Fatal("update tool not registered")
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
