# Walkthrough: Upgrade all frontend and Go dependencies

## What changed and why

Dependencies had drifted: lucide-react was 40 minors behind, and the build toolchain was a major version behind. This brings `web/` and `go.mod` up to date. Two packages stay on their current major version on purpose.

| File | Change |
|---|---|
| `web/package.json`, `web/bun.lock` | All dependencies to latest, except TypeScript (`~5.9.3`) and eslint-plugin-react-hooks (`~7.0.1`) |
| `web/package-lock.json` | Deleted |
| `web/vite.config.ts` | `rollupOptions.output.manualChunks` → `rolldownOptions.output.codeSplitting.groups` |
| `go.mod`, `go.sum` | `go get -u ./...` + `go mod tidy`; `go` directive 1.25.8 → 1.26.0 |
| `README.md`, `Dockerfile` | "Requires Go 1.26+", `golang:1.26` builder |
| `CHANGELOG.md` | Entry under Changed |

## How the pieces fit

### Vite 8 and Rolldown
Vite 8 replaces Rollup with Rolldown. Rolldown accepts `manualChunks` only as a function; the object form (`{ 'vendor-x': ['pkg', …] }`) is gone. Its replacement is `codeSplitting.groups`: each group has a `name` and a `test` regex on the module path.

The old object form pulled each listed package *and its dependencies* into the chunk. A regex only matches what it names, so each group also lists the main transitive package families:
- **markdown:** `remark-`, `micromark`, `mdast-`, `hast-`, `unified`, `unist-`
- **editor:** `prosemirror-`, `markdown-it`
- **charts:** `d3-`, `victory-vendor`

The regexes use `[\\/]` so they also match Windows paths. The resulting chunks match the old build closely. The >500 kB warning comes from the Dependencies chunk (elkjs, 1.46 MB) and was already there before.

### The duplicate prosemirror-model
After `bun update`, `bunx tsc -b` failed in `MarkdownEditor.tsx`: two `Schema` types "with this name exist, but they are unrelated". `bun update` hoisted `prosemirror-model@1.25.12` but kept old nested `prosemirror-state/prosemirror-model@1.25.7` entries in `bun.lock`. A clean `bun install` from the same lockfile reproduced them. The only fix was deleting `bun.lock` and resolving from scratch, which leaves a single copy. If tiptap types break after a future `bun update`, check `grep '/prosemirror-model"' bun.lock` first.

### fractional-indexing 4
The one breaking change: when you pass `digits` explicitly, `intDigits` now defaults to `digits`, so keys become "self-headed" (`"50"` rather than `"a0"`). We call `generateKeyBetween(a, b)` without `digits` (Board, Backlog, `utils/sort.ts`), so keys keep the classic `a0` form. That matters because the Go `internal/sortorder` package produces and compares the same keys.

### The two packages that stay behind
- **TypeScript 5.9:** typescript-eslint 8.71 declares `typescript: >=4.8.4 <6.1.0`. TS 7 is the native Go port, which typescript-eslint can't use yet.
- **eslint-plugin-react-hooks ~7.0.1:** 7.1 adds 20 errors: ref writes and reads during render, setState in effects, and a recursive `useCallback`. They are spread across Backlog, CommandPalette, Menu, App, and `keyboard/hooks.ts`. Fixing them reshapes keyboard and drag-and-drop code, which is more than an upgrade issue should carry. Per the spec's rule (pin and file a follow-up when the fix isn't small), that work is xpo-09fa6f, with the list of errors.

### Go
glamour went v0.10 → v1.0.0. We only use `NewTermRenderer`, `WithAutoStyle`, and `WithWordWrap`, and all three still exist. I rendered the same markdown with both versions (heading, bold, code, link, task list, table, code block). Without ANSI codes the text is byte-identical; only the escape-code encoding differs.

The `go` directive moved to 1.26.0 because every current `golang.org/x/*` module (and bubbletea 2.1, ultraviolet) declares `go 1.26.0`. CI reads `go-version-file: go.mod`, so it follows automatically. The README and Dockerfile were the hand-maintained copies.

### package-lock.json
Nothing used npm: the Makefile runs `bun install`, and CI uses bun. The file had last changed in early commits and was stale, so it's deleted.

## Acceptance criteria
- [x] Frontend dependencies are on their latest compatible versions; `bun.lock` updated. `bun outdated` lists only typescript (kept on purpose) after the upgrade. eslint-plugin-react-hooks is held at ~7.0.1 → xpo-09fa6f.
- [x] `web/package-lock.json` deleted.
- [x] Go modules upgraded and tidied: `go get -u ./...`, `go mod tidy`, `go build ./...`, and `go vet ./...` are clean.
- [x] `make test`, `bunx tsc -b`, and `make build` pass. `make test` exits 0 (lint, 938 frontend tests in 68 files, Go tests). `make build` produced the 38 MB `xpo` binary.
- [x] The board UI is checked for regressions. Headless Chromium compared the DOM of main's frontend (old lockfile, :5174) and this branch (:5173) against the same :8080 API, on 14 routes. Results:
  - Identical on the board, backlog, dependencies, and the rich issue details. On the other routes only the ticking "Nm ago" timestamps differ.
  - On the Dashboard, recharts changed its internal markup (attribute order, an extra `<g>`). All 237 SVG shapes have identical geometry, and the screenshot is pixel-identical.
  - Icons were covered by the lucide SVG path comparison. The user also checked the board by hand.
  - Interactions (typing, drag and drop) were not automated.
- [x] CHANGELOG entry referencing xpo-b2ca41.

## Found along the way
- xpo-66c6a4: on the Dashboard, the Active Epics rows put a `CopyableId` `<button>` inside the row `<button>`. Main has the same bug.
- xpo-09fa6f: eslint-plugin-react-hooks 7.1 errors.
