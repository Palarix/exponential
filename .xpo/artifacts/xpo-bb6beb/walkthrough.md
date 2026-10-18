# Walkthrough: Release prep for 1.5.0

## Version choice
1.5.0 removes remote mode, along with `xpo serve`, `xpo login` and `xpo logout`. That is technically a breaking change. 1.3.0 removed branch mode as a minor release too, so you kept 1.5.0.

## Changelog cleanup
The GitHub release body is the `## [1.5.0]` section verbatim: the "Extract changelog" step in `.github/workflows/build.yml` copies it with awk. The section must therefore read as a release note for 1.4.0 users.

The main finding: v1.4.0 has no agent chat code. Every Changed or Fixed entry about agent chat described a change to something no released version had. The rewrite applies these rules:
- **New features are Added entries that describe the final state.** Agent chat is now 12 Added entries: the chat itself, Chat/Work Mode, approval cards, the question card, the transcript, background tasks, tools, attachments, titles, status, the All chats page, and chats on issues. Their reworks and fixes are folded in, and so are the fixes to issue drafts, the Merged view and the issue ID chips.
- **Replaced designs are gone.** Examples: the `chat_changed` event, the per-chat `/stream` with `Last-Event-ID`, chat-list polling, deleting chats after 30 days, and the 200-chat limit. These are not in the code anymore. `AutoArchiveAfter` in `internal/chat/store.go` confirmed that old chats are archived.
- **Remote mode is mentioned only in Removed.** The remote-mode guards entry (xpo-a60065) moved into the Removed entry. The `XPO_REMOTE_URL` and `permissions` examples left xpo-e81612.
- **Changed and Fixed list only differences from 1.4.0.** One example: the fix for live updates stopping in background tabs (xpo-7e9c4f) affected the 1.4.0 issue stream, so its user-facing part lives in the Changed entry for the one event stream (xpo-6d1102).

Every issue ID from the old section is still there, except two forward references in remote-mode text: xpo-6dd7a0 (canceled) and xpo-8757a9 (the Team Mode epic). The section went from 290 to 196 lines.

## Version and markers
- `make release-prep VERSION=1.5.0` set `CLIVersion` and dated the section.
- `make cli` + `./xpo doctor --fix` refreshed the markers to 1.5.0. The skill content changed since 1.4.0 (xpo-ed312f), and the Codex and OpenCode skills picked that up.
- **Your writing-style experiment:** `doctor --fix` replaced the Claude skill's `SKILL.md` and `references/spec-guide.md` and deleted the ASD-STE100 section from 64ea6a9. I restored both files and bumped only their marker version. They keep their old hashes, so `doctor` still detects the local edits. Apart from the experiment, their content matches the 1.5.0 template. `doctor` warns until the experiment is promoted into the template or dropped.

## README
A new "Upgrading from 1.4.x" section sits above the 1.3.x one. It covers six items:
1. Remote mode removed. It includes a reminder to make sure `.xpo/issues.db` in git has every change before leaving a server.
2. CANCELED and DUPLICATE blockers still block.
3. The one event stream, and the removed `issue_*` events.
4. `null` clears fields in `update`.
5. Go 1.26 is needed to build from source.
6. Refreshing the agent instructions.

## Release steps after merge
1. On main: `make release-tag VERSION=1.5.0` creates the annotated tag `v1.5.0`.
2. `git push origin main v1.5.0` starts `build.yml`, which builds 6 archives and `checksums.txt` and creates the release.

## Acceptance criteria
- [x] No Changed/Fixed entries for features new in 1.5.0, no remote mode outside Removed, no replaced designs. Each entry was checked by hand against the old section.
- [x] Every issue ID is kept. `comm` of the before and after ID lists finds only xpo-6dd7a0 and xpo-8757a9 missing (forward references), and xpo-bb6beb is new.
- [x] `CLIVersion` is `1.5.0`. `xpo version` prints `Exponential Version: 1.5.0`.
- [x] CHANGELOG has `## [1.5.0] — 2026-10-18` with an empty Unreleased section above it.
- [x] README has an "Upgrading from 1.4.x" section.
- [x] Markers carry 1.5.0. `xpo doctor`: 13 passed, 1 warning (the expected local edits).
- [x] `make test` exits 0: lint, 1225 vitest tests, Go tests.
- [ ] Tag `v1.5.0` is on main, and the GitHub release has 6 archives plus `checksums.txt`. Done after merge.
