# Spec: Fix DefaultBranch local fallback

## What

`DefaultBranch()` should detect the local default branch when no remote is configured, instead of hardcoding `"main"`.

## How

Current logic:
1. Try `git symbolic-ref refs/remotes/origin/HEAD` → parse branch name
2. Fall back to `"main"`

Fixed logic:
1. Try `git symbolic-ref refs/remotes/origin/HEAD` → parse branch name
2. Try `git symbolic-ref HEAD` → parse local branch name (the init branch)
3. Fall back to `"main"`

## Test changes

- `TestDefaultBranch_FallbackWithoutRemote`: change expected from `"main"` to `"trunk"`
- Add `TestDefaultBranch_NoGitRepo`: verify fallback to `"main"` outside a git repo

## AC

- [ ] Local repo on `trunk` without remote returns `"trunk"`
- [ ] Local repo on `main` without remote returns `"main"`
- [ ] Remote repo with `develop` default returns `"develop"`
- [ ] Non-git directory returns `"main"`
- [ ] `make test` passes
