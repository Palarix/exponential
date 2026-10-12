package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var gitToolNames = []string{"git_status", "git_diff", "git_log", "git_show", "code_history"}

func listToolNames(t *testing.T, srv *mcp.Server) []string {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func TestGitTools_OnlyOnStdioServer(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()
	cfg := &config.Config{Prefix: "test", User: "U <u@test>", Version: 2}

	stdio := listToolNames(t, newStdioServer(cfg))
	for _, name := range gitToolNames {
		if !slices.Contains(stdio, name) {
			t.Errorf("stdio server is missing %s", name)
		}
	}
	if !slices.Contains(stdio, "show") {
		t.Error("stdio server is missing the issue tools")
	}

	httpSrv := mcp.NewServer(&mcp.Implementation{Name: "xpo", Version: "v0"}, nil)
	RegisterTools(httpSrv, cfg, nil, nil)
	served := listToolNames(t, httpSrv)
	for _, name := range gitToolNames {
		if slices.Contains(served, name) {
			t.Errorf("xpo serve's /mcp offers %s; the server's checkout is not the user's", name)
		}
	}
}

// startMCPIssue seeds a PLANNED issue, starts it and commits one file in its
// worktree.
func startMCPIssue(t *testing.T, ts *toolset, id string) string {
	t.Helper()
	seedMCPIssue(t, id, "Git tools", "PLANNED")
	_, started, err := ts.start(context.Background(), nil, jsonio.StartToolInput{ID: id})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	wt := started.WorktreePath
	os.WriteFile(filepath.Join(wt, "feature.txt"), []byte("feature line\n"), 0644)
	mcpRunGit(t, wt, "add", ".")
	mcpRunGit(t, wt, "commit", "-m", id+": add feature")
	return wt
}

func TestGitDiffTool_EndToEnd(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	startMCPIssue(t, ts, "test-aa0001")

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srv := mcp.NewServer(&mcp.Implementation{Name: "xpo", Version: "v0"}, nil)
	ts.register(srv)
	ts.registerGitTools(srv)
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "v0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	args, _ := json.Marshal(map[string]any{"id": "aa0001", "path": "feature.txt"})
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "git_diff", Arguments: json.RawMessage(args)})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("git_diff returned an error: %s", textOf(res))
	}
	var out exponential.GitDiffResult
	if err := remarshal(res.StructuredContent, &out); err != nil {
		t.Fatal(err)
	}
	if out.IssueID != "test-aa0001" || !strings.Contains(out.Patch, "+feature line") {
		t.Errorf("git_diff = %+v", out)
	}
}

func TestGitStatusTool(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	startMCPIssue(t, ts, "test-aa0002")

	_, out, err := ts.gitStatus(context.Background(), nil, jsonio.GitStatusToolInput{ID: "test-aa0002"})
	if err != nil {
		t.Fatalf("git_status: %v", err)
	}
	if out.Ahead != 1 || out.WorktreePath == "" {
		t.Errorf("git_status = %+v", out)
	}
}

func TestGitTools_NoBranch(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	seedMCPIssue(t, "test-aa0003", "Not started", "PLANNED")

	_, _, err := ts.gitStatus(context.Background(), nil, jsonio.GitStatusToolInput{ID: "test-aa0003"})
	if err == nil || !strings.Contains(err.Error(), "test-aa0003") {
		t.Errorf("git_status error = %v, want it to name the issue", err)
	}
	_, _, err = ts.gitDiff(context.Background(), nil, jsonio.GitDiffToolInput{ID: "test-aa0003"})
	if err == nil || !strings.Contains(err.Error(), "no branch") {
		t.Errorf("git_diff error = %v, want no branch", err)
	}
}

func TestGitTools_RequireArguments(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	ctx := context.Background()
	if _, _, err := ts.gitStatus(ctx, nil, jsonio.GitStatusToolInput{}); err == nil {
		t.Error("git_status without id succeeded")
	}
	if _, _, err := ts.gitDiff(ctx, nil, jsonio.GitDiffToolInput{}); err == nil {
		t.Error("git_diff without id succeeded")
	}
	if _, _, err := ts.gitShow(ctx, nil, jsonio.GitShowToolInput{SHA: "--output=/tmp/x"}); err == nil {
		t.Error("git_show with an option as sha succeeded")
	}
	if _, _, err := ts.codeHistory(ctx, nil, jsonio.CodeHistoryToolInput{Path: "../etc/passwd"}); err == nil {
		t.Error("code_history with an escaping path succeeded")
	}
}

func TestGitLogAndCodeHistoryTools(t *testing.T) {
	ts, _ := setupGitToolset(t, nil)
	startMCPIssue(t, ts, "test-aa0004")
	ctx := context.Background()

	_, log, err := ts.gitLog(ctx, nil, jsonio.GitLogToolInput{ID: "test-aa0004"})
	if err != nil {
		t.Fatalf("git_log: %v", err)
	}
	if len(log.Commits) != 1 || !slices.Equal(log.Commits[0].IssueIDs, []string{"test-aa0004"}) {
		t.Errorf("git_log = %+v", log)
	}

	_, hist, err := ts.codeHistory(ctx, nil, jsonio.CodeHistoryToolInput{Path: "base.txt"})
	if err != nil {
		t.Fatalf("code_history: %v", err)
	}
	if len(hist.Commits) != 1 || hist.Commits[0].Subject != "init" {
		t.Errorf("code_history = %+v", hist)
	}

	_, show, err := ts.gitShow(ctx, nil, jsonio.GitShowToolInput{SHA: log.Commits[0].SHA})
	if err != nil {
		t.Fatalf("git_show: %v", err)
	}
	if !strings.Contains(show.Text, "+feature line") {
		t.Errorf("git_show text = %q", show.Text)
	}
}
