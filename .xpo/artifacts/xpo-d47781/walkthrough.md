# Walkthrough: Extract `Tabs` component with `[`/`]` keyboard cycling

## What was built

A generic, controlled `Tabs<T>` component that renders the underlined tab selector and registers `[`/`]` keyboard cycling through the central keyboard registry (xpo-caf557). Both Backlog and My Issues were migrated to use it, removing duplicated tab markup and Backlog's inline cycling handler. The keyboard help overlay was updated so control-priority shortcuts fold into the active view's group instead of creating separate sections.

A design document (`design-docs/keyboard-system.md`) was added capturing the keyboard system's architecture, priority model, conflict resolution, and design principles — including an industry comparison.

## How the pieces fit together

### Tabs component (`web/src/components/ui/Tabs.tsx`)

`Tabs<T extends string>` accepts:
- `items` — a keyed `Record<T, { label: string }>` (consumers pass their existing config directly; JS insertion order defines display and keyboard order)
- `activeId` / `onChange` — controlled state
- `keyboardNavigationEnabled` — consumers disable cycling when overlays are open

The component registers `[`/`]` at `control` priority via `useKeyboardShortcuts`. It never touches `document.addEventListener`. The cycling logic lives in a pure `cycleTabs` function in `tabs-utils.ts` — extracted to satisfy the `react-refresh/only-export-components` lint rule and to make it independently testable.

### Consumer migration

**Backlog** — the inline `Object.entries(TAB_CONFIGS).map(...)` button loop was replaced with `<Tabs items={TAB_CONFIGS} .../>`. The `[`/`]` block inside `handleKeyboard` was removed along with the `activeTabRef` and `onTabChangeRef` that existed solely for it. The `keyboardNavigationEnabled` prop suppresses cycling when any popover, filter menu, view menu, or context menu is open.

**My Issues** — same pattern. The inline button loop was replaced with `<Tabs>`, with suppression during filter menu or context menu.

Both consumers pass their existing `TAB_CONFIGS` directly — no adapter needed. `Tabs` reads only the `label` field; consumers can attach additional metadata (e.g., Backlog's `statuses` array) without affecting the component.

### Help overlay grouping (`keyboard-help-utils.ts`)

`groupShortcuts` was updated to find the active view's group title (from the first `view`-priority shortcut) and remap any `control`-priority shortcuts into that group. This means `[`/`]` appears under "Backlog" or "My Issues" in the help overlay, not a separate "Tabs" section. When no view is active, control shortcuts keep their own group.

This is a presentation concern — the registry stores the original group and scope for dispatch; only the help utils remap for display.

## Key decisions

- **`cycleTabs` in a separate file** — the `react-refresh/only-export-components` rule forbids exporting non-component functions from a component file. Separating it also made pure unit testing straightforward.
- **`useEffect` for ref syncing** — the `react-hooks/refs` rule flags ref updates during render. Using `useEffect` satisfies the lint rule while keeping callbacks fresh.
- **Control→view folding is general** — any future reusable control registering at `control` priority will automatically fold into the active view's help section. Components don't need to know their parent view's group name.
- **Design doc added** — the keyboard system's architecture, priority model, conflict resolution, and industry comparison were documented in `design-docs/keyboard-system.md` to make the design decisions discoverable for future contributors.

## Acceptance criteria

- [x] A generic `Tabs<T extends string>` component accepts a keyed `items` configuration, `activeId`, and `onChange`
- [x] Backlog and My Issues render their existing tab labels and selected state through `Tabs`
- [x] The migrated tab bars remain visually identical to their current appearance
- [x] `[` selects the previous item and `]` selects the next item in both views, wrapping at either end
- [x] Tab shortcuts do nothing when focus is in an editable target (registry guard)
- [x] Tab shortcuts do nothing while a view popover is open (`keyboardNavigationEnabled` prop)
- [x] `Tabs` registers shortcuts through the keyboard API and does not install a document-level listener
- [x] Existing Backlog-specific inline tab markup and cycling code are removed
- [x] Tab shortcut metadata appears in the keyboard help under the active view's section, not a separate "Tabs" group
- [x] Tests (285 frontend, 8 cycleTabs + 3 help-grouping new), build, and lint pass

## Files changed

| File | Change |
|------|--------|
| `web/src/components/ui/Tabs.tsx` | New — generic tab selector with keyboard registration |
| `web/src/components/ui/tabs-utils.ts` | New — pure `cycleTabs` function |
| `web/src/components/ui/Tabs.test.ts` | New — 8 tests for cycling logic |
| `web/src/components/ui/index.ts` | Added `Tabs` export |
| `web/src/components/Backlog/Backlog.tsx` | Replaced inline tabs and `[`/`]` handler with `<Tabs>` |
| `web/src/components/MyIssues/MyIssues.tsx` | Replaced inline tabs with `<Tabs>` |
| `web/src/components/KeyboardHelp/keyboard-help-utils.ts` | Control-priority shortcuts fold into view group |
| `web/src/components/KeyboardHelp/keyboard-help-utils.test.ts` | 3 new tests for grouping aggregation |
| `design-docs/keyboard-system.md` | New — keyboard system architecture documentation |