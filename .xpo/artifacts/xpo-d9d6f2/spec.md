## Spec: Add `spec` CLI command

## What

Add a `xpo spec` command where the default action is **read with glamour rendering**,
not a subcommand menu. Write and delete are subcommands for the less-common mutations.

## Why

The original issue had `xpo spec read <id>` / `xpo spec write <id>` / `xpo spec delete <id>`,
but this breaks with the convention of other commands where the primary action is the default.
`xpo spec <ID>` should just show the spec — same ergonomics as `xpo show <ID>`.

## How

New file: `cmd/exponential/spec.go`.

### Command structure

```
xpo spec <ID>              → read + glamour render (default action)
xpo spec <ID> --json       → read + JSON output (SpecOutput)
xpo spec --json            → full MCP JSON I/O from stdin (all operations via operation field)
xpo spec write <ID>        → read content from piped stdin, write spec
xpo spec delete <ID>       → delete the spec
```

### Parent command (`specCmd`)

- `Use: "spec [id]"`, `Args: cobra.MaximumNArgs(1)`
- Has `--json` flag
- `RunE`:
  - If `--json` and no positional arg: `runSpecJSON()` — full MCP-compatible stdin dispatch
  - If `--json` and positional arg: read spec, emit `SpecOutput` JSON
  - If no `--json` and positional arg: read spec, render with glamour, print to stdout
  - If no `--json` and no arg: show help

### `write` subcommand

- `Use: "write <issue-id>"`, `Args: cobra.ExactArgs(1)`
- Reads content from piped stdin (`readStdinExplicit` or `readAllStdin` if piped)
- Calls `client.WriteSpec(issueID, content)`
- Prints confirmation

### `delete` subcommand

- `Use: "delete <issue-id>"`, `Args: cobra.ExactArgs(1)`
- Calls `client.DeleteSpec(issueID)`
- Prints confirmation

### `runSpecJSON()` — full JSON dispatch

Mirrors the MCP handler (`toolset.spec` in `tools.go:341-385`):
```
stdin → jsonio.DecodeStrict → SpecToolInput → validate issue_id
→ switch operation:
    write  → validate content → client.WriteSpec → SpecOutput{OK, IssueID, Path}
    read   → client.ReadSpec → SpecOutput{OK, IssueID, Path, Content}
    delete → client.DeleteSpec → SpecOutput{OK, IssueID, Path}
```

### Glamour rendering

Use the same pattern as `internal/ui/details.go:115-124`:
```go
renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(min(termWidth-4, 76)),
)
rendered, _ := renderer.Render(content)
fmt.Print(rendered)
```

### Imports

- `encoding/json`, `fmt`, `os`, `path/filepath`
- `github.com/charmbracelet/glamour`
- `github.com/palarix/exponential/internal/exponential`
- `github.com/palarix/exponential/internal/jsonio`
- `github.com/palarix/exponential/internal/storage`
- `github.com/palarix/exponential/internal/ui` (for `TerminalWidth`)
- `github.com/spf13/cobra`

## Acceptance Criteria

1. `xpo spec <id>` prints glamour-rendered spec content
2. `xpo spec <id> --json` emits `SpecOutput` JSON with the read content
3. `echo '{"operation":"write","issue_id":"xpo-123","content":"..."}' | xpo spec --json` writes spec
4. `echo '{"operation":"read","issue_id":"xpo-123"}' | xpo spec --json` reads spec as JSON
5. `echo '{"operation":"delete","issue_id":"xpo-123"}' | xpo spec --json` deletes spec
6. `xpo spec write <id>` reads from stdin and writes
7. `xpo spec delete <id>` deletes the spec
8. `make test` passes
