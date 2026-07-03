# CLI Commands for Artifacts — Implementation Spec

**Issue:** xpo-4589a9
**Parent spec:** `docs/superpowers/specs/2026-07-03-issue-artifacts-design.md`

## Scope

New file `cmd/exponential/artifact.go` implementing the `xpo artifact` command group with four subcommands: `add`, `show`, `delete`, `list`. All delegate to existing `Client` methods from the business logic layer (xpo-22a371).

## Command Group

```
xpo artifact add <issue-id> [flags]       # add/update an artifact
xpo artifact show <issue-id> <filename>   # print artifact content to stdout
xpo artifact delete <issue-id> <filename> # remove an artifact
xpo artifact list <issue-id>              # list artifacts for an issue
```

The parent `artifactCmd` has no `Run` handler — it prints usage when invoked bare.

## Subcommand Details

### `xpo artifact add <issue-id>`

Writes or overwrites an artifact file on an issue (upsert semantics).

**Flags:**

| Flag | Type | Description |
|---|---|---|
| `--spec` | bool | Write as spec (`spec.md`, type `spec`) |
| `--walkthrough` | bool | Write as walkthrough (`walkthrough.md`, type `walkthrough`) |
| `--name <filename>` | string | Filename for generic artifacts |
| `--file <path>` | string | Read content from a file on disk |

**Filename resolution:** `--spec` and `--walkthrough` are mutually exclusive with each other and with `--name`. When neither shorthand is set, `--name` is required.

**Content source priority:** `--file <path>` > piped stdin > error.

**Dispatch:**
- `--spec` -> `client.WriteSpec(issueID, content)`
- `--walkthrough` -> `client.WriteWalkthrough(issueID, content)`
- `--name` -> `client.AddArtifact(issueID, "generic", filename, content)`

### `xpo artifact show <issue-id> [filename]`

Prints artifact content to stdout (raw, no framing).

**Flags:** `--spec`, `--walkthrough` shorthands. Exactly one source required (positional or flag).

**Dispatch:** `client.ReadArtifact(issueID, filename)` for all cases.

### `xpo artifact delete <issue-id> [filename]`

Removes an artifact. Same filename resolution as `show`.

### `xpo artifact list <issue-id>`

Table output with columns: TYPE, FILENAME, UPDATED, UPDATED BY. Uses `humanize.Time()` for relative timestamps.
