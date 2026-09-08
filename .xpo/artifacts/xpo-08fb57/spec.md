# Spec: Backlog `{`/`}` expand/collapse all parents

## What
Add two keyboard shortcuts to the Backlog view:
- `}` (Shift+]) — expand all parent issue tree nodes
- `{` (Shift+[) — collapse all parent issue tree nodes

## Why
Users must click each parent issue individually to expand or collapse its children. The `{`/`}` keys are a natural extension of the existing `[`/`]` tab-cycling shortcuts — same keys, shifted for a broader action.

## Scope
**Parent issue tree nodes only** — the nested hierarchy managed by `expandedNodes`. Status group sections are out of scope. Ghost parents are not collapsible.

## How

### 1. Wire shortcuts into the keyboard handler
In the `useEffect` handler, add `{`/`}` handling after the `metaKey`/`ctrlKey` guard and before the existing `[`/`]` tab-cycling block. Uses stable refs to the existing `expandAllNodes`/`collapseAllNodes` functions.

### 2. Update KeyboardHelp
Add `{ keys: ["}"], label: "Expand all parents" }` and `{ keys: ["{"], label: "Collapse all parents" }` to the BACKLOG section.

## Acceptance Criteria
- [x] `}` expands all collapsed parent issue nodes
- [x] `{` collapses all expanded parent issue nodes
- [x] Shortcuts are suppressed when a popover is open, focus is in an input, or a modifier key (Cmd/Ctrl) is held
- [x] KeyboardHelp reflects the new shortcuts
- [x] `make test` passes
