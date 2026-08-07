# Walkthrough: Exclude completed children from epic status cascade

## What changed

Two files were modified: `web/src/components/Backlog/Backlog.tsx` and `web/src/components/IssueDetail/PropertySidebar.tsx`.

## The bug

When changing a parent issue's status, the "Update sub-issues?" modal included DONE children in the preview and would reset them to the new status (e.g. PLANNED) when "Update all" was clicked. Additionally, changing status from the detail view had no child-cascade prompt at all.

## The fix

### Backlog (`Backlog.tsx`)

#### `handleQuickStatus`

Filters direct children to decide whether to show the modal. The filter changed from:

```ts
i.parent_id === issueId && i.status !== status
```

To:

```ts
i.parent_id === issueId && i.status !== status && i.status !== "DONE"
```

It also counts DONE children separately and stores the count as `doneCount` in the modal state.

If the only differing children are DONE (`children.length === 0` after filtering), no modal appears — the parent updates directly.

#### `applyStatusChange`

The same `&& i.status !== "DONE"` guard was added to its child filter, so DONE children are never updated even if modal state is stale.

#### Modal UI

The inline status icon was changed from a wrapped `<span>` with `align-middle` to a direct `<StatusIcon>` with `className="inline-block align-[-1px]"` and smaller `size={12}` for proper text alignment.

When `doneCount > 0`, a sentence is appended inline: "N completed issue(s) will not be updated." — part of the same paragraph, not a separate note.

### Detail view (`PropertySidebar.tsx`)

Previously, `handleStatusChange` simply called `saveDraft("UPDATE", { status })` with no child check.

Now it mirrors the backlog logic:

1. Filters non-DONE children whose status differs from the target.
2. If any exist, opens a `moveChildrenPrompt` modal (same UI as the backlog version).
3. If none exist, updates the parent directly.

A new `applyStatusChange` callback handles the bulk update — it calls `saveDraft` for the parent and `addDraft` for each qualifying child, then refreshes.

The modal is rendered at the bottom of the component's return, using the same layout: title sentence with inline status icon and optional completed-issues note, child preview list, and Abort / Just this issue / Update all buttons.
