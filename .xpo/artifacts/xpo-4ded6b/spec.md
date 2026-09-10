# Spec: Registry-driven, view-aware keyboard help

## What

Refactor `KeyboardHelp` into a normal client of the keyboard registry introduced by `xpo-caf557`. It remains mounted while closed, registers its own toggle and close bindings through `useKeyboardShortcuts`, consumes `useActiveShortcuts()`, and displays only shortcuts relevant to the current UI context.

## Why

The current component contains a hardcoded shortcut catalog separate from the actual handlers, so behavior and documentation can drift. Showing shortcuts for inactive views would require preserving a second static catalog and would undermine the registry as the canonical source of truth.

This issue follows `xpo-caf557` and prepares automatic discovery of control shortcuts such as Tabs in `xpo-d47781`.

## Acceptance Criteria

- [ ] `KeyboardHelp` contains no duplicate static catalog of application shortcuts.
- [ ] The always-mounted component registers `?` to toggle itself through `useKeyboardShortcuts`.
- [ ] While open, it registers Escape at overlay priority to close itself.
- [ ] Help-control bindings that should not appear in the listing use `showInHelp: false`.
- [ ] The overlay reads display data only from `useActiveShortcuts()`.
- [ ] It shows global shortcuts plus shortcuts from the current view and its active controls, including registrations whose behavior is temporarily blocked only because Help itself is open.
- [ ] It does not show shortcuts belonging to inactive or unmounted views.
- [ ] Entries use their registered key/sequence, label, and group metadata.
- [ ] Within each responsive group, entries read top-to-bottom in a column before continuing in the next column.
- [ ] Keyboard and pointer dismissal behavior remains accessible.
- [ ] Tests cover closed/open registrations, active filtering, grouping, hidden bindings, and rendered key metadata.
- [ ] Frontend tests, production build, lint, and repository tests pass.

## Flow

1. After `xpo-caf557` is merged, update `web/src/components/KeyboardHelp/KeyboardHelp.tsx` to use `useKeyboardShortcuts` and `useActiveShortcuts`.
2. Keep `KeyboardHelp` mounted regardless of open state so the global `?` binding is always registered.
3. Register one global help-toggle binding. Register the Escape close binding only while open at overlay priority and mark it hidden from the help listing.
4. Replace `GLOBAL`, `BACKLOG`, `BOARD`, `ISSUE_DETAIL`, `MY_ISSUES`, `INBOX`, and other static shortcut arrays with the active registry snapshot.
5. Filter out `showInHelp: false`, group visible entries by registration metadata, and retain compact key badges. Render groups as full-width stacked sections whose shortcut rows flow responsively into balanced, column-major columns, avoiding empty group columns and matching a top-to-bottom scanning path.
6. Preserve click-outside and close-button behavior; all keyboard handling goes through the provider.
7. Add focused tests and run all verification commands.

## Decisions

- **KeyboardHelp is both producer and consumer:** it registers its own shortcuts and reads the same registry as every other component.
- **Provider remains UI-agnostic:** `KeyboardNavProvider` exposes state and dispatch but never imports or renders `KeyboardHelp`.
- **Live active registrations are canonical:** display global plus current view/control shortcuts only. Opening Help must not remove underlying registrations from the metadata snapshot; Help's overlay priority blocks their execution while they remain visible.
- **No inactive-view catalog:** unmounted views do not appear, preventing a second source of truth.
- **Help toggle is owned by the help component:** `App` owns open state and passes toggle/close callbacks, while `KeyboardHelp` owns shortcut declaration.
- **Scope headings remain, but do not occupy independent columns:** groups stack vertically and each group's entries fill responsive columns. Entries are balanced into columns and ordered top-to-bottom within each column before continuing to the right.
- **Search is not required initially:** the active shortcut set should be compact. A search control may remain only if it demonstrably improves the resulting layout.
- **Overlay mechanics use the registry:** Escape is an overlay-priority binding rather than a component-owned document listener.

## Edge Cases

- **MEDIUM — opening changes active registrations:** the listing excludes hidden help-overlay controls and preserves the underlying global/view/control snapshot.
- **MEDIUM — duplicate display metadata:** registry conflict resolution determines the active binding; help renders the same winner rather than duplicate entries.
- **LOW — responsive column count:** the browser rebalances entries when the modal crosses layout breakpoints while preserving top-to-bottom reading order.
- **LOW — empty view group:** global shortcuts still render.
- **LOW — sequences and modifiers:** badges derive from structured registered metadata and preserve order.
- **LOW — closed component:** `?` remains registered because the component remains mounted; Escape does not.
- **LOW — editable target:** provider policy suppresses `?` consistently with the application convention.
- **LOW — pointer dismissal:** backdrop and close button continue to call `onClose` directly.

## Dependencies

- Depends on `xpo-caf557`.

## Assumptions

- `xpo-caf557` exposes `useActiveShortcuts()` and `showInHelp` metadata as specified.
- `KeyboardHelp` remains mounted from `App` with `isOpen`, `onToggle`, and `onClose` callbacks.
- The current active-view router mounts only one primary view at a time.
- Visual redesign beyond what is needed for live grouping is out of scope.

## Open Questions

None.
