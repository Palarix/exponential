# Spec: Permission cards collapsed by default, expand via title chevron

## What

An approval card (`ApprovalCard.tsx`) whose call has details (a command or argument fields) opens collapsed. Collapsed, it uses the single-line layout that cards without details use today: the headline and the Deny / Allow once buttons share a row and wrap when they do not fit. The headline gets a chevron. Clicking the headline expands the card into today's layout: headline, details panel, buttons in a footer. Clicking it again collapses the card.

## Why

Most approvals are read at a glance. The details panel makes each card tall and pushes the thread down, even when the headline says enough.

## How

- `ApprovalCard` gets local state `open`, default `false`. A prop `defaultOpen` sets the start value (tests use it to render the expanded card).
- When the call has details, the headline is a `<button type="button">` with `aria-expanded`. It is a flex row, `items-center`: the headline text in a `min-w-0` span, then a `shrink-0` `ChevronRight` that turns 90° when open, as in `ToolGroup`. The chevron stays centered on the text, also when the text wraps.
- Collapsed: the single-line layout (`flex-wrap` row with the headline and the buttons).
- Expanded: today's details layout.
- A call without details has no chevron and no button: it renders as today.
- The headline button keeps its own Enter: `approvalKeyDecision` only acts on the card itself (`onCard`), so Enter on the focused headline toggles, it does not allow.
- The state is per card and not saved: a reload shows the card collapsed again.

## Acceptance criteria

- [x] A card with details renders collapsed: no details panel, headline and buttons on one row.
- [x] The collapsed headline shows a chevron pointing right, with `aria-expanded="false"`, centered on the headline text.
- [x] Clicking the headline expands the card: the details panel shows, the buttons move to the footer, the chevron turns down, `aria-expanded="true"`.
- [x] Clicking the headline again collapses the card.
- [x] A card without details has no chevron and looks as before.
- [x] Enter and Esc on the focused card still allow once and deny.
- [x] `make test` passes.
