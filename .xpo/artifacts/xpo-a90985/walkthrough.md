# Walkthrough: MCP link tool allows self-links and duplicates

## What changed

Two files: `internal/mcpserver/tools.go` (backend validation), `web/src/components/IssueDetail/PropertySidebar.tsx` (deleted target handling).

## Backend: self-link and duplicate checks

In the `link` tool handler, after resolving both source and target via `GetIssue()`, two checks were added before appending the new dependency:

```go
if src.ID == tgt.ID {
    return nil, linkOut{}, fmt.Errorf("cannot link an issue to itself")
}
for _, dep := range src.Dependencies {
    if dep.TargetID == tgt.ID && string(dep.Kind) == kind {
        return nil, linkOut{}, fmt.Errorf("link %s %s already exists on %s", kind, tgt.ID, src.ID)
    }
}
```

The duplicate check compares both `TargetID` and `Kind`, so different relationship types to the same target are still allowed (e.g. an issue can both `blocks` and `relates_to` the same target).

Note that `GetIssue()` filters out deleted issues, so linking to a deleted issue already fails with "target: issue not found" before reaching these checks.

## Frontend: deleted target display

When a dependency target can't be resolved (deleted issue or old prefixless ID that doesn't match anything), the PropertySidebar previously showed the raw `target_id` as a clickable link pointing to `#/issues/<id>` — which would hit the 404 page.

Now unresolvable targets render as muted italic text with the ID and "(deleted)" label, with no link. The remove button still appears on hover so users can clean up stale dependencies.

```tsx
{target ? (
  <>
    <StatusIcon ... />
    <a href={`#/issues/${target.id}`} ...>{target.title}</a>
  </>
) : (
  <span className="... italic truncate">{dep.target_id} (deleted)</span>
)}
```
