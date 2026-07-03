### Rationale

The artifact system (specs, walkthroughs, generic file attachments) shipped without any user-facing documentation. CLI `--help` output had bare one-liners, MCP tool descriptions were terse and didn't explain *when* to use each tool, and neither README nor CLAUDE.md mentioned the three new tools. This PR is a single documentation pass across all four surfaces — no code logic changed.

### Changes

- **`cmd/exponential/artifact.go`** — Added `Long` descriptions to four cobra commands: `artifactCmd` (explains the three artifact types and storage layout), `artifactShowCmd` (documents `--spec`/`--walkthrough` shorthands vs positional filename), `artifactDeleteCmd` (same filename resolution pattern), `artifactListCmd` (documents the four output columns: TYPE, FILENAME, UPDATED, UPDATED BY). `artifactAddCmd` already had a complete `Long` and was left untouched.

- **`internal/mcpserver/tools.go`** — Rewrote `Description` strings for the `spec`, `walkthrough`, and `artifact` MCP tool registrations. Each now states its purpose, when to use it, the supported operations, and (for `artifact`) the spec.md/walkthrough.md restriction. Wording follows a consistent pattern across all three.

- **`README.md`** — Added an "Artifacts" section between "CLI reference" and "Web UI" covering concepts (first-class vs generic), CLI usage examples for all subcommands, and a storage layout diagram. Added three rows (`spec`, `walkthrough`, `artifact`) to the MCP tools table in the "Agent integration" section.

- **`CLAUDE.md`** — Added three rows to the MCP tool table (`mcp__xpo__spec`, `mcp__xpo__walkthrough`, `mcp__xpo__artifact`). Added a "Specs and Walkthroughs" subsection under "Usage Notes" explaining the read-spec-before-implementation / write-walkthrough-after workflow.

- **`.xpo/issues.db`** — Status transitions and completion comment for the issue tracking this work.

- **`.xpo/artifacts/xpo-4f1ba7/spec.md`** — The spec for this issue, stored as a first-class artifact (dogfooding the system being documented).

### How to verify

1. **CLI help text**: run `xpo artifact --help` and confirm it prints the multi-line description covering all three artifact types. Repeat for `xpo artifact show --help`, `xpo artifact delete --help`, `xpo artifact list --help`.
2. **Build and test**: `make build` and `make test` — both should pass with no changes to any Go logic.
3. **README review**: open `README.md`, find the "Artifacts" section, confirm it appears between "CLI reference" and "Web UI". Check the MCP tools table for the three new rows.
4. **CLAUDE.md review**: confirm the tool table has the three new rows and the "Specs and Walkthroughs" subsection exists under "Usage Notes".
5. **MCP descriptions**: grep `internal/mcpserver/tools.go` for the three `Description:` strings and confirm they match the spec's prescribed wording.

### What to look out for

- The README CLI examples show flags like `--spec --file spec-draft.md` and `--name debug.log --file /tmp/debug.log` — verify these match the actual flag names in `artifactAddCmd`. If the CLI flag names drifted during development, the README examples would be misleading.
- The `artifactListCmd` `Long` claims columns are TYPE, FILENAME, UPDATED, UPDATED BY — worth a quick `xpo artifact list <id>` against a real issue to confirm the actual column headers match.