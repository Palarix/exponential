# Walkthrough: The Dependencies search box doesn't trim spaces

## What changed and why
The Dependencies table filtered on the raw search string. A pasted id with a trailing space, such as `xpo-2fc583 `, missed rows it should have matched. A query of only spaces counted as a real filter and hid nearly every row. Backlog, MyIssues and LabelPicker already trim their queries, so this view was the odd one out.

## How
- `web/src/components/Dependencies/useDepGraph.ts`: new pure helper `filterEdgesBySearch(edges, issueMap, search)`. It computes `search.trim().toLowerCase()`, returns the edges unchanged when the result is empty, and otherwise keeps an edge when its source id, target id, source title or target title contains the query. It sits next to `collectEdges` and `isResolved`, the other pure edge helpers the table uses.
- `web/src/components/Dependencies/Dependencies.tsx`: the `rows` memo in `TableView` still applies the completed and kind filters first, then returns `filterEdgesBySearch(filtered, issueMap, search)`. The inline filter block is gone.

## Decisions
- **A pure helper instead of an inline fix:** the issue asked for a unit test. Pulling the logic into a function makes it testable without rendering the view, and keeps `TableView` shorter.
- The title checks use `!!` because the helper has a `boolean` return type, and an issue missing from the map would otherwise give `undefined`.

## Acceptance criteria
- [x] A query with leading or trailing whitespace matches the same rows as the trimmed query. Evidence: the test `ignores leading and trailing whitespace` in `useDepGraph.test.ts` covers spaces, tabs and newlines.
- [x] A query of only whitespace shows all rows, the same as an empty search. Evidence: the test `treats an empty or whitespace-only query as no filter`.
- [x] A unit test covers both cases. Evidence: the `filterEdgesBySearch` describe block, three tests; `make test` passes.
