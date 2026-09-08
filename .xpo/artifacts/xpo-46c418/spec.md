# Add vitest for frontend unit tests

## What

Set up vitest as the frontend test framework and write initial unit tests for the pure utility functions in the codebase.

## Why

The frontend has zero test coverage. Pure utility functions like diff parsers and file tree builders are easy to test and have caused bugs (xpo-db1b0a). vitest shares Vite's transform pipeline so config is minimal.

## How

### 1. Install & configure vitest

- `bun add -D vitest` in `web/`
- Add `"test": "vitest run"` script to `web/package.json`
- vitest picks up `vite.config.ts` automatically — no separate config file needed unless test-specific overrides are required

### 2. Wire into Makefile

- Add a `make frontend-test` target that runs `cd web && bun run test`
- Update `make test` to run `frontend-test` alongside Go tests

### 3. Export utility functions for testability

The target functions are currently defined inside component files as module-level functions but not exported. Export them so tests can import directly:

- `MergeView.tsx`: `parseDiffByFile`, `addLineNumbers`, `buildFileTree`, `flattenSingleChildDirs`, `buildSplitLines`
- `utils/format.ts`: `formatRelativeTime` (check if already exported)

### 4. Write tests

Create test files alongside the source:

- `web/src/components/IssueDetail/MergeView.test.ts`
  - `parseDiffByFile`: empty input, single file, multi-file, filters `.xpo/` paths
  - `addLineNumbers`: hunk headers, adds, deletes, context lines, meta lines
  - `buildFileTree`: flat files, nested dirs, multiple files in same dir
  - `flattenSingleChildDirs`: single-child chain collapsed, multi-child preserved
  - `buildSplitLines`: matched add/del pairs, unmatched adds, context lines, hunk boundaries
- `web/src/utils/format.test.ts`
  - `formatRelativeTime`: seconds ago, minutes, hours, days, edge cases (future dates, empty input)

## Acceptance Criteria

- [ ] `bun run test` in `web/` runs vitest and all tests pass
- [ ] `make test` runs both Go tests and frontend vitest
- [ ] Tests exist for `parseDiffByFile`, `addLineNumbers`, `buildFileTree`, `flattenSingleChildDirs`, `buildSplitLines`, `formatRelativeTime`
- [ ] No changes to runtime behavior — only exports added and test files created
- [ ] `make lint` passes
