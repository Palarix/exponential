Update project documentation to cover the artifact system.

## Deliverables

### 1. README.md — New "Artifacts" section

Add a section after "CLI reference" (before "Web UI") covering:

- **What artifacts are**: files attached to issues that live in `.xpo/artifacts/<issue-id>/` and travel with the code in git.
- **First-class artifacts**: `spec.md` (design specs written before implementation) and `walkthrough.md` (post-implementation summaries). These have dedicated CLI subcommands and MCP tools.
- **Generic artifacts**: any other file (logs, screenshots, configs) attached via `xpo artifact add --name <filename>`.
- **CLI usage examples** for the `xpo artifact` command group: `add --spec`, `add --walkthrough`, `add --name`, `show --spec`, `show <id> <filename>`, `list`, `delete`.
- **Storage layout**: `.xpo/artifacts/<issue-id>/spec.md`, `.xpo/artifacts/<issue-id>/walkthrough.md`, `.xpo/artifacts/<issue-id>/<generic-file>`.

Also update the "Available tools" table in the "Agent integration (MCP)" section to include the three new MCP tools: `spec`, `walkthrough`, `artifact`.

### 2. CLI help text — Improve `Long` descriptions

Add or improve `Long` descriptions on these cobra commands in `cmd/exponential/artifact.go`:

- `artifactCmd`: add a `Long` field explaining the artifact system, the three artifact types, and storage location.
- `artifactShowCmd`: add a `Long` field documenting the `--spec`/`--walkthrough` shorthands and positional filename arg.
- `artifactDeleteCmd`: add a `Long` field.
- `artifactListCmd`: add a `Long` field explaining the output columns.

`artifactAddCmd` already has a `Long` — review for completeness but no changes required if accurate.

### 3. MCP tool descriptions — Make agent-friendly

Update the MCP tool description strings in `internal/mcpserver/` (find the tool registration code) to follow this pattern for each tool:

- **`spec`**: "Read, write, or delete the design spec (spec.md) for an issue. Specs are written before implementation to capture requirements, acceptance criteria, and design decisions. Use `write` to create/update, `read` to retrieve, `delete` to remove."
- **`walkthrough`**: "Read, write, or delete the implementation walkthrough (walkthrough.md) for an issue. Walkthroughs are written after implementation to document what changed and why. Use `write` to create/update, `read` to retrieve, `delete` to remove."
- **`artifact`**: "Manage generic file artifacts on an issue. Use `add` to attach a file, `read` to retrieve it, `delete` to remove it, `list` to see all artifacts. Cannot write to spec.md or walkthrough.md — use the dedicated spec/walkthrough tools for those."

### 4. CLAUDE.md — Reference artifact tools

Update the MCP tool table to add three new rows:

| Action | MCP tool |
|---|---|
| Read/write/delete a spec | `mcp__xpo__spec` |
| Read/write/delete a walkthrough | `mcp__xpo__walkthrough` |
| Manage generic artifacts | `mcp__xpo__artifact` |

Add a new subsection "Specs and Walkthroughs" under "Usage Notes" explaining:
- Before starting implementation, read the issue's spec (if any) via `spec` with `operation: read`.
- After implementation, write a walkthrough via `walkthrough` with `operation: write` summarizing what changed.
- Use `artifact` for attaching supplemental files (test outputs, design diagrams, etc.).

## Acceptance criteria

- [ ] `xpo artifact --help` prints a multi-line description covering all three artifact types
- [ ] Each subcommand (`add`, `show`, `delete`, `list`) has a `Long` help description
- [ ] README has an "Artifacts" section with storage layout, CLI examples, and concept explanations
- [ ] README MCP tools table lists `spec`, `walkthrough`, and `artifact`
- [ ] MCP tool descriptions for `spec`, `walkthrough`, `artifact` each mention their purpose, supported operations, and constraints
- [ ] CLAUDE.md tool table includes the three artifact tools
- [ ] CLAUDE.md has a "Specs and Walkthroughs" usage-notes subsection
- [ ] `make build` succeeds
- [ ] `make test` passes