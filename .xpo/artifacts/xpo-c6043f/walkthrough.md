# Walkthrough: Add `SearchInput` to Labels TopBar

## What was built
The Labels view now has the shared `SearchInput` (from xpo-ef6602) in its TopBar center slot. It filters the label list by name. This makes Labels match Backlog, Dependencies and Timeline, and is part of the WebUI refactor epic xpo-570a07.

## How the pieces fit
1. **`filterLabelsByName` (`web/src/utils/labels.ts`)** is a pure generic helper over `{ name: string }`.
   - It trims and lowercases the query and keeps the labels whose lowercased name contains it.
   - An empty or whitespace-only query returns the *same array reference*, so the `useMemo` downstream doesn't produce a new list when nothing is being filtered.
   - It lives next to the other label utilities so it can be unit-tested without rendering the view, the same approach used for `search-input-utils`.
2. **`Labels.tsx`**:
   - Holds `search` as local state.
   - Derives `visibleLabels` from the existing merged and sorted `labels` memo, so filtering runs after config labels and issue labels are deduplicated. Counts and colors are therefore unaffected.
   - Only the rendered list switches to `visibleLabels`. Create, edit and delete still work by label name.
3. **Keyboard**: `SearchInput` registers `/` at `control` priority through `useKeyboardShortcuts`.
   - The registry skips `/` when the event target is editable (`isEditableTarget`). Typing `/` in the create or edit name inputs therefore inserts the character instead of jumping to search.
   - Because of that, `keyboardNavigationEnabled` didn't need gating here, unlike Backlog, which gates it on its popovers.

## Empty states
There are three cases, in order:
1. No labels and not creating: "No labels yet" with the create link, unchanged.
2. Labels exist but none match: "No labels match “<query>”".
3. Otherwise: the list.

The create form sits above all of these and is never hidden by the search.

## Decisions
These were confirmed with the user before implementation:
- Search matches on the name only, as a substring rather than fuzzy.
- The no-match message is plain text with no clear link, since `Esc` already clears.
- Search state isn't persisted, the same as Dependencies.

## Acceptance criteria
- [x] The Labels TopBar shows `SearchInput` in the center slot with placeholder "Filter labels...". Evidence: `Labels.tsx` `center={<SearchInput … placeholder="Filter labels..." />}`.
- [x] Typing filters labels by name, case-insensitive substring match. Evidence: `filterLabelsByName` and its tests "matches case-insensitively" and "matches substrings anywhere in the name".
- [x] `/` focuses the search box and `Esc` clears it and blurs. Evidence: provided by the shared `SearchInput`; tested by the user.
- [x] A no-match query shows "No labels match", not "No labels yet". Evidence: the new ternary branch in `Labels.tsx`; tested by the user.
- [x] `filterLabelsByName` has unit tests for empty query, case-insensitivity, substring, no match and whitespace trim. Evidence: 6 tests in `labels.test.ts`, written before the implementation and confirmed failing first.
- [x] `make test` passes. Evidence: lint, 422 frontend tests and the Go suite are green.
