# Walkthrough: .gitattributes merge=union for issues.db

## What changed

Two Go files modified, one new file created in the project:

- `internal/exponential/setup.go` — init logic
- `cmd/exponential/doctor.go` — health check

## Why

`.xpo/issues.db` is an append-only NDJSON event log. When two branches each append events, git's default merge produces conflict markers. The `merge=union` strategy tells git to keep lines from both sides, which is the correct semantic for an append-only file.

## setup.go

Added step 5 to `InitProject`, after the `.gitignore` step:

```go
EnsureGitattributesEntry(".xpo/issues.db merge=union")
```

The new `EnsureGitattributesEntry` function follows the same pattern as the existing `EnsureGitignoreEntry`: read the file, check if the entry exists (substring match), append if missing. Creates the file if it doesn't exist.

This runs on both fresh `xpo init` and `xpo init --force` (re-init).

## doctor.go

Added a check between the `.gitignore` and `.mcp.json` checks. Reads `.gitattributes`, checks for the `merge=union` entry. If missing, calls `EnsureGitattributesEntry` to auto-fix and reports both the detection and the fix to the user.
