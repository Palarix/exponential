# Rebrand: Beats → Exponential (`xpo`)

## Context

The project name "Beats" conflicts with Steve Yegge's `beads`. We're rebranding to **Exponential** (a play on other trackers) with the binary name `xpo`. This is a full, clean-break rebrand — no backward compatibility, no migration, no deprecated fallbacks.

## Key Decisions

| Item | Old | New |
|------|-----|-----|
| Go module | `github.com/palarix/beats` | `github.com/palarix/exponential` |
| Binary | `beats` | `xpo` |
| Data dir | `.beats/` | `.xpo/` |
| User config | `~/.config/beats/` | `~/.config/xpo/` |
| Env prefix | `BEATS_*` | `XPO_*` |
| MCP tools | `beats_list`, etc. | `xpo_list`, etc. |
| MCP server | `"beats"` | `"xpo"` |
| Frontend brand | "Beats" | "Exponential" |
| Commit prefix | `beats:` | `xpo:` |
| Default ID prefix | `beats-` | `xpo-` (fallback in `sanitizePrefix`) |
| Issue ID prefix | (folder-derived) | (unchanged logic) |

## Implementation Phases

### Phase 1: Directory renames

Rename directories first (before touching any code), so file paths are stable for all subsequent edits:

1. `cmd/beats/` → `cmd/xpo/`
2. `internal/beats/` → `internal/xpo/`
3. `claude/skills/beats/` → `claude/skills/xpo/`
4. `claude/skills/beats-backlog/` → `claude/skills/xpo-backlog/`
5. `claude/skills/beats-file/` → `claude/skills/xpo-file/`
6. `claude/skills/beats-work/` → `claude/skills/xpo-work/`

### Phase 2: Go module + imports (~60 files)

1. Update `go.mod` line 1: `module github.com/palarix/exponential`
2. Bulk replace all Go imports: `github.com/palarix/beats/` → `github.com/palarix/exponential/`
3. Update import paths that reference `internal/beats` → `internal/xpo`

Pattern: every `.go` file with `"github.com/palarix/beats/..."` gets updated. The `internal/beats` package becomes `internal/xpo` (package name `xpo`).

### Phase 3: Go package rename

All files in `internal/xpo/` (formerly `internal/beats/`):
- Change `package beats` → `package xpo`
- Update all callers from `beats.Foo` → `xpo.Foo`

Key exported symbols to rename:
- `InitBeats()` → `InitProject()` (more generic)
- `GenerateBeatsAgentDocs()` → `GenerateAgentDocs()`
- `AppendBeatsToAgentFile()` → `AppendAgentInstructions()`
- `MCPConfigStatus.HasBeats` → `MCPConfigStatus.HasXpo`
- `AgentDetectionResult.HasBeatsConfig` → `AgentDetectionResult.HasXpoConfig`
- `CheckCompletionConfig` references — update shell completion for `xpo`

### Phase 4: String literals in Go source

Systematic replacement across all `.go` files:

| Pattern | Replacement | Files |
|---------|-------------|-------|
| `".beats"` (directory path) | `".xpo"` | config.go, remote.go, setup.go, client.go, etc. |
| `"beats-"` (default prefix) | `"xpo-"` | add.go, config.go |
| `"beats"` (fallback name in sanitizePrefix) | `"xpo"` | setup.go |
| `fmt.Sprintf("beats: ..."` (commit msgs) | `fmt.Sprintf("xpo: ..."` | add.go, update.go, comment.go, delete.go, merge.go |
| `"BEATS_*"` env vars | `"XPO_*"` | identity.go, remote.go, config.go |
| `SetEnvPrefix("BEATS")` | `SetEnvPrefix("XPO")` | config.go |
| `"~/.config/beats"` | `"~/.config/xpo"` | config.go, remote.go |
| MCP tool names `"beats_list"` etc. | `"xpo_list"` etc. | tools.go |
| MCP server name `"beats"` | `"xpo"` | server.go |
| `serversMap["beats"]` | `serversMap["xpo"]` | agents.go |
| `servers["beats"]` | `servers["xpo"]` | agents.go |
| Agent doc content `"# Beats Agent Instructions"` | `"# Exponential Agent Instructions"` | agents.go |
| Agent file path `.continue/rules/beats.md` | `.continue/rules/xpo.md` | agents.go |
| `.beats/issues.snapshot.json` | `.xpo/issues.snapshot.json` | setup.go |
| CLI `Use: "beats"` | `Use: "xpo"` | main.go |
| CLI long description "Beats is..." | "Exponential is..." | main.go |
| Error messages mentioning `beats` | Update to `xpo` | main.go |
| Shell completion references | Update binary name | completion.go |

