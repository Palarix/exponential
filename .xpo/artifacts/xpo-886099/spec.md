# Spec: Exclude completed children from epic status cascade

## Problem

When changing a parent's status, the "Update sub-issues?" modal includes children with status `DONE`. Clicking "Update all" resets completed children back to the new status (e.g. PLANNED), which is incorrect.

## Fix

Three locations in `Backlog.tsx` need to exclude DONE children:

1. **`handleQuickStatus`** (~line 198): The filter that populates `moveChildrenPrompt.children` should add `&& i.status !== "DONE"`.
2. **`applyStatusChange`** (~line 184): The filter that selects children for bulk update should add `&& i.status !== "DONE"`.
3. **Modal UI** (~line 1689): Show a note below the child list: "N completed issue(s) will not be updated" when there are DONE children.

## Acceptance Criteria

- [ ] DONE children do not appear in the preview list.
- [ ] DONE children are not updated when "Update all" is clicked.
- [ ] When DONE children exist, a note is shown below the preview.
- [ ] If all non-DONE children match the target status (i.e. only DONE children differ), no modal appears — parent updates directly.
- [ ] `make test` and `make build` pass.
