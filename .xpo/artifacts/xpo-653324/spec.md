# Spec: Expand vitest coverage to remaining utilities

## What

Add vitest tests for all pure utility functions not yet covered by the existing test suite. This brings coverage to `labels.ts` (7 functions), `sort.ts` (3 functions), `constants.ts` (2 functions), `filters.ts` (1 function), and the 4 exported pure functions in `useDepGraph.ts` (`resolveIssue`, `collectEdges`, `isResolved`, `computeStats`).

## Why

The initial vitest setup (xpo-46c418) proved the framework with 72 tests across `diff-utils.ts` and `format.ts`. The remaining pure utility modules contain non-trivial logic — case-insensitive label operations, WCAG luminance calculations, dependency graph traversal, multi-key sorting — that is likely to regress during refactors.

## Acceptance Criteria

- [ ] `web/src/utils/labels.test.ts` — tests for all 7 exported functions
- [ ] `web/src/utils/sort.test.ts` — tests for `computeAppendKey`, `sortIssuesWithinGroups`, `sortGroup`
- [ ] `web/src/constants.test.ts` — tests for `isTerminal`, `isCompleted`
- [ ] `web/src/components/Backlog/filters.test.ts` — tests for `hasActiveFilters`
- [ ] `web/src/components/Dependencies/useDepGraph.test.ts` — tests for `resolveIssue`, `collectEdges`, `isResolved`, `computeStats`
- [ ] `make test` passes (lint + all vitest + Go tests)
- [ ] No new dev dependencies required (vitest already installed)

## Flow

1. Create a shared test helper `web/src/test-utils.ts` with a `makeIssue(overrides)` factory that returns a valid `Issue` stub with sensible defaults — avoids repeating boilerplate across sort/depGraph/filters tests.
2. Write `labels.test.ts`:
   - `canonicalLabel`: match found (case-insensitive), no match returns input
   - `labelColor`: match returns color, no match returns CSS var fallback
   - `toggleLabel`: add new label, remove existing (case-insensitive), empty array
   - `deduplicateLabels`: removes dupes, canonicalizes casing
   - `mergeAndSort`: deduplicates + sorts locale-insensitive
   - `splitLabels`: splits into primary/metadata, preserves ordering
   - `contrastTextColor`: dark background → white, light background → black, edge cases (pure black, pure white, mid-gray)
3. Write `sort.test.ts`:
   - `computeAppendKey`: empty array, single issue, multiple issues with varying sort_order
   - `sortIssuesWithinGroups`: manual mode (status groups then sort_order), each non-manual sort key (priority, created, updated, title, estimate), DONE group always sorted by updated
   - `sortGroup`: manual vs non-manual, DONE items sorted by updated
4. Write `constants.test.ts`:
   - `isTerminal`: each of DONE/CANCELED/DUPLICATE → true, BACKLOG/PLANNED/DOING/BLOCKED → false
   - `isCompleted`: DONE → true, everything else → false
5. Write `filters.test.ts`:
   - `hasActiveFilters`: empty filters → false, each individual filter field set → true, epicId set → true
6. Write `useDepGraph.test.ts`:
   - `resolveIssue`: exact match, suffix match, no match
   - `collectEdges`: no deps, simple forward edges, inverse kind canonicalization (blocked_by→blocks, dependency_of→depends_on), deduplication
   - `isResolved`: blocks (source DONE), depends_on (target DONE), other kinds (both DONE), missing issues
   - `computeStats`: mixed resolved/unresolved blocker edges, non-blocker edges excluded from count

## Decisions

- **Shared test helper over inline stubs:** A `makeIssue()` factory keeps tests readable and avoids 20+ repetitions of the full Issue shape. Placed in `src/test-utils.ts` (not `__tests__/`) so any test file can import it.
- **Skip `keyboard.ts` and `backlogCollision.ts`:** Both require DOM or library mocks (jsdom for keyboard, dnd-kit for collision). The issue description explicitly scopes out component rendering tests; these fall in the same "needs environment" category. They can be covered in a future issue with happy-dom/jsdom setup.
- **Test the exported depGraph functions, not the private ones:** `findConnectedComponent`, `detectCycles`, `computeCriticalPath` are private to the module. Testing them indirectly through the hook would require React test harness + dagre. The 4 exported pure functions cover the graph-logic surface area that matters most.

## Assumptions

- The existing vitest config (implicit via `vite.config.ts`) handles `.ts` imports without additional configuration.
- `fractional-indexing` (used by `computeAppendKey`) is deterministic and needs no mocking.
