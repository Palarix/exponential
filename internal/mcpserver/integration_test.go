package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestEndToEndAddViaMCP wires a real MCP client to a real MCP server
// (both in-memory) and exercises add → show through the
// protocol. This catches schema-generation or JSON-marshalling bugs
// that the direct handler unit tests can't.
func TestEndToEndAddViaMCP(t *testing.T) {
	tmpDir := t.TempDir()
	xpoDir := filepath.Join(tmpDir, ".xpo")
	if err := os.MkdirAll(xpoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xpoDir, "issues.db"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{
		Prefix:           "e2e-",
		User:             "E2E User <e2e@test>",
		EstimationSystem: "fibonacci",
		CountUnestimated: true,
		Version:          2,
	}

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	srv := mcp.NewServer(&mcp.Implementation{Name: "issue-test", Version: "v0"}, nil)
	newToolset(cfg).register(srv)
	serverSession, err := srv.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	// Call add via JSON arguments — exercises the schema-derived
	// unmarshaling path.
	addArgs, _ := json.Marshal(map[string]interface{}{
		"title":        "Created via MCP",
		"description":  "## Body\n\nWith `code` and \"quotes\".",
		"status":       "PLANNED",
		"labels":       []string{"feature"},
		"story_points": 5,
	})
	addRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add",
		Arguments: json.RawMessage(addArgs),
	})
	if err != nil {
		t.Fatalf("CallTool add: %v", err)
	}
	if addRes.IsError {
		t.Fatalf("add returned error: %s", textOf(addRes))
	}
	var addOutput jsonio.AddOutput
	if err := remarshal(addRes.StructuredContent, &addOutput); err != nil {
		t.Fatalf("unmarshal AddOutput: %v", err)
	}
	if addOutput.ID == "" {
		t.Fatalf("expected non-empty issue ID, got: %+v", addOutput)
	}
	if !strings.HasPrefix(addOutput.ID, "e2e-") {
		t.Errorf("expected prefix e2e-, got %q", addOutput.ID)
	}
	if addOutput.Status != "PLANNED" {
		t.Errorf("Status: got %q want PLANNED", addOutput.Status)
	}

	// Round-trip the issue back via show.
	showArgs, _ := json.Marshal(map[string]interface{}{"id": addOutput.ID})
	showRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "show",
		Arguments: json.RawMessage(showArgs),
	})
	if err != nil {
		t.Fatalf("CallTool show: %v", err)
	}
	if showRes.IsError {
		t.Fatalf("show returned error: %s", textOf(showRes))
	}
	var showOutput jsonio.ShowOutput
	if err := remarshal(showRes.StructuredContent, &showOutput); err != nil {
		t.Fatalf("unmarshal ShowOutput: %v", err)
	}
	wantDesc := "## Body\n\nWith `code` and \"quotes\"."
	if showOutput.Description != wantDesc {
		t.Errorf("description round-trip mismatch:\n got: %q\nwant: %q", showOutput.Description, wantDesc)
	}
}

