## Walkthrough: Add `--with-spec` and `--with-walkthrough` to `show` command and MCP tool

## What changed

Four files:
- `internal/jsonio/output.go` — added `Spec` and `Walkthrough` fields to `ShowOutput`
- `internal/jsonio/input.go` — added `IncludeSpec` and `IncludeWalkthrough` to `ShowToolInput`
- `internal/mcpserver/tools.go` — MCP `show` handler honours the new input fields
- `cmd/exponential/show.go` — CLI `--with-spec` and `--with-walkthrough` flags
- `internal/ui/details.go` — `RenderIssueDetails` accepts optional spec/walkthrough content

## How it works

### JSON mode (CLI and MCP)

When `--with-spec` / `include_spec` is set, `client.ReadSpec(issueID)` is called and the
content is placed in the `"spec"` field of `ShowOutput`. Same for walkthrough. If the file
doesn't exist, the error is swallowed and the field is omitted (`omitempty`).

This matches the existing `include_events` pattern in the MCP handler.

### Interactive mode

Spec and walkthrough content is passed into `RenderIssueDetails` via a `DetailOptions`
struct (variadic parameter so existing callers don't break). The content renders with
glamour inside the detail view, positioned after Description and before Sub-Issues:

```
Header → Details → Description → Spec → Walkthrough → Sub-Issues → History → Comments
```

Headers use `ui.RenderHeader()` for visual consistency with History and Comments sections.

## Key decisions

- **Variadic `DetailOptions`:** Adding a new required parameter to `RenderIssueDetails`
  would break all existing callers. Using `...DetailOptions` makes the spec/walkthrough
  content opt-in at the call site with zero changes to existing code.
- **Section placement after Description:** Spec and walkthrough are design artifacts —
  they belong with the issue's content (description), not its activity stream (history,
  comments). The user confirmed this ordering.
- **Glamour rendering inside `RenderIssueDetails`:** Each section creates its own glamour
  renderer (same pattern as the Description section). This keeps rendering self-contained
  per section rather than sharing a renderer instance.
