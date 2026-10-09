# Walkthrough: Release prep for 1.3.0

## What was done and why
1.3.0 is a minor release that includes breaking changes: branch mode is removed, `history --json` changed format, and the `init` subcommands are gone. It also changes how `xpo board` is exposed on the network. This issue bumps the version, cuts the CHANGELOG, gives existing users a single README page of upgrade steps, and makes the release tooling fit the xpo worktree workflow.

## The pieces

### 1. `make release` split into `release-prep` and `release-tag` (Makefile)
The old target committed and tagged straight on `main`, which the xpo workflow forbids; the workflow merges through `xpo merge`. It also used `sed -i` with no backup suffix, which fails with macOS (BSD) sed. It is replaced by two targets that share a `release-check-version` guard: `VERSION` must be set and be x.y.z, and the tag must not exist yet.

- **`release-prep VERSION=x.y.z`** runs in the release issue's worktree. It rewrites `CLIVersion` and turns `## [Unreleased]` into `## [Unreleased]` followed by `## [x.y.z] — YYYY-MM-DD`. It uses `perl -pi`, which behaves the same with GNU and BSD tools. `-CSD` makes perl read and write UTF-8, so `\x{2014}` produces the same em-dash bytes (`e2 80 94`) as the existing 1.2.x headers. It doesn't commit; the normal xpo flow commits and merges.
- **`release-tag VERSION=x.y.z`** runs on `main` after the merge. It refuses to run off `main`, with a dirty tree, when `CLIVersion` differs, or when CHANGELOG has no `[x.y.z]` section. It creates the annotated tag and prints the push command, but doesn't push, because publishing is the user's decision.
- The CI release job (`build.yml`) finds the release notes by matching `[ver]` in a `## [` header, whatever separator follows. The em-dash format therefore needs no CI change.

### 2. Version and CHANGELOG
`make release-prep VERSION=1.3.0` produced `CLIVersion = "1.3.0"` and `## [1.3.0] — 2026-10-09`, with an empty `## [Unreleased]` above it. This issue's own entries (the README section and the Makefile split) are in the 1.3.0 section. The xpo-e669d4 fix was merged to `main` while this branch was open, and its entry merges into 1.3.0 → Fixed, as the `git merge-tree` preview confirmed.

### 3. Generated markers
`xpo doctor --fix`, using the 1.3.0 build, refreshed the managed blocks in CLAUDE.md and AGENTS.md and the Claude/Codex/OpenCode skill files. Only the version in each marker changed (`1.2.1` → `1.3.0`). The content hashes are identical, so the templates themselves did not change in this release; the markers simply record which version wrote them. Running these CLI commands was release verification of the new binary, not issue tracking, which is why it doesn't conflict with the "use MCP, not the CLI" rule.

### 4. README
- **"Upgrading from 1.2.x"** (after Installation) has six numbered items. Each says what changed, what to do and which issue it came from:
  1. Finish branch-mode work before upgrading (xpo-863802).
  2. Re-run `xpo init`, which refreshes the managed blocks and skills (xpo-76e2a1, xpo-35fc16, xpo-e88f1b, xpo-c16e28).
  3. `xpo board` binds loopback by default, and a non-loopback `--host` requires the access token (xpo-809ad0, xpo-897f35).
  4. JSON and integration changes: the `history` envelope, `jsonio` outputs, the `{"error"}` envelope, bidirectional links and `unlink`.
  5. Assignees are people, not agents (xpo-35fc16).
  6. The prefix is stored without a trailing dash (xpo-76e2a1).
- **"The Board"** still said the board "has no authentication", which stopped being true with xpo-897f35. It now describes the token.

## What came up along the way
- **The doctor warning became a blocker.** `doctor` reported out-of-order event timestamps. The user wanted it clean before shipping, so it was fixed first in xpo-e669d4, which blocks this issue. Its own merge still ran in the old MCP server and left one bad line. The user installed the fixed binary and restarted the server, then re-timed that line with the repair script; the permission check stopped me from rewriting the shared `issues.db` myself. Once the fixed MCP server was running, `doctor` passed all 14 checks.
- **`xpo --version` doesn't exist.** The CLI has an `xpo version` subcommand, so I corrected the acceptance criterion.

## Verification
At the user's request I tested the exact tree the merge produces: `main` `83448c6` plus this branch, combined with `git merge-tree` without conflicts and checked out in a temporary detached worktree. On that tree I ran what CI runs:
- `make test`: `go vet` including `GOOS=windows`, eslint, vitest (36 files, 540 tests) and `go test ./...`.
- `go test -race -count=1 ./...`.
- The frontend build.
- The six release cross-compiles (linux, darwin and windows on amd64 and arm64) with the embedded frontend.

All passed. The release worktree also passes `make test` on its own.

## Acceptance Criteria
- [x] `CLIVersion` is `1.3.0`, and `xpo version` prints it. Evidence: `Exponential Version: 1.3.0` from the worktree build.
- [x] CHANGELOG has `## [1.3.0] — 2026-10-09` with an empty `## [Unreleased]` above it. Evidence: the em-dash bytes match the 1.2.1 header.
- [x] README has "Upgrading from 1.2.x" covering items 1–6, and the Board section reflects the access token.
- [x] Regenerated markers carry 1.3.0, and `xpo doctor` is fully clean. Evidence: "All 14 checks passed", including "Event timestamps are in file order" and the integrations reported "Up to date".
- [x] `make release-prep` works with BSD tools: 1.3.0 was prepared with it on macOS. `make release-tag` refuses to run off `main`, and the shared guard rejects `1.3` (not x.y.z) and `1.2.1` (tag exists). Evidence: checked by hand in the worktree.
- [x] `make test` passes, on the worktree and on the merged tree, plus race tests, the frontend build and the cross-compiles.
- [ ] Release tag created only after the user confirms. This happens after the merge.
