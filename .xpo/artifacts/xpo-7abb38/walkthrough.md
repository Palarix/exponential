## Walkthrough: Prevent worktree setup output from corrupting `start --json`

## What changed

One file: `internal/exponential/start.go`.

The `worktree_setup` hook's stdout/stderr was piped directly to `os.Stdout`/`os.Stderr`
via `cmd.Stdout = os.Stdout`. Any hook output was emitted before the JSON envelope,
producing invalid mixed output for `start --json` and corrupting the MCP transport stream.

## How it works

### Before

```go
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

Hook output goes directly to process stdout — breaks JSON/MCP callers.

### After

```go
var hookOut bytes.Buffer
cmd.Stdout = &hookOut
cmd.Stderr = &hookOut
```

Hook output is captured into a buffer. After the hook runs, any non-empty output is
trimmed and appended to the `msgs` slice. This means:

- **JSON callers** see hook output in the `messages` array of `StartOutput`
- **MCP callers** see it in the text result (via `strings.Join(msgs, "\n")`)
- **Interactive CLI** sees it printed line-by-line (same as before, just routed through msgs)

The fix is in the internal package (`StartWork`), so it protects all three callers
(CLI, MCP, HTTP) in one place.

## Key decisions

- **Combined stdout+stderr into one buffer:** Hook diagnostics are informational — splitting
  them would add complexity for no user benefit. Both streams go into the same `msgs` entry.
- **Append output regardless of success/failure:** A failing hook may still produce useful
  diagnostic output (e.g. "missing dependency X"), so captured output is appended to msgs
  in both the success and failure branches.
