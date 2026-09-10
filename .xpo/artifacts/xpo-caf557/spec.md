# Spec: Central keyboard navigation provider and shortcut registry

## What

Introduce a central keyboard-navigation subsystem for the Web UI. A root `KeyboardNavProvider` installs the application's only document-level `keydown` listener, while global features, active views, overlays, and reusable controls register semantic shortcut bindings through a hook.

This replaces the original `useViewKeyboard` proposal, which would have reduced boilerplate but retained one global listener per caller.

## Why

The current independent listeners in `App`, Backlog, Board, My Issues, Inbox, and Dependencies make shortcut behavior dependent on component mount and listener order. Guards, overlay suppression, callback freshness, and default prevention are repeated and can drift. Extracting visual primitives such as `Tabs` amplifies that problem unless controls can declare shortcuts without directly owning global listeners.

This infrastructure supports the component-tree refactor in parent epic `xpo-570a07` and unblocks `xpo-d47781`.

## Acceptance Criteria

- [ ] `KeyboardNavProvider` owns exactly one application-level `document.keydown` listener.
- [ ] `useKeyboardShortcuts` registers and cleans up bindings without adding document listeners.
- [ ] Registrations support stable IDs, keys or key sequences, labels, groups, scopes, priorities, enabled state, callbacks, and `showInHelp` visibility.
- [ ] Dispatch ignores composition and editable targets by default.
- [ ] Modifier matching is explicit; unmodified shortcuts do not fire for Meta, Ctrl, or Alt combinations.
- [ ] Default prevention and held-key repeat behavior are explicit per binding with behavior-preserving defaults.
- [ ] The highest-priority enabled matching registration runs; equal-priority conflicts resolve deterministically and emit a development warning.
- [ ] Callbacks observe current React state without re-installing the document listener.
- [ ] Existing global shortcuts in `App` retain their current behavior, including modified shortcuts and `g` navigation sequences.
- [ ] Existing shortcuts in Backlog, Board, My Issues, Inbox, and Dependencies retain their current behavior.
- [ ] Current popover, modal, palette, and editable-target suppression behavior is preserved or strengthened where the prior behavior was inconsistent.
- [ ] `useActiveShortcuts()` exposes help-ready metadata for enabled global and current view/control registrations so `xpo-4ded6b` can render view-aware help without a duplicate catalog.
- [ ] No application component outside the provider directly registers a document-level `keydown` listener.
- [ ] Focused tests cover matching, guards, priority, conflicts, sequence timeout/reset, enabled state, callback freshness, and cleanup.
- [ ] Frontend tests, production build, lint, and repository tests pass.

## Flow

1. Add a cohesive `web/src/keyboard/` module containing binding types, pure matching/dispatch logic, registry context, `KeyboardNavProvider`, and `useKeyboardShortcuts`.
2. Mount `KeyboardNavProvider` once at the Web UI root around `App`.
3. Implement provider-owned dispatch:
   - normalize keyboard events and modifiers;
   - reject composition and disallowed editable targets;
   - advance or reset pending key sequences;
   - collect enabled matching registrations;
   - select the highest-priority match;
   - warn on ambiguous equal-priority matches;
   - apply the binding's default-prevention and repeat policy;
   - invoke the current callback.
4. Keep registrations stable while storing current callbacks and enabled state in refs or registry updates so dispatch never uses stale closures.
5. Migrate `App` global shortcuts, including command-palette modifiers and `g` sequences, to registered bindings.
6. Migrate the shared overlay primitives (`Modal`, `Popover`, and `ContextMenu`) and `NewIssueModal`, preserving Escape, focus-trap, and capture-priority semantics through overlay registrations.
7. Migrate Layout/ProjectSelector, Backlog, Backlog/FilterMenu, Board, My Issues, Inbox, Dependencies, DepGraph, Timeline overlays, and Issue Detail. Preserve view-specific state changes and suppression conditions.
8. Expose `useActiveShortcuts()` as a read-only snapshot of enabled global and current view/control metadata. Migrate `KeyboardHelp`'s existing `?` toggle and enabled-only-while-open Escape behavior to registry bindings, but leave the help display redesign to `xpo-4ded6b`.
9. Add focused unit tests around the pure dispatcher/registry and migration-sensitive behavior.
10. Confirm with a repository search that the provider is the only application-owned document `keydown` listener, then run all verification commands.

