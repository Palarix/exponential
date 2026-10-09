# Spec: Release prep for 1.3.0

## What
Bump to 1.3.0, cut the CHANGELOG section, add an "Upgrading from 1.2.x" section to the README, and refresh the generated markers. Also make `make release` fit the worktree workflow, so this release and future ones use the same path.

## Why
Unreleased has collected breaking changes (branch mode removed, `history --json` format, init subcommands removed) and a network-binding change. Existing users need one page of migration steps. The current `make release` commits and tags on `main` and uses GNU-only `sed -i`, so it can't be used as-is.

## How

### 1. Split `make release` into two portable targets (Makefile)
- **`make release-prep VERSION=x.y.z`** runs in the issue worktree:
  - Guards: `VERSION` is set and valid semver, the tag doesn't exist yet, and `## [Unreleased]` is present.
  - Rewrites `CLIVersion` in `internal/version/version.go`.
  - Changes `## [Unreleased]` to `## [Unreleased]\n\n## [x.y.z] — YYYY-MM-DD`, using the em-dash format of 1.2.x.
  - Uses `perl -pi -e` instead of `sed -i`, because perl behaves the same on macOS and Linux.
  - Does not commit or tag; the xpo workflow commits.
- **`make release-tag VERSION=x.y.z`** runs on `main` after `merge`:
  - Guards: the current branch is `main`, the tree is clean, the tag doesn't exist, `CLIVersion == VERSION`, and CHANGELOG has `## [VERSION]`.
  - Creates the annotated tag `vVERSION` and prints `git push origin main vVERSION`. It does not push.
- The old `release` target is removed. Both targets go in `.PHONY`.
- The CI changelog extractor (`build.yml`) matches `[ver]` whatever separator follows, so the em-dash header is fine.

### 2. Version bump
Run `make release-prep VERSION=1.3.0` in the worktree. The date is 2026-10-09, or the merge date if that turns out to be later.

### 3. Regenerate markers
- Build the binary in the worktree.
- Run the build's `xpo doctor` to see which managed blocks and skills are stale: `CLAUDE.md`, `AGENTS.md`, and `.claude/`, `.codex/` and `.opencode/skills/xpo/SKILL.md`.
- Refresh them with `xpo init --yes` (or `doctor --fix`), then re-run `doctor` until it is clean.
- These are release-verification runs of the newly built binary, not issue tracking, so CLAUDE.md's "never shell out to the xpo CLI" rule doesn't cover them.
- Only template-driven files may change. If `doctor --fix` wants to touch `.xpo/` state, stop and report.
- The legacy `claude/skills/` folder (no dot, untouched since the rebrand) is out of scope.

### 4. README "Upgrading from 1.2.x"
This goes after "Installation" as a `##` section. Each item is short and links its issue ID:
1. **Branch mode removed** (xpo-863802): `start --mode`, `merge --no-wt` and the MCP/JSON `mode` field are gone, and `worktrees: false` is ignored with a warning. *Before upgrading:* merge or park branch-mode issues and put the hub back on the default branch.
2. **Re-run `xpo init`** in every project. The unified wizard replaces `xpo init mcp` / `xpo init skill` (xpo-76e2a1). It refreshes the managed instruction blocks and skill files, whose templates changed (agent identity xpo-35fc16, branch mode xpo-863802, merge/commit gate xpo-e88f1b, xpo-c16e28). Afterwards `xpo doctor` should be clean.
3. **`xpo board` network access:** it binds loopback by default (xpo-809ad0). A non-loopback `--host` now requires the printed access token (xpo-897f35). `ssh -L` is still the recommended way to reach a board on another machine.
4. **JSON / integration changes:**
   - `history --json` is a single envelope, not JSONL (xpo-f56ce4).
   - `add`/`update`/`rationale --json` emit `jsonio` envelopes instead of plain text, and errors use a uniform `{"error": …}` (xpo-3847a5, xpo-199282, xpo-11e559, xpo-d1a06b).
   - Links are bidirectional, `show` adds `derived: true` inverse links, `update { links }` replaces the full set, and the new `unlink` tool exists (xpo-9d6609, xpo-9d5d00).
5. **Assignees are people, not agents** (xpo-35fc16): old agent assignments display as the person the agent worked for, with no data migration. `start` assigns the issue to the principal.
6. **Prefix stored without a trailing dash** (xpo-76e2a1): configs are normalised on load, so no action is needed. This is mentioned so nobody is surprised by the config diff.

### 4b. README "The Board" section
The Board section still says the board "has no authentication" and that anyone who can reach `--host` can modify issues, which stopped being true with xpo-897f35. Update it to describe the access token.

### 5. CHANGELOG
The new section gets `- Split make release into release-prep / release-tag … (xpo-e60c26)` under Changed. It also gets the README upgrade section under Added.

## Acceptance Criteria
- [ ] `CLIVersion` is `1.3.0`, and `xpo version` prints it. (The CLI has no `--version` flag; the issue text was wrong.)
- [ ] CHANGELOG has `## [1.3.0] — <date>` with an empty `## [Unreleased]` above it.
- [ ] README has an "Upgrading from 1.2.x" section covering items 1–6.
- [ ] Regenerated markers carry 1.3.0, and `xpo doctor` is fully clean. The out-of-order timestamps warning was fixed first in xpo-e669d4, which blocks this issue, at the user's request.
- [ ] `make release-prep` works with BSD tools (it is exercised by this release). `make release-tag` refuses to run off `main`, with a dirty tree, or with a version mismatch (checked by hand in the worktree).
- [ ] `make test` passes.
- [ ] After `merge`, `make release-tag VERSION=1.3.0` and the push run only after the user confirms.

## Decisions
1. **The Makefile fix is part of this issue** (confirmed by the user). `release` is split into `release-prep` and `release-tag`, and this release is made with them.
