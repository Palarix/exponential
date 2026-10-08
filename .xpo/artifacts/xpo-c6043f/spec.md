# Add `SearchInput` to Labels TopBar

## What
Put the shared `SearchInput` (from xpo-ef6602) in the Labels view's `TopBar` `center` slot. It filters the label list by name.

## Why
This is part of the WebUI refactor epic xpo-570a07. Backlog, Dependencies and Timeline already have a search box in the TopBar center slot. Labels is one of the views still missing one.

## How
- Add a pure helper `filterLabelsByName<T extends { name: string }>(labels: T[], query: string): T[]` to `web/src/utils/labels.ts`.
  - It trims the query and matches it case-insensitively as a substring of `name`.
  - An empty or whitespace-only query returns the input unchanged.
- In `Labels.tsx`:
  - Add `const [search, setSearch] = useState("")`.
  - Derive `visibleLabels = useMemo(() => filterLabelsByName(labels, search), ...)`.
  - Render the list from `visibleLabels`.
- `TopBar center={<SearchInput value={search} onChange={setSearch} placeholder="Filter labels..." />}`.
  - Keyboard: `/` focuses the search box and `Esc` clears it and blurs, both through `SearchInput`.
  - The keyboard registry already ignores `/` typed inside the create/edit inputs (`isEditableTarget`), so `keyboardNavigationEnabled` doesn't need gating.
- Empty states:
  - With no labels at all, keep the existing "No labels yet" state.
  - When labels exist but none match, show "No labels match “<query>”".
- The create form stays visible no matter what the search says.

## Acceptance criteria
- [ ] The Labels TopBar shows `SearchInput` in the center slot with placeholder "Filter labels..."
- [ ] Typing filters labels by name, case-insensitive substring match
- [ ] `/` focuses the search box and `Esc` clears it and blurs
- [ ] A no-match query shows a "No labels match" message, not "No labels yet"
- [ ] `filterLabelsByName` has unit tests (empty query, case-insensitivity, substring, no match, whitespace trim)
- [ ] `make test` passes

## Decisions (please confirm)
1. Match on the name only (not color or count), as a substring rather than fuzzy.
2. The no-match state is a plain text message with no "clear search" link (`Esc` already clears).
3. Search state is local and not persisted across navigation, which matches Dependencies.
