## What changed

In the Agent view, **Esc** in the composer now blurs the textarea and keeps the draft. Global shortcuts such as `g → i` work again after `g → a`. **`/`** puts focus back in the composer. The chat popup isn't changed: Esc there still minimizes it.

## How it fits together

- **`composerEscapeBlurs`** (`chat-thread-utils.ts`) decides whether a keydown blurs the composer. It is true only for `Escape`, when the composer opts in, outside IME composition, and when no earlier handler called `preventDefault`. It sits next to `approvalKeyDecision` and follows the same pattern: the key logic is a pure function. The component tests render static markup and can't dispatch keys, so the rule is tested here.
- **`ChatComposer`** gets a `blurOnEscape` prop. Its textarea `onKeyDown` already handled Enter; Esc is now an `else if` that calls `inputRef.current.blur()`. The text isn't cleared. `SearchInput` clears on Esc, but a half-written prompt is worth more than a search term.
- **`AgentView`** passes `blurOnEscape` and registers a view-priority shortcut built by `focusComposerShortcut` (`/` → "Focus composer", group "Agent"). Its `run` bumps the existing `composerFocus` signal, the same one used when the last approval card is answered. The registration sits above the early returns, so the hook order stays stable. It is disabled whenever no composer is on screen: agent unavailable, missing chat, chat history page, archived chat.

## Why it works with the other Esc handlers

The keyboard registry listens on `document`. React's handler on the textarea runs first, at the root. Picker menus and popovers move focus into themselves, so their Esc never reaches the textarea. The `defaultPrevented` check covers any handler that took Esc earlier. The `/` binding doesn't set `allowInEditable`, so the registry ignores it inside the composer and a typed `/` stays text.

Opting in with a prop, rather than checking `wide`, keeps the popup's behaviour explicit. There, the dialog wrapper's Esc → minimize already moves focus away.

## Decisions

- **The draft is kept on Esc.**
- **Esc never stops a running turn.** It only blurs.
- **The popup isn't changed.** It keeps one-step minimize on Esc.
- **No Esc hint** in the composer footer.
- **`m` to minimize the popup was not added.** Esc and ⌘J / Ctrl+J already cover it, and a bare letter can't work from the composer.

## Follow-ups discussed, not done

- In the Agent view, ⌘J / Ctrl+J opens the popup on top of it, sometimes on the same chat. Docking via Ctrl+J was proposed. The user is thinking about it, so no change and no issue yet.

## Acceptance Criteria

- [x] Agent view: Esc in the composer blurs it, and the typed text stays. Tophatted by the user in the dev server.
- [x] After Esc, `g → i` goes to Issues. Tophatted by the user.
- [x] Agent view: `/` focuses the composer when focus is outside an editable, and is listed in `?` help. `focusComposerShortcut` test checks the key, the label, that it isn't allowed in editables, and that it shows in help. Tophatted by the user.
- [x] `/` typed inside the composer is inserted as text. The binding has no `allowInEditable`, so the registry skips editable targets (`registry.ts:78`).
- [x] Esc while the agent is running only blurs. `composerEscapeBlurs` doesn't depend on `running`, and nothing calls `onStop`.
- [x] Popup: Esc still minimizes. `ChatPopup` doesn't pass `blurOnEscape`, and the `composerEscapeBlurs({ enabled: false })` test covers it. Tophatted by the user.
- [x] Esc during IME composition doesn't blur. `composerEscapeBlurs` test with `composing: true`.
- [x] Tests added; `make test` passes (lint, 1120 frontend tests, Go tests).
