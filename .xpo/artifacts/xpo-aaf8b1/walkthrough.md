# Walkthrough: approval card polish, keyboard answers, and surface-agnostic button variants

## What changed and why

The issue started as "make Allow wider and add ⌘+Enter". Each review round on a tophat page (`#/tophat`, deleted before commit) showed the next problem. The result is a reworked approval card and a `Button` whose variants work on any surface.

### The approval card (`web/src/components/Agent/ApprovalCard.tsx`)

The card has two layouts, chosen by `hasArgs`:

- **With arguments:** a column with the headline, the arguments panel and an error, then a footer row with the buttons.
- **Without arguments:** the headline and the buttons share one `flex-wrap` row.

One detail in the one-row layout matters. The headline uses `flex-[1_1_auto]`, not `flex-1`. `flex-1` sets the flex basis to 0, so the row never thinks it overflows. A long headline then squeezes into a narrow column beside the buttons. That was the wrapping the user first reported. With `auto`, the basis is the headline's natural width, and the buttons wrap below when they do not fit. A test pins this.

**Arguments panel.** `approvalArgs` (in `chat-thread-utils.ts`) now returns every non-empty argument. It no longer drops the one that the headline shows, so the `detail` parameter is gone. The card renders the shell command and the key/value rows in one monospace panel. A long value clamps to 3 lines and keeps the full text in `title`.

**Tool name.** "via" has the headline's size, in muted text. The tool name sits in a mono code chip with the `--color-tint-rest` wash. The label is `whitespace-nowrap`, and a plain space comes before it instead of a margin, so a wrapped label does not start indented.

**Buttons.** The footer is `[Deny] [Allow once | ▾]`:

- Deny uses the tint tokens, the same look as `secondary`.
- "Allow once" and the chevron are two joined buttons in the AI primary colors.
- The chevron opens a `Popover` and `Menu`, styled as in `ComposerPickers`, with "Allow for this chat".

The session grant has a wider effect, so it takes a deliberate second click. The footer needs only two controls, which is what lets a card without arguments fit on one row.

### Keyboard model

The card root has `tabIndex={0}` and the composer's focus border (`focus-within:border-[var(--color-border-focus)]`).

1. **Which card is active.** `activeApprovalId(pending, focusedId)` returns the focused card, else the oldest pending approval card, and never a question card. `Turn` in `ChatThread.tsx` tracks the focused card from each card's `onFocusChange`. It passes `active` down.
2. **Taking focus.** When a card becomes active, an effect focuses it with `preventScroll`. The effect skips this when a text field with text in it has focus (`typing()`), so a half-written message keeps its caret.
3. **Enter and Esc.** `approvalKeyDecision` maps the key to `allow` or `deny`. It acts only when the card itself is the event target, so a focused button keeps its own Enter. It ignores modified keys and IME composition. The handler stops propagation, so the app's document-level keyboard registry never sees the key. In particular, Esc on a card does not close the popup.
4. **After the last card.** When the last card goes, its DOM node goes with it and focus falls to `<body>`. `Turn` sees the pending count drop to zero and calls `onCardsDone`, using the pure rule `cardsDone(before, after, focusOnPage)`. `ChatPopup` and `AgentView` bump a counter, and `ChatComposer` focuses its input when its new `focusSignal` prop changes.

The first design used a global ⌘+Enter through the keyboard registry, with a `⌘↵` hint. Both are gone: the hint was noise, and Enter on a focused card is what a form does.

### `Button` variants (`web/src/components/ui/Button.tsx`, `web/src/index.css`)

The tophat showed that `secondary` and `ghost` used fixed surface colors. On `surface-4`, `secondary` looked darker than the surface behind it. The fix is a wash of the text color:

```css
--color-tint-rest:   color-mix(in srgb, currentColor 8%, transparent);
--color-tint-hover:  color-mix(in srgb, currentColor 13%, transparent);
--color-tint-active: color-mix(in srgb, currentColor 18%, transparent);
--color-tint-border: color-mix(in srgb, currentColor 15%, transparent);
```

`currentColor` in an unregistered custom property resolves where the property is used, so each button tints with its own text color. That gives three properties for free:

- the tint lifts off any surface;
- colored text gets a matching colored wash, which is how `tone` works;
- a future light theme darkens instead of lightens.

| Variant | Rest | Hover | Pressed |
|---|---|---|---|
| `primary` | accent | accent hover (lighter) | new `--color-accent-primary-active` (12% black) |
| `secondary` | tint 8% | tint 13% | tint 18% |
| `ghost` | transparent | tint 8% | tint 13% |
| `danger` | 6% black | 6% white | 14% black |
| `warning` (new) | 6% black | 6% white | 14% black, with `--color-text-on-warning` |

- `bordered` adds `--color-tint-border` to any variant.
- `tone` (`neutral`, `danger` or `warning`) sets the text color of `secondary` and `ghost` only. The text classes live in `toneStyles`, not in `variantStyles`, so two text colors never compete.
- The interim `tinted` variant became `secondary`. Only three call sites used `secondary`: "Save draft" twice and "New chat". They now render borderless.
- The solid status colors straddle their base color, so the base sits between rest and hover. `primary` keeps its base color at rest.

## Decisions made during review

