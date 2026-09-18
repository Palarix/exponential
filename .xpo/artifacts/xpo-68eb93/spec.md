## Spec: Add `--with-spec` and `--with-walkthrough` to `show` command and MCP tool

## What

Add opt-in flags to include the actual spec and walkthrough markdown content in the
`show` output, reducing a 3-call integration flow to a single call.

## Why

Rendering a full issue detail view via `--json` currently requires separate calls to
`xpo show`, `xpo spec`, and `xpo walkthrough`. Adding opt-in content inclusion lets
integrations fetch everything in one round-trip.

## How

Four files changed.

### 1. `internal/jsonio/output.go` — add fields to `ShowOutput`

```go
Spec         string `json:"spec,omitempty"`
Walkthrough  string `json:"walkthrough,omitempty"`
```

### 2. `internal/jsonio/input.go` — add fields to `ShowToolInput`

```go
IncludeSpec        bool `json:"include_spec,omitempty" jsonschema:"Include spec content"`
IncludeWalkthrough bool `json:"include_walkthrough,omitempty" jsonschema:"Include walkthrough content"`
```

### 3. `cmd/exponential/show.go` — add CLI flags

Add `showWithSpec` and `showWithWalkthrough` bool flags (`--with-spec`, `--with-walkthrough`).

In the JSON path, after building the `ShowOutput`, conditionally call `client.ReadSpec`
and `client.ReadWalkthrough`. If the file doesn't exist, leave the field empty (swallow
the "not found" error). Populate `out.Spec` / `out.Walkthrough`.

For the interactive path, these flags have no effect (the terminal detail view doesn't
render inline spec/walkthrough content).

### 4. `internal/mcpserver/tools.go` — honour new input fields in MCP handler

Same pattern as `include_events`: after building the `ShowOutput`, conditionally read
spec/walkthrough content and populate the fields.

```go
if in.IncludeSpec {
    if content, err := c.ReadSpec(issue.ID); err == nil {
        out.Spec = content
    }
}
if in.IncludeWalkthrough {
    if content, err := c.ReadWalkthrough(issue.ID); err == nil {
        out.Walkthrough = content
    }
}
```

## Acceptance Criteria

1. `xpo show <id> --json --with-spec` includes spec markdown in `"spec"` field
2. `xpo show <id> --json --with-walkthrough` includes walkthrough in `"walkthrough"` field
3. Both flags together work
4. Missing spec/walkthrough → field omitted (no error)
5. MCP `show` with `include_spec` / `include_walkthrough` produces same output
6. `make test` passes
