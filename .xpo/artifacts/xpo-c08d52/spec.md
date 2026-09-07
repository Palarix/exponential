# Spec: Fix all frontend eslint errors

## What
Fix 63 lint issues (47 errors, 16 warnings) across ~21 files in `web/src/`.

## Why
The pre-commit hook runs `bun run lint` — these errors block any frontend commit.

## How
Fix each error at the source — no `eslint-disable` comments. Categories:
- **`no-empty`** — empty catch blocks: add meaningful error handling or explicit ignore comment
- **`react-hooks/exhaustive-deps`** — missing or unnecessary deps in useEffect/useMemo/useCallback
- **`react-hooks/preserve-manual-memoization`** — React Compiler can't preserve manual useCallback/useMemo; fix deps
- **`@typescript-eslint/no-unused-vars`** — remove unused variables/params
- **`@typescript-eslint/no-explicit-any`** — replace `any` with proper types
- **`react-refresh/only-export-components`** — move non-component exports to separate files
- **React Compiler errors** — `Cannot access refs during render`, `Cannot reassign variable after render`, `setState synchronously within effect`

## Acceptance Criteria
- `cd web && bun run lint` exits 0
- No `eslint-disable` comments added
- No functional regressions in the frontend