// TestEndToEndListsAllTools verifies tools/list returns our full registered
// surface. If a registration call is dropped during refactoring this catches it.
func TestEndToEndListsAllTools(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".xpo", "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{Prefix: "e2e-", User: "x", EstimationSystem: "fibonacci", Version: 2}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srv := newStdioServer(cfg)
	srvSess, _ := srv.Connect(ctx, st, nil)
	defer srvSess.Close()
	cli := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil)
	cliSess, _ := cli.Connect(ctx, ct, nil)
	defer cliSess.Close()

	res, err := cliSess.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want := map[string]bool{
		"list": false, "show": false, "history": false,
		"add": false, "update": false, "comment": false, "link": false,
		"start": false, "merge": false, "rationale": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func textOf(r *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// remarshal round-trips a value through JSON so we can decode the SDK's
// any-typed StructuredContent into our typed struct.
func remarshal(src interface{}, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// Bug: the merge description did not say a committed worktree is required,
// so agents only learned it from the failure.
func TestMergeToolDescriptionRequiresCommit(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".xpo", "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{Prefix: "e2e-", User: "x", EstimationSystem: "fibonacci", Version: 2}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srv := newStdioServer(cfg)
	srvSess, _ := srv.Connect(ctx, st, nil)
	defer srvSess.Close()
	cli := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil)
	cliSess, _ := cli.Connect(ctx, ct, nil)
	defer cliSess.Close()

	res, err := cliSess.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "merge" {
			continue
		}
		for _, want := range []string{"must be committed", "squash"} {
			if !strings.Contains(tool.Description, want) {
				t.Errorf("merge description missing %q: %s", want, tool.Description)
			}
		}
		return
	}
	t.Fatal("merge tool not registered")
}

// Bug xpo-328824: agents could not clear a parent through `update`. `null`
// failed with "no fields set" and the literal string `""` was looked up as an
// issue ID. Goes through the protocol so the raw arguments reach the handler.
func TestEndToEndUpdateClearsParent(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".xpo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".xpo", "issues.db"), []byte{}, 0644)
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &config.Config{Prefix: "e2e-", User: "x", EstimationSystem: "fibonacci", Version: 2}
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srv := newStdioServer(cfg)
	srvSess, _ := srv.Connect(ctx, st, nil)
	defer srvSess.Close()
	cli := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil)
	cliSess, _ := cli.Connect(ctx, ct, nil)
	defer cliSess.Close()

	call := func(name string, args map[string]interface{}) *mcp.CallToolResult {
		t.Helper()
		raw, _ := json.Marshal(args)
		res, err := cliSess.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: json.RawMessage(raw)})
		if err != nil {
			t.Fatalf("CallTool %s: %v", name, err)
		}
		return res
	}
	addIssue := func(title string) string {
		t.Helper()
		res := call("add", map[string]interface{}{"title": title})
		var out jsonio.AddOutput
		if err := remarshal(res.StructuredContent, &out); err != nil {
			t.Fatal(err)
		}
		return out.ID
	}
	parentOf := func(id string) string {
		t.Helper()
		res := call("show", map[string]interface{}{"id": id})
		var out jsonio.ShowOutput
		if err := remarshal(res.StructuredContent, &out); err != nil {
			t.Fatal(err)
		}
		return out.ParentID
	}

	parent := addIssue("P")
	child := addIssue("C")

	cases := []struct {
		name  string
		clear map[string]interface{}
	}{
		{"null", map[string]interface{}{"parent": nil}},
		{"empty string", map[string]interface{}{"parent": ""}},
		{"quoted empty string", map[string]interface{}{"parent": `""`}},
		{"null with another field", map[string]interface{}{"parent": nil, "title": "C2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if res := call("update", map[string]interface{}{"id": child, "parent": parent}); res.IsError {
				t.Fatalf("set parent: %s", textOf(res))
			}
			if got := parentOf(child); got != parent {
				t.Fatalf("parent before clear: got %q want %q", got, parent)
			}
			args := map[string]interface{}{"id": child}
			for k, v := range tc.clear {
				args[k] = v
			}
			if res := call("update", args); res.IsError {
				t.Fatalf("clear parent: %s", textOf(res))
			}
			if got := parentOf(child); got != "" {
				t.Errorf("parent after clear: got %q want empty", got)
			}
		})
	}

	t.Run("omitted parent is unchanged", func(t *testing.T) {
		call("update", map[string]interface{}{"id": child, "parent": parent})
		if res := call("update", map[string]interface{}{"id": child, "title": "C3"}); res.IsError {
			t.Fatalf("update title: %s", textOf(res))
		}
		if got := parentOf(child); got != parent {
			t.Errorf("parent: got %q want %q", got, parent)
		}
	})
}

// Bug xpo-328824: the `parent` field had no description, so agents had to
// guess how to clear it.
func TestUpdateParentSchemaDocumentsClearing(t *testing.T) {
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
		desc := schema.Properties["parent"].Description
		if !strings.Contains(desc, "null") || !strings.Contains(desc, `""`) {
			t.Errorf("parent description should explain clearing with null or \"\", got %q", desc)
		}
		return
	}
	t.Fatal("update tool not registered")
}
