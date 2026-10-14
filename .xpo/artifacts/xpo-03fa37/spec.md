# Agent chat: dedicated AI surface tokens

## What
Give the agent chat UI its own surface scale, the `--color-*-ai-*` tokens. Each layer is a tad above its app counterpart, so the chat popup lifts off the board and agent cards sit a step higher, while staying in the app's monochrome palette.

## Why
The chat popup sits on `surface-1`, the same layer as the app's panels, so it blends into the board. Moving it to `surface-3` would only make it look like one more popover. A separate scale gives the agent UI its own layers to work with.

A violet-tinted first version stood out, but it no longer felt like the same app. The scale is now monochrome. See **Rejected alternative: violet palette** for the full palette.

## Tokens (`web/src/index.css`, `@theme`)

| Token | Value | App counterpart | Use |
|---|---|---|---|
| `--color-surface-ai-1` | `#1a1a1a` | `surface-1` `#161616` | Chat popup shell |
| `--color-surface-ai-2` | `#202020` | `surface-2` `#1c1c1c` | Cards, composer, approvals, user bubble, menus, popovers |
| `--color-surface-ai-3` | `#272727` | `surface-3` `#222222` | Items raised on ai-2 |
| `--color-hover-surface-ai-1` | `#202020` | | Hover on ai-1 and on the Agent view canvas |
| `--color-hover-surface-ai-2` | `#272727` | | Hover on ai-2 |
| `--color-hover-surface-ai-3` | `#2e2e2e` | | Hover on ai-3 |
| `--color-border-ai` | `#333333` | `border-overlay` `#323232` | Popup edge, cards, inputs, menus |
| `--color-border-ai-subtle` | `#242424` | `border-subtle` `#1e1e1e` | Dividers inside the thread and view |
| `--color-button-ai` | `var(--color-accent-primary)` | | Primary buttons (send, allow) |
| `--color-button-ai-hover` | `var(--color-accent-primary-hover)` | | Hover on primary buttons |

Each hover step is one layer up, the same pattern as the app scale. The primary buttons use the app's indigo accent. The tokens stay, so a later palette change touches only `index.css`.

There is no ai-0. The Agent view is a full page in `main`, not a floating layer. It shows the app canvas (the `main` gradient from `Layout.tsx`) like every other view, so it doesn't need its own canvas color. An earlier iteration painted it solid ai-0, which hid that gradient and made the view feel apart from the app.

## Scope

| Area | Treatment |
|---|---|
| `ChatPopup` shell, header, starter chips | AI tokens: shell on ai-1 with `border-ai` |
| `AgentView` header, suggestion cards, recent list | AI tokens. No background of its own: the app canvas shows through |
| `ChatThread` (bubbles, step badges, error tooltip, dividers) | AI tokens |
| `ChatComposer`, `ComposerPickers` | AI tokens: box on ai-2, send button on `button-ai` |
| `ApprovalCard` | AI tokens: card on ai-2, allow button on `button-ai` |
| `ChatTitleBar` rename input and menu | AI tokens |
| Menus and popovers opened from AI surfaces | AI tokens (see below) |
| `AgentStatusBar` bar and chips | App tokens, no change |
| `IssueChats` (issue detail) | App tokens, no change |

### Hovers on the shell and canvas
Buttons and rows that sit directly on the popup shell (ai-1) or the Agent view canvas hover to `hover-surface-ai-1`. `ChatTitleBar` and `ChatThread` render in both places, so they need one hover that reads on both. Items on ai-2 (composer pickers, approval buttons) hover to `hover-surface-ai-2`.

### Menus and popovers
`Menu`, `PopoverPanel` and `MenuItem` are shared UI and hard-code app tokens (`surface-2`, `hover-surface-3`/`-4`). They are portaled to `document.body`, so CSS scoping on the popup root doesn't reach them.

Approach: a `.surface-ai` class in `index.css` remaps the app tokens the shared primitives use to their AI equivalents (`--color-surface-2` and `-3` → ai-2, `--color-hover-surface-3` → `hover-surface-ai-2`, `--color-hover-surface-4` → `hover-surface-ai-3`, `--color-border-elevated`/`-default`/`-overlay` → `border-ai`, `--color-border-subtle` → `border-ai-subtle`). The Agent components pass `className="surface-ai"` to the framed `Menu`/`PopoverPanel` they open. Bare menus sit inside a `PopoverPanel` that carries the class.

The `ChatPopup` and `AgentView` roots also carry `.surface-ai`, so non-portaled shared content inside them (such as `prose-exponential` code blocks in agent replies) follows the AI scale. Agent components themselves use the explicit `--color-*-ai-*` tokens, not the remap.

## Decisions
- The remap is a class (`.surface-ai`), not a data attribute. Confirmed by the user.
- The send and "Allow" buttons use `button-ai`, so all primary actions on AI surfaces match. Confirmed by the user. In the monochrome version, `button-ai` points to the app accent.
- The scale is monochrome, a tad above the app. The violet version was rejected because it broke the app's coherence.
- The Agent view has no canvas color of its own (no ai-0). This came from the user's "make everything feel like the same app again".
- The status bar chips stay on app tokens, because they live in the app chrome.

## Rejected alternative: violet palette
This was the first version, built and checked visually, then rejected. It stood out clearly from the board, but the violet tint made the chat feel like a different app. It used the same tokens, component mapping and `.surface-ai` remap as the final version. Only the values differed, and the Agent view canvas was painted ai-0.

| Token | Value | Origin |
|---|---|---|
| `--color-surface-ai-0` | `#0D0C14` | user |
| `--color-surface-ai-1` | `#13121D` | user |
| `--color-surface-ai-2` | `#191622` | user |
| `--color-surface-ai-3` | `#201D2B` | derived |
| `--color-hover-surface-ai-0` | `#13121D` | derived |
| `--color-hover-surface-ai-1` | `#191622` | derived |
| `--color-hover-surface-ai-2` | `#201D2B` | derived |
| `--color-hover-surface-ai-3` | `#272433` | derived |
| `--color-border-ai` | `#33323F` | user |
| `--color-border-ai-subtle` | `#22202D` | derived |
| `--color-button-ai` | `#3D2159` | user |
| `--color-button-ai-hover` | `#4A2869` | derived |

Two things to know if this is revisited:
- **Lightness.** ai-0 and ai-1 are about as dark as the app canvas (`#0f0f0f`) and darker than the app's panels (`#161616`). The popup stood out by hue, border and shadow, not by being lighter.
- **Buttons.** The send and Allow buttons used `#3D2159` with white text in place of the indigo accent.

It existed as snapshot commit 2ed14f9 on the issue branch, which may not be reachable after the squash merge.

## Acceptance criteria
- [ ] The AI tokens above exist in `index.css`, each with a usage comment. There is no ai-0.
- [ ] Every surface, hover, border and accent token in the in-scope components uses an AI token. Text and semantic colors and `border-focus` stay.
- [ ] The Agent view root paints no background, so the app canvas gradient shows.
- [ ] Menus opened from the Agent view, the title bar and the composer pickers render on ai-2 with AI hovers.
- [ ] The status bar and `IssueChats` are unchanged.
- [ ] Checked visually in the dark theme. There is no light theme today, so a light-theme check is not possible.
- [ ] `make test` passes. `ai-surface.test.ts` guards the tokens, the remap, the absence of app tokens in scope, `surface-ai` on framed menus and roots, and no background on the Agent view root.
