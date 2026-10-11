# Spec: Dependencies search box trims whitespace

## What
The Dependencies table search trims the query before filtering. A query that is only whitespace counts as an empty search.

## Why
Pasting `xpo-2fc583 ` (with a trailing space) misses rows, and a query of only spaces hides almost every row. Backlog, MyIssues and LabelPicker already trim, so this view should do the same.

## How
In the `TableView` `rows` memo (`web/src/components/Dependencies/Dependencies.tsx`), compute `const q = search.trim().toLowerCase()` and filter only when `q` is non-empty. This matches `Backlog.tsx`.

## Acceptance criteria
- [ ] A query with leading or trailing whitespace matches the same rows as the trimmed query
- [ ] A query of only whitespace shows all rows, the same as an empty search
- [ ] A unit test covers both cases
