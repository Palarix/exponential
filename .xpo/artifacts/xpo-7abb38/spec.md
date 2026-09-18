## Spec: Prevent worktree setup output from corrupting `start --json`

## What

Capture the `worktree_setup` hook's stdout/stderr into a buffer instead of piping
directly to `os.Stdout`/`os.Stderr`, and fold the captured output into the `msgs`
slice returned by `StartWork`.

## Why

When `--json` is set (or MCP transport is active), hook output leaks to stdout before
the JSON envelope, producing invalid mixed output. The hook's stdout contaminates the
JSON/MCP transport stream.

## How

Single file change: `internal/exponential/start.go`, lines 102-112.

### Current code

```go
cmd := exec.Command("sh", "-c", c.Config.WorktreeSetup)
cmd.Dir = absPath
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

### Fix

```go
var hookOut bytes.Buffer
cmd := exec.Command("sh", "-c", c.Config.WorktreeSetup)
cmd.Dir = absPath
cmd.Stdout = &hookOut
cmd.Stderr = &hookOut
if err := cmd.Run(); err != nil {
    msgs = append(msgs, fmt.Sprintf("Warning: worktree_setup hook failed: %v", err))
    if s := strings.TrimSpace(hookOut.String()); s != "" {
        msgs = append(msgs, s)
    }
} else {
    msgs = append(msgs, "Ran worktree_setup hook")
}
```

This preserves hook diagnostics in the `msgs` slice (visible in both JSON and
interactive output) without contaminating stdout.

### Import change

Add `"bytes"` to the import block (if not already present).

## Acceptance Criteria

1. `start --json` emits exactly one valid JSON document even when `worktree_setup` writes to stdout
2. Hook output appears in the `messages` array of `StartOutput`
3. Non-JSON `start` behavior unchanged (hook messages appear via the msgs slice)
4. `make test` passes
