package mcpserver

import (
	"context"
	"fmt"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Read-only git tools. They run git on this machine, where the hub and the
// worktrees live, so only the stdio server registers them: `xpo serve`'s
// /mcp would read the server's checkout, not the caller's.

const gitToolNote = " Read-only: never changes the repository. Text output is capped at 64 KiB (`truncated: true`); narrow with `path` or `stat`."

func (t *toolset) registerGitTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "git_status",
		Description: "Show an issue branch's state: ahead/behind the default branch, and the staged, unstaged and untracked files in its worktree." +
			gitToolNote,
	}, t.gitStatus)

	mcp.AddTool(s, &mcp.Tool{
		Name: "git_diff",
		Description: "Show what an issue changed: the diff from its merge base with the default branch to its worktree, including uncommitted changes (untracked files are listed by name). Use `stat` for the file list only." +
			gitToolNote,
	}, t.gitDiff)

	mcp.AddTool(s, &mcp.Tool{
		Name: "git_log",
		Description: "List commits (sha, subject, author, date, and the issue IDs in each subject): an issue branch's own commits, or the default branch's, optionally only those touching `path`." +
			gitToolNote,
	}, t.gitLog)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "git_show",
		Description: "Show one commit's metadata, message and patch." + gitToolNote,
	}, t.gitShow)

	mcp.AddTool(s, &mcp.Tool{
		Name: "code_history",
		Description: "Find the commits and issues that touched a file, or a line range with `start_line`/`end_line`, on the default branch. Use it to answer \"why is this code like this?\", then read the issues' specs with `rationale` or `spec`." +
			gitToolNote,
	}, t.codeHistory)
}

func (t *toolset) gitStatus(ctx context.Context, req *mcp.CallToolRequest, in jsonio.GitStatusToolInput) (*mcp.CallToolResult, exponential.GitStatusResult, error) {
	res, err := t.clientFor(req).GitStatus(in.ID)
	if err != nil {
		return nil, exponential.GitStatusResult{}, err
	}
	return textResult(fmt.Sprintf("%s: %d ahead, %d behind %s; %d staged, %d unstaged, %d untracked",
		res.Branch, res.Ahead, res.Behind, res.BaseBranch, len(res.Staged), len(res.Unstaged), len(res.Untracked))), *res, nil
}

func (t *toolset) gitDiff(ctx context.Context, req *mcp.CallToolRequest, in jsonio.GitDiffToolInput) (*mcp.CallToolResult, exponential.GitDiffResult, error) {
	res, err := t.clientFor(req).GitDiff(in.ID, exponential.GitDiffOptions{Path: in.Path, Stat: in.Stat})
	if err != nil {
		return nil, exponential.GitDiffResult{}, err
	}
	return textResult(fmt.Sprintf("Diff of %s against %s (merge base %.12s)%s", res.Branch, res.BaseBranch, res.MergeBase, truncatedNote(res.Truncated))), *res, nil
}

func (t *toolset) gitLog(ctx context.Context, req *mcp.CallToolRequest, in jsonio.GitLogToolInput) (*mcp.CallToolResult, exponential.GitLogResult, error) {
	res, err := t.clientFor(req).GitLog(exponential.GitLogOptions{IssueID: in.ID, Path: in.Path, Limit: in.Limit})
	if err != nil {
		return nil, exponential.GitLogResult{}, err
	}
	return textResult(fmt.Sprintf("%d commit(s) in %s", len(res.Commits), res.Range)), *res, nil
}

func (t *toolset) gitShow(ctx context.Context, req *mcp.CallToolRequest, in jsonio.GitShowToolInput) (*mcp.CallToolResult, exponential.GitShowResult, error) {
	res, err := t.clientFor(req).GitShow(in.SHA, in.Path)
	if err != nil {
		return nil, exponential.GitShowResult{}, err
	}
	return textResult(fmt.Sprintf("Commit %.12s%s", res.SHA, truncatedNote(res.Truncated))), *res, nil
}

func (t *toolset) codeHistory(ctx context.Context, req *mcp.CallToolRequest, in jsonio.CodeHistoryToolInput) (*mcp.CallToolResult, exponential.CodeHistoryResult, error) {
	res, err := t.clientFor(req).CodeHistory(exponential.CodeHistoryOptions{
		Path: in.Path, StartLine: in.StartLine, EndLine: in.EndLine, Limit: in.Limit,
	})
	if err != nil {
		return nil, exponential.CodeHistoryResult{}, err
	}
	return textResult(fmt.Sprintf("%d commit(s), %d issue(s) touched %s", len(res.Commits), len(res.Issues), res.Path)), *res, nil
}

func truncatedNote(truncated bool) string {
	if truncated {
		return " — truncated at 64 KiB"
	}
	return ""
}
