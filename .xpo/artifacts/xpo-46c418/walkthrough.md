# Add vitest for frontend unit tests

## What was built

Frontend test infrastructure using vitest, with initial coverage of all pure utility functions.

## Setup

- **vitest 5.0.0** installed as a dev dependency — it shares Vite's transform pipeline, so no separate config file is needed.
- `"test": "vitest run"` added to `web/package.json`.
- `frontend-test` Makefile target added and wired into `make test`, so the CI pipeline runs lint → vitest → Go tests in sequence.

## Utility extraction

The target functions (`parseDiffByFile`, `addLineNumbers`, `buildFileTree`, `flattenSingleChildDirs`, `buildSplitLines`) were originally defined as module-level functions inside `MergeView.tsx`. Exporting them triggered `react-refresh/only-export-components` lint errors — React Fast Refresh requires component files to only export components.

**Fix:** extracted all five functions plus their types (`FileTreeNode`, `NumberedLine`, `SplitLine`) into a new `diff-utils.ts` in the same directory. `MergeView.tsx` imports from it. No runtime behavior change — the functions are pure and have no React dependencies.

## Test coverage

### `MergeView.test.ts` (27 tests)

| Function | Tests | Key cases |
|---|---|---|
| `parseDiffByFile` | 4 | empty input, single file, multi-file, `.xpo/` path filtering |
| `addLineNumbers` | 5 | hunk headers, add/del/context numbering, meta lines, multiple hunks |
| `buildFileTree` | 5 | flat files, nested paths, shared directories |
| `flattenSingleChildDirs` | 5 | single-child collapse, multi-child preserved, recursive chains |
| `buildSplitLines` | 8 | matched pairs, unmatched adds, context, hunk boundaries, empty input |

### `format.test.ts` (45 tests)

Covers all 8 exported functions: `shortName`, `displayActor`, `formatRelativeTime`, `formatShortDate`, `formatTriage`, `stripMarkdown`, `linkifyIssueIds`, `formatDuration`.

## Acceptance Criteria

- [x] `bun run test` in `web/` runs vitest and all tests pass — 72 tests green
- [x] `make test` runs both Go tests and frontend vitest — `frontend-test` target wired as prerequisite
- [x] Tests exist for `parseDiffByFile`, `addLineNumbers`, `buildFileTree`, `flattenSingleChildDirs`, `buildSplitLines`, `formatRelativeTime` — plus 7 additional format utilities
- [x] No changes to runtime behavior — only exports added via extraction to `diff-utils.ts`
- [x] `make lint` passes — utility extraction resolved the `react-refresh/only-export-components` errors
