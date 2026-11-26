# Walkthrough: Release prep for 1.6.0

## Summary
This change prepares the 1.6.0 release. The Unreleased CHANGELOG section is now a release note for 1.5.0 users. `CLIVersion` is `1.6.0`, and the agent-instruction markers carry 1.6.0. The tag and the GitHub release follow after the merge.

## Changes

### CHANGELOG.md
A script rebuilt the Unreleased section from its own entries. The script asserted that the set of issue IDs did not change, and it stopped if any lookup matched zero entries or more than one.

- **Folds:** four Fixed entries moved into the entries they fix, because their features are new in 1.6.0.
  - xpo-397365, xpo-ea5d6b and xpo-7ffcf5 (thread follow) moved into the glide entry xpo-50b9d9. v1.5.0 has no Motion preference and no glide.
  - xpo-ea2c99 (TipTap crash) moved into xpo-c35d83. xpo-c35d83 made the chat composer a TipTap editor, so the crash cannot occur in 1.5.0. Its focus and placeholder behavior is now one sentence in the Composer sub-bullet.
  - xpo-8fc8ba (a test fixture broke `make frontend` during this release) moved into the `make frontend` entry xpo-a67011.
- **Migration:** the Milestones sub-bullet is now one sentence. Releases never shipped, so only boards that ran a development build have Releases data.
- **Order:** Added starts with Milestones and the factory workflow. Agent chat, board UI and CLI/MCP/API follow. Changed keeps the BREAKING workflow entry first. Fixed groups agent chat, board UI, CLI and tooling.
- **Format:** every entry has one blank line before it. Five Fixed entries had no blank line before them.
- `make release-prep VERSION=1.6.0` turned the section into `## [1.6.0] — 2026-11-26`, with an empty Unreleased section above it.

### internal/version/version.go
`make release-prep` set `CLIVersion = "1.6.0"`.

### Agent instructions
`./xpo doctor --fix` changed the marker line in 11 files: `CLAUDE.md`, `AGENTS.md`, and `SKILL.md`, `references/mcp-tools.md` and `references/spec-guide.md` for Claude Code, Codex and OpenCode. Each marker changed from 1.5.0 to 1.6.0. The content hashes did not change, so no content changed.

### Board
The 1.6.0 milestone goal no longer has "plus privacy-respecting telemetry". This release has no telemetry issue.

## Decisions
- **README unchanged.** "Upgrading from 1.5.x" covers the workflow split (xpo-f19bfa). No other 1.6.0 change breaks an existing setup.
- **No CHANGELOG entry for this issue.** The README did not change.
- **Caret sub-bullet dropped.** xpo-ea2c99 also guarded the caret placement in issue descriptions. That guard is a precaution with no visible change for 1.5.0 users, so the entry does not mention it.

## Acceptance criteria
- [x] The 1.6.0 section has no Changed or Fixed entries for features new in 1.6.0. Evidence: the Motion, glide, composer and fixture fixes are folded. The Filter menu, submenus and agent chat exist in v1.5.0, so their fixes stay.
- [x] The 1.6.0 section has no descriptions of designs replaced within the release. Evidence: the Releases migration text is one sentence.
- [x] Every issue ID from the current Unreleased section appears in 1.6.0. Evidence: the script asserted equal ID sets (47 IDs).
- [x] Every entry has one blank line before it. Evidence: an awk check found no entry without one.
- [x] `CLIVersion` is `1.6.0`. `xpo version` prints `1.6.0` once the `v1.6.0` tag exists. Before the tag, it prints `1.5.0-<sha>`, because the version comes from `git describe` (xpo-c7a4cf).
- [x] CHANGELOG has `## [1.6.0] — 2026-11-26` with an empty Unreleased section above it.
- [x] Markers carry 1.6.0, and `xpo doctor` passes with no warnings.
- [x] `make test` passes. Evidence: exit 0, vitest 1777/1777 in 122 files, Go tests and lint clean.
- [ ] After merge, tag `v1.6.0` is on main. The tag is pushed only after the user confirms. (after merge)
- [ ] The GitHub release has 6 archives and `checksums.txt`, and its body is the 1.6.0 section. (after push) The CI extraction step, run locally, gives the 1.6.0 section (23 KB, 122 lines).
- [ ] The 1.6.0 milestone goal has no telemetry phrase, and the milestone is completed. The goal is updated. The milestone is completed after the release.
