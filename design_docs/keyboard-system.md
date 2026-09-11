# Keyboard Shortcut System

## Overview

Exponential uses a centralized keyboard shortcut system built around a single
`document.keydown` listener, a typed registry, and priority-based dispatch. The
design gives every component a clean way to declare shortcuts without touching
the DOM, while the registry resolves conflicts and generates the help overlay
from live registrations.

## Architecture

```
KeyboardNavProvider          (sole document.keydown listener)
  └─ KeyboardRegistry        (dispatch, priority, sequences, metadata)
       ├─ useKeyboardShortcuts()   (declarative registration)
       ├─ useKeyboardHandler()     (single-handler convenience)
       └─ useActiveShortcuts()     (live help metadata)
```

`KeyboardNavProvider` mounts at the application root (`main.tsx`) and owns the
only application-level `document.keydown` listener. No component outside the
provider attaches its own.

## Priority Model

Dispatch uses four fixed priority bands:

| Priority   | Numeric | Purpose                                      |
|------------|---------|----------------------------------------------|
| `global`   | 0       | Always-available shortcuts (navigation, `?`) |
| `view`     | 1       | Active view shortcuts (Backlog, Board, etc.)  |
| `control`  | 2       | Reusable UI controls (Tabs, pickers)          |
| `overlay`  | 3       | Modals, popovers, context menus               |

When a key is pressed, the registry:

1. Filters to enabled registrations.
2. Finds the highest active blocking priority (overlays block by default).
3. Discards bindings below that priority.
4. Among matches, highest priority wins. Ties break by registration order (later wins).
5. In dev mode, same-priority conflicts log a console warning.

This gives us VS Code-style context specificity without boolean `when`
expressions. The fixed bands map directly to how users think about UI layers:
"I'm in a modal" trumps "I'm in the Backlog" trumps "this is a global shortcut."

### Why not `when` clauses?

VS Code uses arbitrary boolean expressions (`editorFocus && !suggestWidgetVisible`)
for keybinding context. That's powerful but complex — it requires a context-key
registry, evaluation engine, and careful ordering rules. Our four-band model
covers every case we've encountered with zero runtime expression evaluation.
If we ever need finer-grained control, the `enabled` flag on each registration
already handles conditional activation.

## Registration

Components declare shortcuts through `useKeyboardShortcuts`:

```tsx
useKeyboardShortcuts({
  scope: "backlog",
  priority: "view",
  shortcuts: [
    { id: "backlog.next", key: "j", label: "Next issue", group: "Backlog", run: handleNext },
    { id: "backlog.prev", key: "k", label: "Previous issue", group: "Backlog", run: handlePrev },
  ],
});
```

Key properties of the registration model:

- **Declarative:** components describe bindings, not listener lifecycle.
- **Automatic cleanup:** React's effect lifecycle removes registrations on unmount.
- **Stable callbacks:** the registry holds a ref to the registration, so callback
  identity changes don't cause re-registration.
- **Conditional activation:** the `enabled` flag disables a registration without
  removing it. The overlay's `blocksLowerPriorities` flag (default `true` for
  overlays) prevents lower-priority shortcuts from firing even if the overlay
  doesn't handle the key.

## Sequences

The registry supports two-key sequences (e.g., `g` then `b` for "Go to Board"):

```tsx
{ id: "nav.board", key: ["g", "b"], label: "Go to Board", run: goToBoard }
```

After the first key of a sequence, the registry waits up to 1 second for the
continuation. Invalid continuations are consumed (not dispatched to other
bindings), matching the behavior users expect from `g`-prefixed navigation.

## Conflict Resolution

| Scenario                             | Behavior                                          |
|--------------------------------------|---------------------------------------------------|
| Different priorities match           | Highest priority wins                             |
| Same priority, different order       | Later registration wins                           |
| Same priority, same key (dev mode)   | Console warning + later registration wins         |
| Overlay active, unmatched key        | Key is consumed (does not leak to lower scopes)   |
| Sequence prefix active, wrong second | Consumed (does not dispatch as standalone key)    |

### Industry comparison

Most keyboard libraries use one of three models:

- **Last-wins** (Mousetrap, hotkeys-js): simple but fragile — registration order
  is implicit and hard to reason about.
- **All-fire** (react-hotkeys-hook): every matching handler runs — no conflict
  resolution at all.
- **Context-specificity** (VS Code): most-specific context wins — powerful but
  requires a context-expression engine.

Our model sits between last-wins and context-specificity: the priority bands
give deterministic, easy-to-reason-about dispatch without boolean expressions.

## Help Overlay

The help overlay (`?`) is generated entirely from live registrations via
`useActiveShortcuts()`. There is no separate static shortcut catalog to maintain.

### Display grouping

Shortcuts declare a `group` string (e.g., `"Backlog"`, `"Global"`) and a
`showInHelp` flag. The help overlay groups shortcuts by their group, with
`"Global"` always first.

**Control-priority shortcuts fold into the active view's group.** A reusable
component like `Tabs` registers at `control` priority with its own group name,
but the help display aggregates it into the view section (e.g., "Backlog"). This
keeps the overlay organized by what the user sees (Global + current view) rather
than by internal component structure. If no view is active, control shortcuts
keep their own group.

This is a presentation concern — the registry stores the original group for
dispatch; only the help utils remap it for display.

### What gets shown

- Shortcuts with `showInHelp: false` are excluded.
- Only enabled registrations appear.
- Alternate bindings with the same label merge into one row (e.g., `j / ArrowDown`).

## Editable Target Guards

The registry checks `isEditableTarget(event)` — true when focus is in an
`<input>`, `<textarea>`, or `contentEditable` element. Shortcuts are suppressed
in editable targets unless `allowInEditable: true` is set. This guard runs once
in the registry, not in each component.

## Design Principles

1. **One listener.** The provider is the only application-owned `document.keydown`
   listener. Components never touch `document.addEventListener`.

2. **Declare, don't manage.** Components describe what shortcuts they want. The
   registry handles dispatch, priority, guards, sequences, and cleanup.

3. **Priority bands over boolean expressions.** Four levels cover our needs
   without a context-expression engine. The `enabled` flag handles edge cases.

4. **Live help, no duplicate catalog.** The help overlay reads from the same
   registrations that drive dispatch. Shortcuts can never drift from behavior.

5. **Display grouping is a presentation concern.** Components register at their
   own scope for dispatch correctness. The help overlay aggregates into
   user-facing groups (Global + active view).