### Phase 5: Frontend

- `web/index.html`: `<title>Beats</title>` → `<title>Exponential</title>`
- `web/src/App.tsx`: default prefix `'beats-'` → `'xpo-'`, title logic, localStorage keys, error message
- `web/src/components/Layout/Layout.tsx`: logo alt text, brand text "Beats" → "Exponential", footer version text
- `web/src/components/Dashboard/Dashboard.tsx`: welcome title, localStorage keys (`beats-dashboard-*` → `xpo-dashboard-*`)
- `web/src/components/Board/Board.tsx`: localStorage key
- `web/src/components/Backlog/Backlog.tsx`: localStorage keys
- `web/src/components/Dependencies/Dependencies.tsx`: localStorage key
- `web/src/components/CommandPalette/CommandPalette.tsx`: prefix strip
- `web/src/components/Cycles/Cycles.tsx`: CLI reference in help text
- `web/src/components/IssueDetail/MergeView.tsx`: commit message template
- `web/src/index.css`: rename `prose-beats` → `prose-xpo` class (definition + all usages)
- All component files using `prose-beats` class name

### Phase 6: Build & CI

- `Makefile`: `BINARY_NAME=xpo`, install target, docker target
- `Dockerfile`: binary path references, ENTRYPOINT
- `.github/workflows/build.yml`: binary names `beats-*` → `xpo-*`, artifact names, checksums
- `.mcp.json`: server name + command

### Phase 7: Config & settings

- `.claude/settings.json`: all `mcp__beats__beats_*` → `mcp__xpo__xpo_*` permission entries, `Bash(./beats` → `Bash(./xpo` etc.
- `.claude/settings.local.json`: update if it references beats MCP server

### Phase 8: Documentation

- `CLAUDE.md`: all MCP tool references, workflow instructions, build commands
- `README.md`: all references
- `CHANGELOG.md`: add rebrand entry under [Unreleased]
- `docs/design.md`: all references
- `AGENTS.md`, `GEMINI.md`: all references
- `claude/skills/*/SKILL.md` files: update tool names and descriptions
- `design_docs/beats-features.md` → rename + update content

### Phase 9: Tests

All `*_test.go` files with hardcoded `beats` references:
- Issue ID prefixes: `"beats-3c2135"` → `"xpo-3c2135"` etc.
- Branch name patterns: `"beats-abc123/feature"` → `"xpo-abc123/feature"`
- `.beats/` path references in test setup
- MCP tool names in integration tests
- Env var names in config tests
- Package import paths (already handled by Phase 2)

Representative files: `branches_test.go`, `inbox_test.go`, `review_test.go`, `remote_test.go`, `integration_test.go`, `tools_test.go`, `agents_test.go`

## Verification

1. `make test` — full test suite passes
2. `make build` — binary compiles as `xpo`
3. Grep verification (all should return 0 hits in `.go` files):
   - `grep -r 'github.com/palarix/beats' --include='*.go'`
   - `grep -r '"beats"' --include='*.go'` (excluding test data that legitimately contains the word)
   - `grep -r '\.beats' --include='*.go'`
   - `grep -r 'BEATS_' --include='*.go'`
4. `./xpo init` works in a temp directory
5. Frontend builds: `cd web && bun run build`
