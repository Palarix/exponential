# Agent chat: approval card polish, keyboard answers, and surface-agnostic button variants

## What

The approval card gets a new layout and a keyboard model. The card shows the call's arguments in a monospace panel and names the tool in a code chip. "Allow once" is a split button, and "Allow for this chat" is in its menu. The active card takes focus: Enter allows once, and Esc denies.

The shared `Button` gets surface-agnostic variants. `secondary` and `ghost` use a tint of their own text color, `primary` gets a pressed state, and `danger` and `warning` get a solid scale.

## Why

The first request was a wider primary button and a keyboard shortcut. A tophat review then found these problems:

- A long headline wrapped beside the buttons, and the buttons moved between the headline and the details.
- Arguments such as `force true` read as part of the headline.
- "via tool" used a smaller font than the headline.
- The `⌘↵` hint added noise.
- `secondary` and `ghost` used fixed colors, so they looked wrong on some surfaces. `secondary` looked darker than `surface-4`.

Approval cards come from xpo-7bc86e. The sweep of hand-rolled buttons is xpo-a0d3b9.

## Acceptance Criteria

Card layout:
- [ ] A card with arguments has three rows: the headline, the arguments panel, and the buttons on the right.
- [ ] A card without arguments shows the headline and the buttons on one row.
- [ ] When they do not fit, the buttons wrap below the headline, aligned right.
- [ ] The layout holds at popup width (380px) and full-page width (720px).
- [ ] An error shows below the headline and the arguments.

Arguments and tool:
- [ ] The panel lists every non-empty argument as key/value rows, including the one in the headline.
- [ ] A shell call shows its command and its other arguments in one panel. The `description` stays in the headline only.
- [ ] A long value shows at most 3 lines. Its `title` holds the full value.
- [ ] A call without arguments shows no panel.
- [ ] "via" has the headline's size in muted text. The tool name is in a mono code chip.
- [ ] The "via <tool>" label never breaks inside itself.

Buttons:
- [ ] The buttons read `[Deny] [Allow once | ▾]`.
- [ ] The chevron opens a menu with "Allow for this chat". Choosing it sends `allow_session`.
- [ ] The chevron has the accessible name "More ways to allow" and `aria-haspopup="menu"`.
- [ ] While a decision sends, the main button shows a spinner and both parts are disabled.
- [ ] Deny uses the `secondary` tint. "Allow once" uses the AI primary colors, with a pressed state.

Keyboard:
- [ ] The active card takes focus and shows the focus border, as the composer does.
- [ ] The active card is the focused card, else the oldest pending approval card. Question cards never become active.
- [ ] The card does not take focus from a text field that has text in it.
- [ ] With focus on the card itself, Enter sends `allow` and Esc sends `deny`.
- [ ] With focus on a button inside the card, Enter presses that button.
- [ ] After a decision, the next pending card takes focus.
- [ ] After the last card, the composer takes focus.
- [ ] The card shows no key hint. There is no ⌘+Enter shortcut.

`Button`:
- [ ] `secondary` uses `--color-tint-rest`, `--color-tint-hover` and `--color-tint-active`, a `currentColor` wash at 8, 13 and 18%.
- [ ] `bordered` (default `false`) adds a `--color-tint-border` border (15%).
- [ ] `tone` (`neutral`, `danger` or `warning`) sets the text color of `secondary` and `ghost`. The tint follows the text color.
- [ ] `ghost` is transparent at rest, `--color-tint-rest` on hover, and `--color-tint-hover` when pressed.
- [ ] `primary` gets `--color-accent-primary-active` (accent mixed with 12% black) when pressed.
- [ ] `danger` and `warning` rest at 6% black, hover at 6% white, and press at 14% black, from their base color.
- [ ] `warning` uses `--color-text-on-warning` for its text.
- [ ] `make test` passes.

## Flow

