# Walkthrough: Polish first-run CLI UX

## What was built

Unified `xpo init` wizard, reorganized `xpo doctor` with `--fix`, managed blocks for safe integration refresh, and a branded first-run experience. The three separate init subcommands (`init`, `init mcp`, `init skill`) collapsed into one wizard that handles the full setup flow.

## Architecture

### Managed blocks (`internal/exponential/managed_block.go`)

The foundation for safe integration refresh. Agent instruction files (CLAUDE.md, AGENTS.md, etc.) now use delimited blocks:

```
<!-- xpo:begin 1.2.1 sha256:ab12cd34ef56 -->
# Agent Instructions
...content...
<!-- xpo:end -->
```

Key functions: `FindManagedBlock` parses markers (supports markdown `<!-- -->` and plain `#` formats), `ComputeBlockHash` produces a 12-char SHA-256 prefix, `BlockWasEdited` compares stored vs computed hash, `ReplaceManagedBlock` surgically replaces only the block content.

Legacy heading-based sections (no markers) auto-migrate: the first refresh wraps them in markers.

### Health check (`internal/exponential/health.go`)

Extracted from `doctor.go` into a testable internal function. `CheckAgentHealth` classifies each agent's issues into three tiers:

- **AutoFix**: silent fixes (missing MCP, missing skill, stale version)
- **Interactive**: needs user decision (edited managed block, legacy format) — `doctor --fix` shows the replace/keep/diff prompt
- **attention** (in doctor counts): truly unfixable (corrupted issues.db) — shown under "Cannot fix automatically"

The `Configured` flag distinguishes agents the user set up (at least one of MCP/instructions/skill present) from agents merely found on PATH. Doctor only reports problems for configured agents; unconfigured ones get an informational hint pointing to `xpo init`.

### Change plan (`internal/exponential/plan.go`)

`ComputeInitPlan` scans which files would be created (`+`) vs modified (`~`) for a set of agents. Results are sorted: dotfiles first, then alphabetically. Used by both `init` and `doctor --fix` to show a terraform-style preview before any writes.

### Unified init wizard (`cmd/exponential/init.go`)

Fresh init flow: banner → git check → prefix prompt → agent multi-select → change plan → confirm → apply. All prompts happen before any writes, so Ctrl-C at any point leaves no partial state (including git init, which is deferred to the apply phase).

Re-init flow branches by version comparison:
- **Same version, no issues**: one-liner "up to date"
- **Same version, edited blocks**: targeted replace/keep/diff per file (not a full re-init)
- **Stale version**: shows version diffs, change plan, confirm
- **Newer version (downgrade)**: refuses with upgrade hint

### Prompt helpers (`cmd/exponential/prompt.go`)

- `promptConfirm`: inline `? Question (Y/n)` — simple bufio
- `promptInput`: raw terminal mode with live-updating preview line below the input, cursor at end of value
- `promptSelect`: delegates to `huh.Select` for arrow key navigation (replace/keep/diff)
- `withCtrlC`: installs SIGINT handler that prints "No changes were made" and exits 130
- `abortInit`: friendly decline message with command hint, exits 1

### Doctor (`cmd/exponential/doctor.go`)

Reorganized sections: xpo → Project → Integrations → This machine. Added `--fix` (consent-follows-blast-radius) and `--strict` (for CI). Issues.db is now validated with line-number reporting via `storage.ValidateEvents`.

`--fix` handles auto-fixable problems (change plan + confirm) and interactive problems (replace/keep/diff prompt) in one pass. Truly unfixable items show under "Cannot fix automatically." Summary counts what was fixed vs what remains.

### Prefix handling

Stored bare (`pay` not `pay-`). The `-` separator is added at point of use: `add.go` (ID construction), `local_transport.go` (prefix matching), `agents.go` (example ID display), `ui/list.go` (column width). Config normalizes on load by stripping trailing `-`.

### Brand consistency

`ui.Tagline` and `ui.Banner()` defined once in `styles.go`. Used by `main.go` (root command, non-project guard) and `init.go` (wizard header).

## Key decisions

- **`huh` only for multi-select and select** — confirms stay as inline `(Y/n)` prompts (cleaner than huh's button widgets). The prefix input uses raw terminal mode for a live-updating preview.
- **Health check tiers** — AutoFix vs Interactive vs attention emerged from review: the user pointed out that edited blocks shouldn't be silently overwritten by `--fix`, but they also shouldn't be punted to a separate command. The three-tier model lets `--fix` handle them with appropriate UX.
- **Doctor --fix scope** — only touches configured agents. An unconfigured agent on PATH is scope expansion, handled by `xpo init`.

## Files changed

**New:**
- `internal/exponential/managed_block.go` — block parsing, hashing, replacement
- `internal/exponential/managed_block_test.go` — 14 tests
- `internal/exponential/health.go` — agent health check logic
- `internal/exponential/health_test.go` — 10 regression tests for bugs found during review
- `internal/exponential/doctor_scenarios_test.go` — 29 scenario tests
- `internal/exponential/plan.go` — change plan computation
- `internal/version/version_test.go` — version comparison tests
- `cmd/exponential/prompt.go` — inline prompt helpers

**Rewritten:**
- `cmd/exponential/init.go` — unified wizard
- `cmd/exponential/doctor.go` — reorganized sections + --fix
- `cmd/exponential/main.go` — branded welcome, non-project guard

**Deleted:**
- `cmd/exponential/init_mcp.go`
- `cmd/exponential/init_skill.go`

**Modified across codebase:**
- `internal/exponential/agents.go` — managed block writes, `WriteAgentInstructions`
- `internal/exponential/setup.go` — bare prefix, integration_version stamp
- `internal/config/config.go` — `IntegrationVersion` field, prefix normalization
- `internal/exponential/add.go`, `local_transport.go`, `server/handlers.go` — separator at point of use
- `internal/storage/read.go` — `ValidateEvents` with line numbers
- `internal/ui/styles.go` — `Tagline`, `Banner()`, `InfoPrefix`
- `internal/ui/list.go` — ID column width for bare prefix
- `internal/version/version.go` — `CompareVersions`
- `internal/demo/demo.go` — bare prefix
- `cmd/exponential/utils.go` — shared `isInteractive()`
