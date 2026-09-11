# Spec: Extract shared `Tabs` component

## What

Extract the underlined tab-selector UI and its keyboard cycling behavior into a reusable, typed `Tabs` component built on the central keyboard registry from `xpo-caf557`. `Tabs` is selector-only: each parent view continues to own the active state and render its tab content separately below `TopBar`, while placing `Tabs` in `TopBar.left`. Migrate the Backlog and My Issues views to the component; Timeline is explicitly excluded because its former tabs were replaced by the event-type filter menu in `xpo-eb76cd`.

## Why

Backlog and My Issues still duplicate the same button structure, active/inactive styling, and tab-selection behavior. A shared component keeps those views visually and behaviorally consistent and advances the component-tree refactor tracked by parent epic `xpo-570a07`.

## Acceptance Criteria

- [ ] A generic `Tabs<T extends string>` component accepts a keyed `items` configuration, `activeId`, and `onChange`.
- [ ] Backlog and My Issues render their existing tab labels and selected state through `Tabs`.
- [ ] The migrated tab bars remain visually identical to their current appearance.
- [ ] `[` selects the previous item and `]` selects the next item in both views, wrapping at either end.
- [ ] Tab shortcuts do nothing when focus is in an editable target.
- [ ] Tab shortcuts do nothing while a view popover is open.
- [ ] `Tabs` registers shortcuts through `xpo-caf557`'s keyboard API and does not install a document-level listener.
- [ ] Existing Backlog-specific inline tab markup and cycling code are removed.
- [ ] Tab shortcut metadata appears in the keyboard help under the active view's section, not a separate "Tabs" group.
- [ ] Relevant frontend tests, type checks, and lint checks pass.

## Flow

1. Add a shared `Tabs` component in the Web UI's established shared-component directory.
2. Normalize the keyed configuration once inside `Tabs`, render one button per entry using the existing underlined-tab classes, and call `onChange(id)` when clicked.
3. Register `[` / `]` through `useKeyboardShortcuts` at control priority, using modular wrapping and a consumer-provided enabled condition for open popovers. `Tabs` must not install a document-level listener.
4. Replace the Backlog tab buttons with `Tabs`, pass its existing tab configuration and active selection, and delete its now-duplicated cycling handler.
5. Replace the My Issues tab buttons with `Tabs`, preserving its labels, state transitions, and popover suppression.
6. Update `groupShortcuts` in keyboard-help-utils so control-priority shortcuts fold into the active view's group in the help overlay rather than creating separate sections.
7. Add or adjust focused tests for cycling logic, help-grouping aggregation, and edge cases; run the repository's frontend verification commands.

## Decisions

- **Two consumers, not three:** Timeline is excluded because `xpo-eb76cd` intentionally replaced its exclusive tabs with a multi-select filter menu.
- **Selectors remain decoupled from tab content:** `Tabs` receives controlled state and emits selection changes but does not accept, own, or render tab panels. Consumers may place the selector in `TopBar.left` while rendering content beneath the top bar.
- **Tab semantics belong to `Tabs`, global dispatch does not:** `Tabs` registers previous/next actions with the central keyboard registry from `xpo-caf557` and never installs its own document listener.
- **Consumers pass their existing keyed tab configuration directly:** normalization is owned by `Tabs`, avoiding a repeated `Object.entries(...).map(...)` adapter in every parent. JavaScript insertion order defines both visible and keyboard order, so they cannot drift.
- **Preserve the existing visual contract:** this refactor does not redesign tab styling or introduce a new navigation pattern.
- **Use the project's existing editable-target helper:** avoid divergent definitions of text-entry contexts.
- **Help grouping is a presentation concern, not a component concern:** Components register at their own scope/priority for dispatch correctness. The keyboard-help display aggregates control-priority shortcuts into the active view's group. This means `Tabs` (and any future reusable control) doesn't need to know its parent view's group name — the help overlay resolves it automatically.

## Edge Cases

- **MEDIUM — open popovers:** each consumer must suppress cycling while its filter or other relevant popover is open so the shortcut cannot change background view state unexpectedly.
- **LOW — zero or one item:** keyboard cycling is a no-op.
- **LOW — unknown active ID:** do not emit a change until the active ID matches an item.
- **LOW — modifier keys:** ignore `[` / `]` when command/control/alt modifiers indicate another shortcut.
- **LOW — held keys:** use normal browser repeat behavior unless existing project conventions suppress repeats.
- **LOW — no active view:** if only global and control shortcuts exist (no view-priority registration), control shortcuts keep their own group title in the help overlay.

## Dependencies

- `xpo-caf557` must be completed first; it provides the single-listener provider and shortcut-registration API used by `Tabs`.
- `xpo-4ded6b` follows `xpo-caf557` and must be completed before this issue; it makes keyboard help consume live registrations.

## Assumptions

- Backlog and My Issues are the only current underlined-tab consumers.
- Their current tab labels, ordering, state ownership, and styling remain unchanged.
- The shared component never attaches a document listener; it only registers bindings with `useKeyboardShortcuts`.
- No Timeline keyboard-help entry should be added because Timeline no longer exposes tabs.

## Open Questions

None.