### Rationale

The command palette was capped at 560px (`max-w-140` in Tailwind) while the create-new-issue modal uses an 860px max-width. A single class swap brings them in line — no structural changes needed because the dialog already uses `w-full` for responsive scaling.

### Changes

- `web/src/components/CommandPalette/CommandPalette.tsx` — replaced `max-w-140` (560px) with `max-w-[860px]` on the dialog container (line 165). This matches the `2xl` Modal variant used by the create-issue dialog.
- `.xpo/artifacts/xpo-d31711/spec.md` — new file capturing the requirements agreed on before implementation.

### How to verify

1. `make test` and `make frontend` — both pass, confirming no build regressions.
2. Open the app, hit the command palette shortcut (Cmd/Ctrl+K). Confirm it's visually the same width as the create-new-issue dialog.
3. Resize the browser below 860px — the palette should shrink with the viewport (no horizontal overflow, no scrollbar).
4. Open the create-new-issue dialog side-by-side (in separate tabs) to eyeball width parity.

### What to look out for

The arbitrary value `max-w-[860px]` is a one-off rather than a shared token. If the Modal's `2xl` width ever changes, these two dialogs will drift apart again. Acceptable for now since CommandPalette doesn't use the Modal component, but worth noting.