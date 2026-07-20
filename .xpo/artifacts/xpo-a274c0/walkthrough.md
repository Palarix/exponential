# Walkthrough: Clickable priority icon in Backlog view

## What changed

### 1. Priority icon popover (`web/src/components/Backlog/Backlog.tsx`)

The plain `<PriorityIcon>` was wrapped with the same button + Popover + Picker pattern used by the status icon:

- A `<div>` with `onClick={stopPropagation}` prevents the row click from firing.
- A `<button>` toggles `openPopover` with `type: "priority"`.
- When open, a `<Popover>` renders a `<PriorityPicker>` that calls `handleQuickPriority` on selection.

The `handleQuickPriority` handler follows the same shape as `handleQuickEstimate` — it calls `addDraft(issueId, "UPDATE", { priority })`, closes the popover, and refreshes.

The `p` key was added to the existing keyboard shortcut block (alongside `s` for status, `l` for labels, `e` for estimate) to open the priority picker on the focused row.

### 2. Checkmark/shortcut alignment (`PriorityPicker.tsx`, `StatusPicker.tsx`, `EstimatePicker.tsx`)

All three pickers had the same visual bug: the checkmark for the selected option didn't align with the shortcut numbers on other rows. The checkmark SVG (`w-4 h-4`) and the number `<span>` (`text-xs`) each independently used `ml-auto` to push right, but their different intrinsic widths put them in different horizontal positions.

The fix: both are now wrapped in a single `<span className="ml-auto w-4 flex items-center justify-center shrink-0">` container. The `ml-auto` lives on the wrapper, and `w-4` gives both contents the same fixed-width column. The `ml-auto` was removed from the `CheckIcon` SVG since the wrapper handles positioning.
