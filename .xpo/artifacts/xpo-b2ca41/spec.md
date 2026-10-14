# Upgrade all frontend and Go dependencies

## What
Bring `web/` npm packages and Go modules up to date, and clean up the stale `web/package-lock.json`.

## Why
lucide-react sat 40 minors behind until xpo-855e28. Other packages have drifted too, including several major versions of the build toolchain.

## Frontend

**Minor/patch** (all taken): react/react-dom 19.3, recharts 3.10, @radix-ui/react-tooltip 1.4, tiptap 3.32, tailwind-merge 3.7, @types/*, typescript-eslint 8.71, vitest 5.0.4.

**Major**
| Package | Was | Now | Notes |
|---|---|---|---|
| vite | 7.3 | 8.3 | Rolldown bundler. `build.rollupOptions` → `build.rolldownOptions`. The object form of `manualChunks` is gone, so the four vendor chunks are now `codeSplitting.groups` with path regexes |
| @vitejs/plugin-react | 5.1 | 6.1 | Requires vite 8 |
| eslint + @eslint/js | 9.39 | 10.x | No config changes needed |
| globals | 16 | 17 | |
| eslint-plugin-react-refresh | 0.4 | 0.5 | |
| @types/node | 24 | 26 | |
| fractional-indexing | 3.2 | 4.0 | The only breaking change: passing `digits` explicitly now makes keys self-headed. We never pass `digits`, so keys keep the classic `a0` format the Go `sortorder` package uses |
| typescript | 5.9 | **5.9 (kept)** | typescript-eslint's peer range is `<6.1.0`. TS 7 is the native port |
| eslint-plugin-react-hooks | 7.0.1 | **~7.0.1 (kept)** | 7.1 adds 20 errors (refs read/written during render, setState in effects) across Backlog, CommandPalette, Menu, App, and keyboard hooks. Follow-up: xpo-09fa6f |

**Lockfile:** `bun update` kept stale nested `prosemirror-model@1.25.7` entries next to the hoisted 1.25.12. TypeScript then saw two unrelated `Schema` types in `MarkdownEditor.tsx`. `bun.lock` is regenerated from scratch to get a single copy.

## Go
`go get -u ./...` + `go mod tidy`. Notable bumps: glamour v0.10 → v1.0.0 (API we use is unchanged; rendered text is identical, only the ANSI encoding differs), go-sdk 1.6 → 1.8, golang.org/x/*.

The `go` directive moves 1.25.8 → **1.26.0**, because every current `golang.org/x/*` module and bubbletea 2.1 require it. CI reads `go.mod`. README ("Requires Go 1.26+") and Dockerfile (`golang:1.26`) are updated to match.

## Decisions
- **package-lock.json: deleted.** Nothing (Makefile, CI) uses npm. bun.lock is the lockfile.
- If a major upgrade breaks something and the fix isn't small, pin that package to its current major and file a follow-up issue rather than expanding scope. Applied to eslint-plugin-react-hooks.

## Acceptance criteria
- [ ] Frontend dependencies are on their latest compatible versions; `bun.lock` updated.
- [ ] `web/package-lock.json` deleted.
- [ ] Go modules upgraded and tidied.
- [ ] `make test`, `bunx tsc -b`, and `make build` pass.
- [ ] The board UI is checked in a browser for regressions (icons, editor, charts, drag and drop).
- [ ] CHANGELOG entry referencing xpo-b2ca41.
