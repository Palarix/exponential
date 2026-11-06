# Walkthrough: Show the build's git version

## What was built
A running xpo now reports the exact build. The value is `1.5.0-de0d346` between tags and `1.5.0` on a tag. A trailing `!` marks uncommitted changes. Before this change, all builds between releases reported the same number, so a running board could not tell whether it included a given change.

## How the pieces fit together

### Two version values (`internal/version/version.go`)
- `CLIVersion` stays the release version, always plain `x.y.z`.
- `Build()` is new and returns the display value.

`Build()` tries three sources in order:
1. `gitDescribe`, a package variable that the linker sets.
2. Go's embedded VCS data, from `debug.ReadBuildInfo`, for a plain `go build`.
3. `CLIVersion`, when no git data exists. Test binaries use this path.

### The build (`Makefile`, `.github/workflows/build.yml`)
`make cli` passes raw `git describe --tags --match 'v[0-9]*' --dirty='!'` output through `-ldflags -X`. `make`, `make build` and `make install` all use the `cli` target. `build.yml` now calls `make cli` too, so the Makefile is the only place that defines the flags. Without this change, released binaries would report `1.5.0-<sha>`.

### Readers
`xpo version`, the `xpo doctor` version line, `/api/config` and the MCP server info call `Build()`. The board footer reads `/api/config`. It now shows only `v1.5.0-de0d346`, because the word "Exponential" made it too long. The sidebar logo already names the product.

## Key decisions
- **`CLIVersion` stays.** The issue asked whether the constant had to go. Integration markers and `CompareVersions` need plain `x.y.z`. A build string as the stamp would mark integrations stale at each commit. `CompareVersions` would also read `1.5.3-abc` as `1.5.0`. So `make release-prep` and `make release-tag` did not change.
- **`init` and `doctor` integration messages keep `CLIVersion`.** "Integrations are up to date (1.5.0)" describes the stamp, not the build.
- **`!`, not `-dirty`.** The user chose the shorter marker during review.
- **Normalization is in Go, not `sed`.** `formatDescribe` removes the `v`, the commit count and the `g`. It has table tests, and the Makefile needs no text processing.

## Non-obvious details
- Go's `Main.Version` looks useful, but past a tag it is a pseudo-version of the next patch: `v1.5.1-0.20261106165833-de0d3461dbfe`. `formatBuildInfo` uses it only to detect a build exactly on the release tag.
- After the `release-prep` merge and before the tag, `make cli` reports `1.5.0-<sha>` and a plain `go build` reports `1.6.0-<sha>`. The Makefile uses the last tag, and the fallback uses `CLIVersion`. This window is short, so the difference is accepted.
- `git describe` runs when `make` runs the recipe. `make install` rebuilds the binary, so it reads the git state at install time.

## Acceptance criteria
- [x] A `make cli` build past the tag reports the version without the commit count. Evidence: `1.5.0-3bb282a` in a temporary clone. `TestFormatDescribe` covers `v1.5.0-28-gde0d346`.
- [x] A `make cli` build exactly on the tag reports `1.5.0`. Evidence: temporary clone with a local tag.
- [x] A `make cli` build with uncommitted changes adds `!`. Evidence: `1.5.0-de0d346!` in this worktree, and `1.5.0!` on the tag in the clone.
- [x] A plain `go build` reports the same format. Evidence: `1.5.0-de0d346!` in this worktree, and `1.5.0` on the tag in the clone.
- [x] A build with no git data reports `CLIVersion`. Evidence: `TestBuildWithoutGitDataIsCLIVersion` and the `no git data` case in `TestFormatBuildInfo`.
- [x] `xpo version`, `xpo doctor`, the board footer and the MCP server info report the build version. Evidence: `xpo version` and `xpo doctor` output, and the user confirmed the footer. The MCP server info was checked by code only: it calls `version.Build()`.
- [x] The footer shows the version without "Exponential", and "Disconnected" does not change. Evidence: `Layout.test.tsx` and the user's check of the footer.
- [x] Integration markers contain plain `x.y.z`, and a new commit does not make integrations stale. Evidence: the code that writes the markers still uses `CLIVersion`, and the existing integration tests pass.
- [x] Binaries from a release tag report exactly `x.y.z`. Evidence: a shallow tag clone reported `1.5.0` with the workflow's flags, and Linux and Windows cross-builds linked `gitDescribe=v1.5.0`. `build.yml` has not run in GitHub Actions. The first real check is the next tag push.
- [x] `make release-prep` and `make release-tag` work as before. Evidence: this change does not edit these targets or `CLIVersion`. They were not run.
- [x] `make test` passes. Evidence: exit 0, 1636 frontend tests, Go tests and lint.
