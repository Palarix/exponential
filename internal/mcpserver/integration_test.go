package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestEndToEndAddViaMCP wires a real MCP client to a real MCP server
// (both in-memory) and exercises add → show through the
// protocol. This catches schema-generation or JSON-marshalling bugs
// that the direct handler unit tests can't.
func TestEndToEndAddViaMCP(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	exec.Command("git", "init", "-b", "main", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(tmpDir, "init.txt"), []byte("init"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "init").Run()

	xpoDir := filepath.Join(tmpDir, ".xpo")
	os.MkdirAll(xpoDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	storage.ResetHubRoot()
	storage.ResetRefStore()
	if err := storage.InitRefStore(); err != nil {
		t.Fatalf("InitRefStore: %v", err)
	}
	defer func() {
		os.Chdir(origDir)
		storage.ResetHubRoot()
		storage.ResetRefStore()
	}()

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
	var addOutput addOut
	if err := remarshal(addRes.StructuredContent, &addOutput); err != nil {
		t.Fatalf("unmarshal addOut: %v", err)
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
	var showOutput showOut
	if err := remarshal(showRes.StructuredContent, &showOutput); err != nil {
		t.Fatalf("unmarshal showOut: %v", err)
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
	srv := mcp.NewServer(&mcp.Implementation{Name: "xpo", Version: "v0"}, nil)
	newToolset(cfg).register(srv)
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
