# Approval card: "Allow", and a stable layout while sending

## What

Two polish items on `web/src/components/Agent/ApprovalCard.tsx`:

1. The primary button reads "Allow" instead of "Allow once".
2. While a decision is being sent, the spinner sits over the button's label instead of beside it. The label turns `invisible`, so it still holds the button's width. No button changes size, and the collapsed row does not wrap.

## Why

- "Allow once" is long and crowds the collapsed row. The chevron menu already offers "Allow for this chat", so the plain "Allow" is clear. This reverses the "Allow once, not Allow" decision in xpo-aaf8b1. The outcome label after the decision ("Allowed once") stays as it is.
- Today the spinner is put in front of the label. The button gets wider, the row shifts, and the collapsed card can wrap its buttons for a moment.

## How

- Add a small `ButtonLabel({ text, sending })` in `ApprovalCard.tsx`, exported for tests:
  - a `relative inline-grid` wrapper;
  - the label, with `invisible` while sending;
  - while sending, the `Spinner` in an `absolute inset-0` centred overlay.
- Deny and Allow render through `ButtonLabel`. Allow shows the spinner for `allow` and `allow_session`, as today.
- Update the comments that name "Allow once" (the component doc and the buttons comment) and `ApprovalCard.test.tsx`.

## Acceptance criteria

- [ ] The primary button reads "Allow".
- [ ] Sending a decision (Enter, Esc, or a click) does not change the size of any button or the card's layout.
- [ ] The spinner still shows on the button for the decision being sent.

## Tests

- `ApprovalCard.test.tsx`: the split button reads `>Allow<`, and no "Allow once" remains.
- `ButtonLabel` unit tests (static markup): idle renders the label visibly and no spinner; sending renders the same label text with `invisible` plus an absolutely positioned spinner.

## Out of scope

- The "Allowed once" outcome label and the CHANGELOG history of 1.5.0.
