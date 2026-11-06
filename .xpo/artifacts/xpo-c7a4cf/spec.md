# Spec: Show the build's git version

## What
Every place that shows the running version reports the exact build: `1.5.0-de0d346` between tags, `1.5.0` on a tag, with `!` added for uncommitted changes. The board footer shows only the version, with its usual `v` prefix: `v1.5.0-de0d346`. `CLIVersion` stays as the release version for integration stamps and comparisons.

## Why
`CLIVersion` changes only at release prep, so all builds between releases report the same number. A running board cannot tell whether it includes a given change. This came up during the investigation of xpo-ea5d6b.

## Acceptance Criteria
- [ ] A `make cli` build 28 commits after `v1.5.0` reports `1.5.0-de0d346`. The commit count is not shown.
- [ ] A `make cli` build with HEAD exactly on `v1.5.0` reports `1.5.0`.
- [ ] A `make cli` build with uncommitted changes adds `!`: `1.5.0-de0d346!`, or `1.5.0!` on a tag.
- [ ] A plain `go build` in a git checkout reports the same format, with the base taken from `CLIVersion`.
- [ ] A build with no git data reports `CLIVersion`.
- [ ] `xpo version`, `xpo doctor`, the board footer and the MCP server info report the build version.
- [ ] The board footer shows `v1.5.0-de0d346`, without the word "Exponential". "Disconnected" does not change.
- [ ] Integration markers (`xpo:begin`, `xpo:skill`) still contain plain `x.y.z`. A new commit does not make integrations stale.
- [ ] Binaries from a release tag report exactly `x.y.z`.
- [ ] `make release-prep` and `make release-tag` work as before.
- [ ] `make test` passes.

## Flow
1. In `internal/version/version.go`, add the package variable `gitDescribe`. The linker sets it.
2. Add `formatDescribe(describe string) string`. It converts raw `git describe` output to the display format.
3. Add `formatBuildInfo(release, mainVersion, revision string, modified bool) string`. It builds the fallback from Go's embedded VCS data.
4. Add `Build() string`. It uses `gitDescribe` first, then `debug.ReadBuildInfo`, then `CLIVersion`.
5. Write table tests for steps 2 and 3 before the implementation.
6. In `Makefile`, add `GIT_DESCRIBE`, from `git describe --tags --match 'v[0-9]*' --dirty='!'`.
7. In `Makefile`, pass `GIT_DESCRIBE` to the `cli` target through `-ldflags -X`.
8. In `.github/workflows/build.yml`, build with `make cli BINARY_NAME=xpo${EXT}`. The Makefile then is the only place that defines the flags.
9. Replace `CLIVersion` with `Build()` in `cmd/exponential/version.go`, `cmd/exponential/doctor.go:74`, `internal/server/handlers.go:323` and `internal/mcpserver/server.go:40`.
10. In `web/src/components/Layout/Layout.tsx`, remove the word "Exponential" from the footer.
11. Add a `CHANGELOG.md` entry at commit time.

## Decisions
1. **Keep `CLIVersion`, and add `Build()` for display. Do not replace the constant.** Integration markers and `CompareVersions` need plain `x.y.z`. With a build string, each commit would mark integrations stale and rewrite the managed files. `CompareVersions` also reads `0-de0d346` as 0, so `1.5.3-abc` compares as `1.5.0`. The release is also the correct meaning for those stamps, because the templates belong to a release.
2. **`release-prep` and `release-tag` do not change.** This follows from decision 1. `release-tag` still checks that `CLIVersion` matches the tag, and that check protects the integration stamps.
3. **The user-facing `init` and `doctor` integration messages keep `CLIVersion`.** Messages such as "Integrations are up to date (1.5.0)" describe the stamp, not the build. The only `doctor` line that changes is `Version %s`.
4. **Uncommitted changes add `!`, not `-dirty`.** The user chose the shorter marker. A build with uncommitted changes is not the same as the commit it names. That is common in worktrees, and it is the question this issue wants to answer.
5. **The format drops the commit count and the `g`, as the issue asks.** The SHA alone identifies the commit.
6. **The `git describe` output is normalized in Go, not with `sed` in the Makefile.** Go code can have table tests. The Makefile then needs no text processing.
7. **Plain `go build` falls back to `CLIVersion` plus `vcs.revision`.** Go 1.26 embeds this data, and the output has the same format. Go's own `Main.Version` (`v1.5.1-0.20261106165833-de0d3461dbfe`) is not shown, because it invents a next patch version. It is used only to detect a build exactly on the tag: `v1.5.0`, with or without `+dirty`.
8. **The footer drops the word "Exponential".** The user found the footer too long with the build version. The sidebar logo already names the product.

## Edge Cases
- **MEDIUM: a build after the `release-prep` merge and before the tag.** `make cli` reports `1.5.0-<sha>` while `CLIVersion` is `1.6.0`. This is correct: the last tag is still 1.5.0. A plain `go build` reports `1.6.0-<sha>` in this short window. I accept this difference.
- **MEDIUM: a build from a checkout with no tags.** `git describe` fails and `GIT_DESCRIBE` is empty. `Build()` uses the fallback.
- **MEDIUM: the release workflow.** `actions/checkout` at a tag ref fetches that tag, so `git describe` returns the tag with the default fetch depth. Windows and macOS targets still build on Linux, because `make cli` keeps `GOOS` and `GOARCH` from the environment.
- **LOW: a SHA longer than 7 characters.** `git describe` can extend the SHA to keep it unique. `Build()` keeps what Git returned.
- **LOW: a false `!` from generated files.** `internal/server/static` and the `xpo` binary are git-ignored, so `make build` does not add `!`.
- **LOW: tests.** Test binaries have no linker value and no VCS data, so `Build()` returns `CLIVersion`.
- **LOW: the footer before the config loads.** The version is empty, so the footer shows only the connection dot until the config arrives.

## Assumptions
- The footer keeps its `v` prefix, and the version value has no `v`, as it does today.
- `xpo version` keeps the current line format: `Exponential Version: 1.5.0-de0d346`.
- The `/api/config` `version` field carries the build version. The only reader in the frontend is the footer.

## Open Questions
None.
