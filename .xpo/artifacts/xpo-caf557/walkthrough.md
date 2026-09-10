# Walkthrough: Central keyboard navigation provider and shortcut registry

## What was built

The Web UI now has one keyboard event pipeline instead of independent document listeners scattered across views and overlays.

`KeyboardNavProvider`, mounted once around `App`, owns the sole application-level `document.keydown` listener. Components register semantic bindings through hooks:

- `useKeyboardShortcuts` for bindings with individual callbacks
- `useKeyboardHandler` for components whose existing key-switch logic is best preserved as one callback while still publishing canonical per-key metadata
- `useActiveShortcuts` for consumers such as the forthcoming registry-driven KeyboardHelp display

The registry supports stable binding IDs, single keys and sequences, labels and groups, enabled state, help visibility, modifiers, editable-target policy, repeat policy, default prevention, scopes, and priority.

## Architecture

### Provider boundary

`web/src/main.tsx` mounts `KeyboardNavProvider` above `App`. The provider creates one `KeyboardRegistry` instance and attaches its dispatch method to `document.keydown`.

No component outside the provider directly attaches a document keydown listener. Backlog retains one window-level keydown/keyup pair solely for tracking modifier state during drag-and-drop; it is not shortcut dispatch and requires matching keyup state.

### Registration lifecycle

Registration hooks keep the latest registration and callbacks in refs. Mounting adds one registry entry, unmounting removes it, and metadata changes refresh the observable snapshot without reinstalling the provider listener. This avoids stale closures and remains safe under React Strict Mode effect replay.

### Dispatch

For each event, the registry:

1. Rejects IME composition.
2. Collects enabled registrations.
3. Applies blocking scope rules. Enabled overlays block lower-priority controls, views, and globals even when the overlay does not bind the pressed key.
4. Matches key or sequence, modifiers, repeat policy, and editable-target policy.
5. Selects the highest priority: `overlay > control > view > global`.
6. Resolves equal-priority matches deterministically by registration order and warns during development.
7. Applies the binding's default-prevention policy.
8. Invokes only the winning current callback.

### Sequences

Multi-key bindings such as `g` then a destination key are registry state, with a one-second timeout. Prefix keys are consumed. Invalid continuation keys are also consumed rather than being reinterpreted as unrelated shortcuts, preserving the previous App behavior.

## Migration

The following direct document listeners were moved into the registry:

- App global navigation, create, and command-palette shortcuts
- KeyboardHelp toggle and dismissal
- Layout project selector dismissal
- Modal focus trapping and dismissal
- Popover and ContextMenu dismissal/navigation
- New Issue confirm-discard handling
- Backlog and its FilterMenu
- Board
- My Issues
- Inbox
- Dependencies and DepGraph
- Timeline filter and commit-detail overlays
- Issue Detail

Existing component business logic remains local. The keyboard module knows how to dispatch but does not import views or encode actions such as moving a Board card or selecting a Backlog row.

## KeyboardHelp integration

KeyboardHelp is now a normal registry participant:

- Its always-mounted global `?` binding toggles the overlay.
- While open, overlay-priority Escape and `?` bindings close it.
- Close-only bindings are marked `showInHelp: false`.

The component still renders its legacy static shortcut catalog in this issue. `xpo-4ded6b` will replace that display with `useActiveShortcuts()`, showing global plus current view/control registrations from the canonical registry.

## Key decisions

### Active overlays block lower scopes

Priority is not limited to resolving identical keys. An enabled overlay blocks all lower-priority registrations so unmatched keys cannot operate the obscured view—for example, `j` cannot move a Backlog selection behind an open context menu.

### Structured migration adapter

Complex existing view handlers use `useKeyboardHandler`. They retain their proven branch logic while declaring each handled key as registry metadata. This reduced migration risk without creating a second listener path.

### Central mechanics, local meaning

Reusable components and views own the meaning of their shortcuts. Future extracted controls such as Tabs register their own actions, while the provider supplies uniform dispatch and policy.

### Help data is live, not a catalog

The registry exposes callbacks-free active metadata. Inactive and unmounted views are not maintained in a parallel help catalog. This enables KeyboardHelp to become view-aware without documentation drift.

## Verification

- Repository-level `make test` passes, including 269 frontend tests and the Go suite.
- TypeScript and the production Vite build pass.
- ESLint passes.
- `git diff --check` passes.
- A source audit finds the provider as the only `document.keydown` owner.
- Registry tests cover priority, overlay blocking, modifiers, editable-target guards, cleanup, sequences, invalid sequence continuation, and observable help metadata.
- Manual UI top-hat testing found no immediate keyboard regressions.

## Follow-on chain

1. `xpo-4ded6b` consumes `useActiveShortcuts()` and removes KeyboardHelp's static catalog.
2. `xpo-d47781` resumes after that and implements Tabs as a controlled visual selector that registers `[` and `]` through this provider rather than installing a listener.