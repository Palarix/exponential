# Release prep for 1.5.0

## What / Why
Cut 1.5.0, a minor release. Unreleased has about 290 lines that were written one issue at a time. The GitHub release body is that section verbatim (`build.yml` extracts `## [1.5.0]`), so it should read as a release note for someone upgrading from 1.4.0, not as a development log.

## How

### 1. Clean the Unreleased section
Rules:
- **Agent chat is new in 1.5.0.** v1.4.0 has no chat code, so every Changed or Fixed entry about agent chat describes a change to something 1.4.0 users never had. Fold each one into the Added entry it changes, describing the final behavior only. Examples: approval cards (xpo-7bc86e + xpo-aaf8b1 + xpo-8579fb), the question card (xpo-447687 + xpo-46afc0 + xpo-9761dd), and chat cost overcounting (xpo-66fce3), which goes into usage (xpo-1446f2). Keep every issue ID on the merged entry.
- **Same for other features new in this release**, such as issue drafts (xpo-758398) and the Merged view (xpo-64be51): fixes to them fold into their Added entry.
- **Drop superseded intermediate states.** Example: xpo-ae6385 added the `chat_changed` SSE event, then xpo-6d1102 removed it. xpo-bbcab4 says the chat list "keeps polling in hidden tabs", and later work stopped the polling. xpo-7e9c4f describes `/stream` + `Last-Event-ID`, which xpo-6d1102 removed. Only the final design is described. The Removed bullet under xpo-6d1102 lists only things that existed in 1.4.0.
- **Remote mode is removed in this release** (xpo-67281a), so remove remote-mode sub-bullets from other entries (xpo-ae6385, xpo-00f0e5, xpo-6d1102) and the `XPO_REMOTE_URL` / `permissions` examples in xpo-e81612.
- **Real changes to 1.4.0 behavior stay in Changed or Fixed**: blockers (xpo-35943d), labels (xpo-e35f16), delete dialog (xpo-33c92b), links color (xpo-1d9d5d), buttons, pills, dependency upgrades / Go 1.26, the activity timeline, MCP `update` null clearing, the 404 fix, and so on.
- **Order and grouping:** within each section, put the Agent chat entries first as a group (the headline feature), then board/web UI, then MCP/CLI/API, then dependencies.
- **Format:** one blank line between every entry, the issue ID in a trailing parenthesis on each entry (the 404 fix, commit 1b35c4b, has no issue and stays without one), and the house writing style. Don't add any new information.

### 2. Version bump
`make release-prep VERSION=1.5.0` in the worktree. It sets `CLIVersion` and turns `## [Unreleased]` into `## [1.5.0] — <date>`, with an empty Unreleased section above it.

### 3. Markers
`make cli`, then `./xpo doctor --fix`, which refreshes the `xpo:begin` / `xpo:skill` markers to 1.5.0. The skill templates changed since v1.4.0 (xpo-ed312f), so the content changes too. `xpo doctor` must be clean.

### 4. README "Upgrading from 1.4.x"
Placed above the 1.3.x section. It covers what can break existing setups:
1. Remote mode is gone: `xpo serve`/`login`/`logout` are removed, `remote.*`, `permissions` and `servers:` are ignored, and the Docker files are gone. Share issues through git.
2. A CANCELED or DUPLICATE `blocked_by` blocker still blocks `start` and drive.
3. Live-update API for integrations: one `/api/events` stream with topics. `GET /api/local/chats/{id}/stream` and the `issue_*` / `draft_changed` SSE events are removed.
4. MCP `update` / `update --json`: `null` now clears `assignee`, `cycle_id`, `priority`, `story_points` and `parent`.
5. Building from source requires Go 1.26.
6. Refresh agent instructions with `xpo init` / `xpo doctor --fix`.

Add a CHANGELOG entry for this issue under Added, as in 1.4.0.

### 5. Tag and GitHub release (after merge)
On main: `make release-tag VERSION=1.5.0`, which creates the signed annotated tag locally. **Ask the user before** `git push origin main v1.5.0`. The tag push runs `build.yml`, which builds 6 binaries plus checksums and creates the GitHub release with the 1.5.0 CHANGELOG section as its body. Then check it with `gh release view v1.5.0` (assets and body).

## Acceptance criteria
- [ ] The 1.5.0 section has no Changed/Fixed entries for agent chat or other features new in 1.5.0, no remote-mode mentions outside Removed, and no descriptions of designs replaced within the release.
- [ ] Every issue ID from the current Unreleased section still appears in 1.5.0.
- [ ] `CLIVersion` is `1.5.0`, and `xpo version` prints it.
- [ ] CHANGELOG has `## [1.5.0] — <date>` with an empty Unreleased section above it.
- [ ] README has an "Upgrading from 1.4.x" section.
- [ ] Markers carry 1.5.0. `xpo doctor` passes, except for the expected "local edits" warning on the Claude skill (see Decisions).
- [ ] `make test` passes.
- [ ] After merge, tag `v1.5.0` is on main and is pushed only after the user confirms. The GitHub release has 6 archives plus `checksums.txt`, and its body is the 1.5.0 section.

## Decisions
- **Keep the ASD-STE100 writing-style experiment.** `doctor --fix` would delete the edits from commit 64ea6a9 in `.claude/skills/xpo/SKILL.md` and `references/spec-guide.md`. Those two files keep their content and their 1.4.0 hashes, with only the marker version bumped to 1.5.0. Apart from the experiment, their content matches the 1.5.0 template. `doctor` warns "skill has local edits" until the experiment is promoted into the template or dropped.
- **Minor, not major.** Remote mode and three commands are removed. That is technically breaking, but 1.3.0 removed branch mode as a minor release too, and you asked for 1.5.0.
- **Rewrite, not just reformat.** Folding the agent-chat history into its Added entries makes the section much shorter. The per-issue detail stays in each issue's spec and walkthrough.
