# Makefile release notes: complete the milestone before tagging

## What / Why
The comment above the release targets in `Makefile` lists two steps: prepare in the release issue's worktree, then tag on main. It does not say when to complete the milestone. In 1.6.0 (xpo-444842), the milestone was completed after the push. Completing a milestone writes `.xpo/issues.db`, so the tagged commit shows 1.6.0 as open, and the completion needed its own commit on main.

The comment also starts with "make release VERSION=x.y.z". That target does not exist.

## How
Replace the comment block above `release-check-version` with:

```make
# Cut a release in three steps, so the xpo workflow does the committing and merging:
#   1. In the release issue's worktree: make release-prep VERSION=x.y.z, then
#      commit and merge the issue as usual.
#   2. On main: complete the milestone, then commit .xpo/issues.db, so the
#      tagged commit records the completed milestone.
#   3. On main: make release-tag VERSION=x.y.z, push, then check the GitHub release.
```

## Acceptance criteria
- [ ] The comment lists the three steps, with the milestone completed before `make release-tag`.
- [ ] The comment does not name a `make release` target.
- [ ] No target changes.
- [ ] `make test` passes.

## Decisions
- **Comment only.** `release-tag` does not check the milestone. A check would need to read the board from make, and the comment is enough for a step done once per release.
- **No CHANGELOG entry.** The CHANGELOG becomes the GitHub release body for xpo users. A Makefile comment for maintainers changes nothing for them.
