# Walkthrough: Makefile release notes: complete the milestone before tagging

## Summary
The comment above the release targets in `Makefile` lists three steps. The new step 2 completes the milestone and commits `.xpo/issues.db` before `make release-tag`. Then the tagged commit records the completed milestone. In 1.6.0 (xpo-444842), the milestone was completed after the push, so the tag shows it as open.

## Changes

### Makefile
- The comment no longer names `make release`. That target does not exist.
- Step 1 is unchanged: run `make release-prep` in the release issue's worktree, then commit and merge.
- Step 2 is new: on main, complete the milestone and commit `.xpo/issues.db`.
- Step 3 tags, pushes and checks the GitHub release.

No target changed.

## Decisions
- **Comment only.** A check in `release-tag` would need to read the board from make. The comment is enough for a step done once per release.
- **No CHANGELOG entry.** The CHANGELOG becomes the GitHub release body for xpo users, and this comment is for maintainers. The first draft had an entry, and the user asked to remove it.

## Acceptance criteria
- [x] The comment lists the three steps, with the milestone completed before `make release-tag`. Evidence: `Makefile`, the comment above `release-check-version`.
- [x] The comment does not name a `make release` target.
- [x] No target changes. Evidence: the diff changes only comment lines, and `make -n release-check-version VERSION=9.9.9` parses.
- [x] `make test` passes. Evidence: exit 0, vitest 1777/1777, Go tests and lint clean.