- Split button, not a separate "Allow for this chat" button. Layouts with the grant as a third button were tried and replaced.
- Focus plus Enter and Esc, not ⌘+Enter with a hint.
- Key/value panel, not a CLI-style line or raw JSON.
- Code chip for the tool name, chosen over plain muted text.
- `currentColor` tint, not a white overlay. Both look alike on today's dark surfaces, but only the tint follows colored text and a light theme.
- An "Always allow in this project" grant is possible but needs backend storage. The user decided not to file it.
- Hand-rolled buttons across the app move to these variants in xpo-a0d3b9. That issue also covers the `className` / variant background conflict.

## Acceptance criteria

Card layout:
- [x] A card with arguments has three rows: the headline, the arguments panel, and the buttons on the right. Evidence: test "puts the buttons in a footer, below the details"; tophat screenshots at 380px and 720px.
- [x] A card without arguments shows the headline and the buttons on one row. Evidence: test "puts the headline and the buttons on one row…".
- [x] When they do not fit, the buttons wrap below the headline, aligned right. Evidence: tophat "No arguments, long headline" at both widths; `flex-[1_1_auto]` regression test.
- [x] The layout holds at popup width (380px) and full-page width (720px). Evidence: tophat screenshots each round, and the live chat at 1440px and 1000px.
- [x] An error shows below the headline and the arguments. Evidence: the error renders in the content column (with arguments) or below the row (without); code review.

Arguments and tool:
- [x] The panel lists every non-empty argument, including the one in the headline. Evidence: `approvalArgs` tests "lists every argument…"; card test "lists the call's arguments in a monospace panel".
- [x] A shell call shows its command and its other arguments in one panel; `description` stays in the headline. Evidence: `approvalArgs` test with `timeout`.
- [x] A long value shows at most 3 lines, with the full value in `title`. Evidence: tophat Edit `new_string` sample.
- [x] A call without arguments shows no panel. Evidence: test "shows no panel for a call without arguments".
- [x] "via" has the headline's size; the tool name is in a mono code chip. Evidence: test "names the tool in a code chip at the headline's size".
- [x] The "via <tool>" label never breaks inside itself. Evidence: test "keeps the tool label on one line".

Buttons:
- [x] The buttons read `[Deny] [Allow once | ▾]`. Evidence: test "offers Deny and a split Allow once button…".
- [x] The chevron opens a menu with "Allow for this chat", which sends `allow_session`. Evidence: Playwright on the tophat opened the menu, chose the item, and the card resolved.
- [x] The chevron has the name "More ways to allow" and `aria-haspopup="menu"`. Evidence: same card test.
- [x] While a decision sends, the main button shows a spinner and both parts are disabled. Evidence: `busy` disables both and the spinner covers `allow` and `allow_session`; code review.
- [x] Deny uses the `secondary` tint; "Allow once" has a pressed state. Evidence: tophat screenshots; `--color-button-ai-active`.

Keyboard:
- [x] The active card takes focus and shows the focus border. Evidence: Playwright on the tophat (`document.activeElement` was the first card); card test for `tabindex="0"` and the focus-border class.
- [x] The active card is the focused card, else the oldest pending approval card; never a question card. Evidence: `activeApprovalId` tests.
- [x] The card does not take focus from a text field with text in it. Evidence: Playwright with the stand-in composer (focus stayed in the `TEXTAREA`).
- [x] With focus on the card, Enter sends `allow` and Esc sends `deny`. Evidence: `approvalKeyDecision` tests; Playwright (Enter, then Esc, each resolved a card).
- [x] With focus on a button inside the card, Enter presses that button. Evidence: `approvalKeyDecision` "focused buttons" case.
- [x] After a decision, the next pending card takes focus. Evidence: Playwright (focus moved to the next card after Enter and after Esc).
- [x] After the last card, the composer takes focus. Evidence: `cardsDone` tests; the user checked it in a real dev server, where focus moved through the cards and back to the composer.
- [x] The card shows no key hint, and there is no ⌘+Enter shortcut. Evidence: card test (no `<kbd>`, no `↵`); the registry shortcut is removed.

`Button`:
- [x] `secondary` uses the tint tokens at 8, 13 and 18%. Evidence: `Button.test.tsx`.
- [x] `bordered` adds a `--color-tint-border` border. Evidence: `Button.test.tsx`.
- [x] `tone` sets the text color of `secondary` and `ghost`, and is ignored on solid variants. Evidence: `Button.test.tsx`.
- [x] `ghost` is transparent, tint 8% on hover, 13% when pressed. Evidence: `Button.test.tsx`.
- [x] `primary` darkens when pressed. Evidence: `Button.test.tsx`.
- [x] `danger` and `warning` rest at 6% black, hover at 6% white, press at 14% black. Evidence: `Button.test.tsx`; tophat screenshots with base-color reference rows.
- [x] `warning` uses `--color-text-on-warning`. Evidence: `Button.test.tsx`.
- [x] `make test` passes. Evidence: 72 files, 1016 tests, lint clean.

## Worth knowing

- `Button` puts its variant classes and `className` side by side without merging them. A caller's `bg-*` does not reliably override the variant. The tophat hit this, and xpo-a0d3b9 covers it.
- The tophat workflow (a temporary `#/tophat` page with every variant side by side) is a useful pattern for UI polish. It found the `flex-1` bug that screenshots of one live chat had missed.
