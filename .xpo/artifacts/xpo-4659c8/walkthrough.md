# Walkthrough: Release prep for 1.4.0

## Version choice
You asked for 1.3.1. Unreleased contains new features (assignee picker buttons on backlog rows and board cards, timeline Activity, `link --overwrite`) and a behavior change: `link` now fails on an already-linked pair unless `overwrite` is set. The CHANGELOG claims SemVer, which calls for a minor release here, so you chose 1.4.0.

## What changed
- **Version bump:** `make release-prep VERSION=1.4.0` in the worktree set `CLIVersion = "1.4.0"` and turned `## [Unreleased]` into `## [1.4.0] — 2026-10-12`, with a new empty Unreleased section above it. This is the same target and em-dash format as 1.3.0.
- **Markers:** `make cli` + `./xpo doctor --fix` refreshed the `xpo:begin` / `xpo:skill` markers to 1.4.0 in CLAUDE.md, AGENTS.md and the Claude/Codex/OpenCode skill files.
  - The skill template changed since v1.3.0: xpo-c93b11 added the one-link-per-pair guidance. So each `SKILL.md` also gained those two lines, and its template hash changed.
  - `doctor --fix` also listed `.gitignore`, `.mcp.json`, `.codex/config.toml`, `opencode.json` and `.gitattributes` as changed, but they were rewritten byte-identical, and `git status` shows no diff for them.
- **README:** a new "Upgrading from 1.3.x" section sits above the 1.2.x one. It covers the one-link-per-pair rule, including `--overwrite`, the `add`/`update` duplicate-target check and that existing multi-link pairs keep working. It also says to run `xpo init` or `xpo doctor --fix` to pick up the updated skill text.
- **CHANGELOG:** the entry for this issue is in 1.4.0 → Added.

## Verification
- `xpo version` prints `Exponential Version: 1.4.0`.
- `xpo doctor`: all 14 checks passed, integrations up to date.
- `make test`: exit 0, which covers go vet, eslint, 638 vitest tests and Go tests.
- `go test -race -count=1 ./...`: exit 0, 15 packages ok, no data races.
- `make frontend`: exit 0.
- Cross-compiles of `./cmd/...` succeed for linux, darwin and windows on amd64 and arm64.

## Release steps after merge
1. On main: `make release-tag VERSION=1.4.0` creates the annotated tag `v1.4.0` locally.
2. After your confirmation: `git push origin main v1.4.0`. The tag push triggers the goreleaser jobs in `.github/workflows/build.yml`.

## Acceptance criteria
- [x] `CLIVersion` is `1.4.0`, and `xpo version` prints it.
- [x] CHANGELOG has `## [1.4.0] — 2026-10-12` with an empty Unreleased section above it.
- [x] README has an "Upgrading from 1.3.x" section.
- [x] Markers carry 1.4.0, and `xpo doctor` is clean: all 14 checks passed.
- [x] `make test` passes, plus the race tests, the frontend build and the cross-compiles.
- [ ] Tag `v1.4.0` is created on main after merge and pushed only after your confirmation. This is done after merge; see the issue comments.
