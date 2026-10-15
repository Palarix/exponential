## What

In the Agent view, **Esc** in the composer blurs the input and keeps the draft. **`/`** puts focus back in the composer.

## Why

`g → a` focuses the composer. Esc does nothing there, so global shortcuts such as `g → i` type into the input until the user presses Tab 4–5 times. Other text inputs on the board blur on Esc (`SearchInput`). The popup doesn't have this problem: its dialog wrapper minimizes the chat on Esc, which also moves focus away.

## How

- `ChatComposer` textarea `onKeyDown`: on `Escape`, if the event isn't `defaultPrevented` and isn't IME-composing, blur the input. Don't clear the text, and don't stop or prevent anything else.
  - The composer only does this when it's in the Agent view, so the popup keeps its current Esc → minimize. Gate it on a prop (for example `blurOnEscape`, set by `AgentView`), or on `wide`. Prefer an explicit prop.
- `AgentView` registers a `view`-priority shortcut `/` → "Focus composer". It shows in `?` help, isn't allowed in editables, and bumps the existing `composerFocus` signal.
- React's handler on the textarea runs before the registry's `document` listener. Picker menus and popovers move focus into themselves, so their Esc never reaches the textarea.

## Decisions

- **Keep the draft on Esc.** `SearchInput` clears on Esc, but a half-written prompt is worth keeping.
- **Esc never stops a running turn.** Esc only blurs; stopping stays on the Stop button.
- **The popup isn't changed.** Esc still minimizes it in one step.
- **No Esc kbd hint** in the composer footer.

## Acceptance Criteria

- [ ] Agent view: Esc in the composer blurs it, and the typed text stays.
- [ ] After Esc, `g → i` goes to Issues.
- [ ] Agent view: `/` focuses the composer when focus is outside an editable. It is listed in `?` help.
- [ ] `/` typed inside the composer is inserted as text, not swallowed.
- [ ] Esc while the agent is running only blurs. The turn keeps going.
- [ ] Popup: Esc still minimizes the chat, as before.
- [ ] Esc during IME composition doesn't blur.
- [ ] Tests in `ChatComposer.test.tsx` / `AgentView.test.tsx` cover the above; `make test` passes.
