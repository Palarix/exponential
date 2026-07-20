# Spec: Clickable priority icon in Backlog view

## Problem

The priority icon in Backlog rows is view-only. Users must open the issue detail or use the right-click context menu to change priority.

## Requirements

- Clicking the priority icon in a Backlog row opens a popover with the `PriorityPicker`.
- Pressing `p` on a focused row opens the priority picker popover.
- Selecting a priority updates the issue and closes the popover.
- The interaction pattern must match the existing status icon popover exactly.

## Design

Reuse the existing `openPopover` state by adding `"priority"` to its type union. Wrap the `PriorityIcon` with the same button+Popover+Picker pattern used for the status icon. Add a `handleQuickPriority` callback following the `handleQuickEstimate` pattern.

## Files changed

- `web/src/components/Backlog/Backlog.tsx`
  - Add `PriorityPicker` to imports
  - Add `"priority"` to `openPopover` type union
  - Add `handleQuickPriority` handler
  - Wrap `PriorityIcon` with button + Popover + PriorityPicker
  - Add `"p"` keyboard shortcut

## Acceptance criteria

- [ ] Clicking priority icon opens picker popover
- [ ] Selecting a priority updates the issue
- [ ] `p` key opens priority picker on focused row
- [ ] Popover closes on selection, Escape, or outside click
- [ ] Existing status/estimate/label popovers unaffected
