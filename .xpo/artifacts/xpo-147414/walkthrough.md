# Walkthrough: Relationship Links in Issue Details View Broken

## Root Cause

When creating or updating an issue with dependency links via the MCP `add` or `update` tools, the `LinksToDependencies()` function in `internal/inputs/inputs.go` stored the user-provided `target` value directly as `TargetID` without resolving it to a full issue ID. Agents typically pass short IDs (e.g. `d662a2`), so the dependency was stored without the `xpo-` prefix.

The MCP `link` tool did this correctly — it called `c.GetIssue()` on both source and target to resolve full IDs before storing. But `add` and `update` skipped this step.

In the frontend, `PropertySidebar.tsx` looked up the target issue with `issues.find(t => t.id === dep.target_id)`. With a prefixless `target_id`, the exact match failed, so the fallback raw ID was shown (without prefix) and the link pointed to `#/issues/d662a2` which didn't match any issue.

## Fix 1: Resolve dependency target IDs (backend)

In `internal/mcpserver/tools.go`, both the `add` and `update` tool handlers now resolve each dependency's `TargetID` via `c.GetIssue()` after building the payload:

```go
for i, dep := range payload.Dependencies {
    tgt, err := c.GetIssue(dep.TargetID)
    if err != nil {
        return nil, addOut{}, fmt.Errorf("links[%d]: %w", i, err)
    }
    payload.Dependencies[i].TargetID = tgt.ID
}
```

This ensures full `xpo-`-prefixed IDs are stored in events going forward. The resolution also validates that the target issue exists, giving a clear error if it doesn't.

`LinksToDependencies()` itself was not changed — it has no access to the client/transport, so resolution must happen in the callers.

## Fix 2: Suffix fallback for old data (frontend)

Old events already stored with prefixless IDs need to work without a data migration. Two places were updated:

**`PropertySidebar.tsx`** — the dependency lookup now tries exact match first, then falls back to suffix matching:

```tsx
const target = issues.find(t => t.id === dep.target_id)
  ?? issues.find(t => t.id.endsWith(dep.target_id));
```

The link `href` uses `target.id` (the full ID) when a match is found, rather than the raw `dep.target_id`.

**`App.tsx`** — the selected issue lookup uses the same suffix fallback:

```tsx
const selectedIssue = selectedIssueId
  ? (issues.find(i => i.id === selectedIssueId)
    ?? issues.find(i => i.id.endsWith(selectedIssueId))
    ?? null)
  : null;
```

This handles links from old dependency data that point to prefixless IDs.

## Fix 3: Issue not found page (frontend)

When `selectedIssueId` is set but no matching issue is found, `App.tsx` now renders a 404 page instead of silently falling through to the Backlog view. The page follows the `EmptyState` component's visual pattern: centered layout with a faded search icon, "Issue not found" heading, the searched ID in monospace, a helpful message, and a "Back to Backlog" button.