1. In `chat-thread-utils.ts`, add `activeApprovalId(pending, focusedId)`.
2. In `approvalArgs`, keep every non-empty argument. Remove the `detail` parameter.
3. In `ApprovalCard.tsx`, render the headline, the arguments panel and the error in a column (`gap-2.5 px-3 pt-3 pb-2.5`).
4. Render the buttons in a footer row (`px-3 pb-3`). Without arguments, render the headline and the buttons in one `flex-wrap` row.
5. Give the headline `flex-[1_1_auto]` in the one-row layout. Do not use `flex-1`.
6. Render the split button: "Allow once" joined to a chevron button.
7. Open a `Popover` (`bottom-end`) with a `Menu` from the chevron. Use `PopoverPanel` with `surface-ai`.
8. Make the card root focusable (`tabIndex={0}`). Give it `focus-within:border-[var(--color-border-focus)]`.
9. When `active` becomes true, focus the card with `preventScroll`. Skip this when a text field with text has focus.
10. On `keydown` with the card as target, handle Enter and Esc. Call `preventDefault` and `stopPropagation`.
11. In `ChatThread.tsx`, track the focused card per turn. Pass `active` and `onFocusChange` to each card.
12. In `Turn`, call `onCardsDone` when the pending count drops to zero and focus is on the page body.
13. In `ChatPopup.tsx` and `AgentView.tsx`, bump a `focusSignal` from `onCardsDone`. `ChatComposer` focuses its input when `focusSignal` changes.
14. In `index.css`, add the tint tokens, `--color-accent-primary-active`, `--color-button-ai-active`, and the solid danger and warning tokens.
15. In `ui/Button.tsx`, change `secondary`, `ghost`, `primary` and `danger`. Add `warning`, `tone` and `bordered`.
16. Add tests for the card, `activeApprovalId`, `approvalArgs` and `Button`.

## Decisions

- **Focus and Enter/Esc, not a global ⌘+Enter.** Enter and Esc answer the card as in a form, and the focus border shows which card answers. A key hint added noise. This replaces the first design of this issue.
- **Do not take focus from a text field with text.** The user can be writing the next message when a card arrives. An empty composer gives up focus.
- **Esc denies, and does not reach the app.** With focus on the card, Esc is the card's key. The chat popup does not close.
- **Oldest card first, focus overrides.** It matches the thread's top-down order. A tabbed-to card answers instead.
- **"Allow for this chat" in a split-button menu.** The session grant has a wider effect and is the rarer choice. It takes a second, deliberate click. The footer needs only two controls, so a card without arguments fits on one row.
- **"Allow once", not "Allow".** It names the difference to the session grant and matches the outcome label "Allowed once".
- **All arguments in a key/value panel.** Key/value rows work for every tool. A CLI line would invent a syntax, and raw JSON is noisy.
- **Tool name in a code chip.** Same size as the headline, so the line does not shrink. The chip marks the tool as a technical name.
- **`currentColor` tints, not fixed surface colors.** A wash of the text color lifts off any surface. It also works for colored text (`tone`) and a future light theme.
- **`tinted` became `secondary`.** Only three call sites used `secondary`, and its fixed colors failed on `surface-4`. `bordered` keeps the outlined look available.
- **Solid status buttons straddle their base color.** Rest is a little darker and hover a little lighter, so the base color sits between them. `primary` keeps its base color at rest.
- **The card keeps its own button sizes.** Its controls are h-7 and text-xs, as in the composer. `Button` has no such size yet; xpo-a0d3b9 can add one.

## Edge Cases

- MEDIUM: The user is typing in the composer when a card arrives. The card does not take focus. The user can Tab or click into it.
- MEDIUM: The popup and the Agent view show the same chat. Only one thread is mounted at a time today.
- LOW: An IME composition is in progress. The card skips `isComposing` key events.
- LOW: A decision fails. The card keeps focus and shows the error. Enter retries.
- LOW: `Button` puts its variant classes and `className` side by side without merging. A caller cannot override the variant's background. xpo-a0d3b9 covers this.

## Assumptions

- An approval card is pending only while its turn runs.
- `pendingApprovals` returns cards in thread order, oldest first.
- The app is dark-only today.
