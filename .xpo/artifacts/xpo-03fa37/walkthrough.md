# Walkthrough: AI surface tokens for the agent chat

## What was built
The agent chat UI now has its own surface scale, the `--color-*-ai-*` tokens in `web/src/index.css`. Each layer is a few shades above its app counterpart. The chat popup lifts off the board, and cards, the composer, message bubbles and menus inside it sit one step higher than they would elsewhere in the app. The palette stays monochrome and the primary buttons use the app's indigo accent, so the chat still looks like part of the same app.

| Token | Value | App counterpart |
|---|---|---|
| `surface-ai-1` | `#1a1a1a` | `surface-1` `#161616` |
| `surface-ai-2` | `#202020` | `surface-2` `#1c1c1c` |
| `surface-ai-3` | `#272727` | `surface-3` `#222222` |
| `hover-surface-ai-1/2/3` | `#202020` / `#272727` / `#2e2e2e` | one layer up |
| `border-ai` / `border-ai-subtle` | `#333333` / `#242424` | `border-overlay` / `border-subtle` |
| `button-ai` / `button-ai-hover` | app accent / accent hover | |

## How the pieces fit
There are two mechanisms, and each one covers a case the other can't.

1. **Explicit tokens in agent components.** `ChatPopup`, `AgentView`, `ChatThread`, `ChatComposer`, `ComposerPickers`, `ApprovalCard` and `ChatTitleBar` use only `--color-*-ai-*` for surfaces, hovers, borders and primary buttons. A reader can see in the markup which layer an element sits on.

2. **The `.surface-ai` remap class**, for shared components the agent code doesn't own. `Menu`, `PopoverPanel` and `MenuItem` hard-code app tokens (`surface-2`, `hover-surface-3`, `border-elevated` and others), and they are portaled to `document.body`, outside the popup's DOM. `.surface-ai` redefines those app custom properties to their AI equivalents for its subtree. The agent components pass `className="surface-ai"` to every framed `Menu` and `PopoverPanel` they open. The `ChatPopup` and `AgentView` roots carry the class too, so non-portaled shared content inside them, such as `prose-exponential` code blocks in replies, follows the AI scale without any changes to that code.

## Key decisions
- **Its own token family instead of moving the popup to `surface-3`.** On `surface-3`, the popup would read as one more dropdown, and the elements inside it would have nowhere higher to go.
- **Monochrome, not violet.** The first version used a violet palette. It stood out, but it broke the app's coherence. The token structure didn't change, so the switch only touched values in `index.css`. The spec's "Rejected alternative: violet palette" section has the full violet token table.
- **No ai-0 canvas.** The Agent view is a full page in `main`, not a floating layer. Painting it a solid color covered the `main` gradient from `Layout.tsx` and made it feel separate from the other views, so it now has no background of its own.
- **Shell hovers use `hover-surface-ai-1`.** `ChatTitleBar` and `ChatThread` render both on the popup shell and on the Agent view canvas. One hover step has to read on both, and a step based on the canvas would match the popup's background exactly.
- **The status bar and `IssueChats` stay on app tokens.** They live in the app chrome and the issue detail, not on an agent surface.

## Non-obvious details
- `.surface-ai` is a plain class outside `@theme`. Tailwind's arbitrary values like `bg-[var(--color-surface-2)]` resolve the custom property on the element itself, so a redefinition on an ancestor applies.
- A new agent menu needs `className="surface-ai"`, or it renders on app tokens. The test catches this for framed menus. Bare menus rely on the `PopoverPanel` around them.

## Acceptance criteria
- [x] The AI tokens exist in `index.css`, each with a usage comment, and there is no ai-0. Evidence: `ai-surface.test.ts` "defines --color-…" cases and the "no canvas color" case.
- [x] The in-scope components use no app surface, hover, border or accent tokens. Evidence: the "uses no app surface, border or accent tokens" case for each of the 7 files.
- [x] The Agent view root paints no background. Evidence: the "has no canvas color of its own" case; the user checked it visually.
- [x] Menus opened from the Agent view, the title bar and the composer pickers render on ai-2 with AI hovers. Evidence: the "puts every framed menu and popover panel on the AI surface" cases and the `.surface-ai` remap cases.
- [x] The status bar and `IssueChats` are unchanged. Evidence: neither file is in the diff.
- [x] Checked visually in the dark theme by the user ("lgtm"). There is no light theme.
- [x] `make test` passes: exit 0, 879 frontend tests, plus Go tests.