## Proposed API

```tsx
interface ShortcutBinding {
  id: string;
  key: string | readonly string[];
  label: string;
  group?: string;
  showInHelp?: boolean;
  run: (event: KeyboardEvent) => void;
  modifiers?: {
    meta?: boolean;
    ctrl?: boolean;
    alt?: boolean;
    shift?: boolean;
  };
  allowInEditable?: boolean;
  preventDefault?: boolean;
  allowRepeat?: boolean;
}

interface ShortcutRegistration {
  scope: string;
  priority?: "global" | "view" | "control" | "overlay";
  enabled?: boolean;
  shortcuts: readonly ShortcutBinding[];
}

useKeyboardShortcuts(registration);
```

The final spelling may change during implementation if React lifecycle constraints require it, but these semantics are required.

## Decisions

- **One provider listener, not one listener per hook call:** the hook only mutates a central registry.
- **Central mechanics, local domain behavior:** the provider owns dispatch policy; components own semantic actions such as selecting the next tab or opening an issue.
- **Declarative registrations:** components describe bindings and metadata rather than branching over raw keyboard events.
- **Explicit scope priority:** `overlay > control > view > global`. This prevents background actions while a more specific active layer owns the same key.
- **Deterministic conflicts:** if multiple enabled bindings at the winning priority match, registration sequence provides a stable tie-breaker and development builds warn with both binding IDs.
- **Structured modifiers:** modifier matching is data, not ad hoc checks inside every callback.
- **Sequences are provider state:** multi-key navigation such as `g`, then a destination key, shares one timeout/reset policy.
- **KeyboardHelp is a normal producer and consumer:** it remains mounted, registers its own `?` toggle and conditional Escape binding, and can read `useActiveShortcuts()`. The provider stays UI-agnostic; registry-driven rendering remains in `xpo-4ded6b`.
- **The active registry is canonical:** help contains global plus current view/control shortcuts only. Unmounted views are not maintained in a second static catalog.
- **Active views register their own behavior:** only the rendered view participates; reusable controls such as `Tabs` may register at control priority.
- **No centralized business-logic monolith:** the keyboard module must not import individual views or encode view state transitions.

## Edge Cases

- **HIGH — destructive or state-changing collisions:** only the winning binding executes. Ambiguous equal-priority matches warn during development and resolve deterministically rather than invoking multiple callbacks.
- **MEDIUM — overlays and modals:** overlay registrations outrank underlying controls and views; existing explicit `enabled` gates remain available during migration.
- **MEDIUM — sequence prefixes:** a pending sequence resets on timeout, Escape, focus change, or a non-matching continuation. A completed sequence invokes only its final binding.
- **MEDIUM — editable targets:** bindings are suppressed in inputs, textareas, and contenteditable elements unless they explicitly opt in.
- **LOW — IME composition:** no shortcut dispatch while `event.isComposing` is true.
- **LOW — repeated keydown:** bindings preserve normal repeat by default; actions that must be single-shot opt out.
- **LOW — unmount during dispatch:** removed registrations cannot run on subsequent events; registry iteration uses a safe snapshot.
- **LOW — Strict Mode:** effect replay must not create duplicate registrations.
- **LOW — disabled registration:** changing `enabled` takes effect without replacing the provider listener.
- **LOW — platform modifiers:** Meta and Ctrl remain distinct in matching while display metadata may choose a platform label later.

## Assumptions

- Only one Web UI root is mounted per document.
- Views are mutually exclusive at the router level, but overlays and reusable controls may coexist with the active view.
- Current shortcuts are the compatibility baseline; behavioral redesign is out of scope.
- Native/browser shortcuts remain untouched unless an existing application shortcut already overrides them.
- Sequence timeout should preserve the current `g` navigation feel; its exact existing duration will be confirmed during implementation.
- Backlog's window-level keydown/keyup listener used only to track drag modifier state is out of scope because it is not shortcut dispatch and requires keyup state.
- `xpo-4ded6b` and then `xpo-d47781` will consume this infrastructure after this issue is reviewed, documented, and merged.

## Open Questions

None.