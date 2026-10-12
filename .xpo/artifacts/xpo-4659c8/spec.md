# Spec: Release prep for 1.4.0

## What / Why
Cut 1.4.0, a minor release, chosen by the user over the requested 1.3.1. Unreleased adds features and changes `link` behavior, so SemVer calls for a minor bump.

## How
1. **Version bump:** `make release-prep VERSION=1.4.0` in this worktree sets `CLIVersion` and turns `## [Unreleased]` into `## [1.4.0] — 2026-10-12`, with a new empty Unreleased section above it.
2. **Markers:** `make cli`, then `./xpo doctor --fix`, which refreshes the `xpo:begin` / `xpo:skill` markers to 1.4.0. The skill template did change since 1.3.0: xpo-c93b11 added the one-link-per-pair guidance. So the refreshed skill files also pick up that text, as in xpo-e60c26.
3. **README:** add "## Upgrading from 1.3.x" above "Upgrading from 1.2.x". It covers:
   - **One link per issue pair** (xpo-c93b11): `link` on an already-linked pair fails unless `overwrite` / `--overwrite` is set. `add`/`update` reject a `links` set with two entries for the same issue. Pairs that already have more than one link keep working until a write changes them.
   - **Re-run `xpo init`, or `xpo doctor --fix`,** to pick up the updated skill text about links.
   - No other Unreleased item changes behavior for existing users. The rest are UI additions or fixes.
4. **Add the CHANGELOG entry** for this issue in the 1.4.0 section.
5. **After merge:** on main, `make release-tag VERSION=1.4.0`. Then `git push origin main v1.4.0`, only after the user confirms. That push triggers the goreleaser build in `.github/workflows/build.yml`.

## Acceptance criteria
- [ ] `CLIVersion` is `1.4.0`, and `xpo version` prints it.
- [ ] CHANGELOG has `## [1.4.0] — 2026-10-12` with an empty Unreleased section above it.
- [ ] README has an "Upgrading from 1.3.x" section.
- [ ] Markers carry 1.4.0, and `xpo doctor` is clean.
- [ ] `make test` passes.
- [ ] Tag `v1.4.0` is created on main after merge and pushed only after the user confirms.
