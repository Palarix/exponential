# Spec: Backlog view `[` / `]` to switch sub-tabs

## What

Add keyboard shortcuts `[` and `]` to cycle through the Backlog view's status-filtering sub-tabs: **All Issues → Backlog → Active → Done**.

## Why

The Backlog sub-tabs are only clickable — there's no keyboard shortcut to switch between them. `[` / `]` gives quick linear navigation through the filter tabs without reaching for the mouse.

## How

### Tab order

Use the `TAB_CONFIGS` key order (Backlog.tsx ~line 79):

1. `all` — All Issues
2. `backlog` — Backlog
3. `active` — Active
4. `done` — Done

### Key handling

Add `[` and `]` cases to the existing Backlog keyboard handler (Backlog.tsx ~lines 1168–1270), guarded by `isEditableTarget()`.

- `]` — advance to the next tab; wrap from `done` → `all`.
- `[` — go to the previous tab; wrap from `all` → `done`.

Use the `onTabChange` prop (which calls `setBacklogTab` in App.tsx) to update the active tab.

### Keyboard help

Add a new entry to the keyboard help modal (`KeyboardHelp.tsx`) under an appropriate section:
- `[` / `]` — Previous / Next tab

## Acceptance Criteria

- [ ] `]` moves to the next Backlog sub-tab, `[` to the previous
- [ ] Wraps at edges (Done → All, All → Done)
- [ ] Suppressed when focus is in an input, textarea, or contentEditable
- [ ] Keyboard help modal lists the new shortcuts
- [ ] Existing keyboard shortcuts are unaffected
