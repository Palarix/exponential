# Walkthrough: Permission cards collapsed by default

## What was built

Approval cards for calls with details (a shell command, or argument fields) now open collapsed. A collapsed card is one row: the headline, then Deny and Allow once. The headline holds a chevron. Clicking the headline shows the details panel and moves the buttons to a footer. Clicking it again collapses the card. Cards without details have nothing to expand and look as before.

Most approvals are clear from the headline alone ("Merge xpo-1 via xpo merge"). The details panel made every card tall and pushed the thread down.

## How it works

All changes are in `web/src/components/Agent/ApprovalCard.tsx`.

- **State.** The card keeps `open` in local state. It starts from the new `defaultOpen` prop, `false` by default. The state is not saved, so a reload shows the card collapsed again.
- **One switch for the layout.** `expanded = hasArgs && open` picks the layout. Before, `hasArgs` alone picked between the details layout and the single-row layout. A collapsed card with details now takes the single-row path, the one cards without details already used. No new layout was needed.
- **The headline.** The title content (action, detail, "via" tool chip) moved into `title`. With details, the headline is a `<button type="button" aria-expanded>` that toggles `open`. Without details, it stays a `<div>` and has no chevron. `headlineClass` keeps the old rules: `flex-[1_1_auto] py-1` in the single-row layout, so the headline wraps above the buttons instead of squeezing beside them.
- **Chevron alignment.** The headline button is a flex row with `items-center`: the title in a `min-w-0` span, then a `shrink-0` `ChevronRight` that turns 90° when open, as in `ToolGroup`. A first version put the chevron inline in the text with a manual baseline offset, and it sat off center. Review caught that, and the flex row fixed it.

## Keyboard

`approvalKeyDecision` only answers Enter and Esc when the card itself has focus (`onCard`). A focused headline button keeps its own Enter, so Enter on the headline toggles the card and does not allow the call. Enter and Esc on the card still allow once and deny.

## Tests

`ApprovalCard.test.tsx` renders static markup, so it cannot click. The tests use `defaultOpen` to render the expanded state:

- a collapsed card has no `<dl>` and shares a `flex-wrap` row with the buttons
- the chevron is in an `aria-expanded="false"` button and is not rotated when collapsed, and rotated with `aria-expanded="true"` when open
- a card without details has no `lucide-chevron-right`
- the existing details tests (shell command, footer order, argument list) render with `defaultOpen`

The tests were written first and failed before the change. `make test` passes: lint clean, 1108 frontend tests, and the Go suite.

## Acceptance criteria

- [x] A card with details renders collapsed: no details panel, headline and buttons on one row. *(test "opens collapsed"; checked in the browser with a merge call)*
- [x] The collapsed headline shows a chevron pointing right, with `aria-expanded="false"`, centered on the headline text. *(test "puts a chevron in the headline"; alignment checked by the user)*
- [x] Clicking the headline expands the card: the details panel shows, the buttons move to the footer, the chevron turns down, `aria-expanded="true"`. *(test with `defaultOpen`; checked in the browser)*
- [x] Clicking the headline again collapses the card. *(checked in the browser)*
- [x] A card without details has no chevron and looks as before. *(test "shows no panel for a call without arguments")*
- [x] Enter and Esc on the focused card still allow once and deny. *(`approvalKeyDecision` unchanged; the headline button handles its own Enter)*
- [x] `make test` passes.
