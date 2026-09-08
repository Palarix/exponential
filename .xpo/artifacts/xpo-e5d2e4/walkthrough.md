# Walkthrough: Backlog `[` / `]` tab cycling

## What changed

Two files edited to add keyboard shortcuts for cycling through the Backlog view's sub-tabs (All Issues, Backlog, Active, Done).

### `Backlog.tsx`

Added two stable refs (`activeTabRef`, `onTabChangeRef`) alongside the existing ref block (~line 1163), following the same pattern used for `onIssueClickRef` and others — this avoids stale closures in the `useEffect` keyboard handler.

The `[` / `]` handler is inserted early in the `keydown` listener (before the `f` / `/` keys, ~line 1173) so it fires regardless of which row is focused. It reads the `TAB_CONFIGS` keys as an ordered array, finds the current tab's index, and advances or retreats with modular wrapping:

```typescript
const tabs = Object.keys(TAB_CONFIGS) as Tab[];
const idx = tabs.indexOf(activeTabRef.current);
const next = e.key === "]"
  ? (idx + 1) % tabs.length
  : (idx - 1 + tabs.length) % tabs.length;
onTabChangeRef.current(tabs[next]);
```

The handler inherits the existing guards — `isEditableTarget()` suppresses it in inputs/textareas, and the `metaKey`/`ctrlKey` check avoids conflicts with browser shortcuts.

### `KeyboardHelp.tsx`

Added two entries to the Backlog shortcut group: `[` → "Previous tab", `]` → "Next tab".

## Key decisions

- **Tab order derived from `TAB_CONFIGS` key order** rather than a separate constant — keeps the shortcut in sync with the rendered tab bar, which iterates the same object.
- **Placed before the `f`/`/` handlers** so the bracket keys are handled cleanly without falling through to the row-level shortcuts below.
