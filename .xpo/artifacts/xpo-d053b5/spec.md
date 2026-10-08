# Spec: Drop bubbletea v1's import-time terminal query

## What
Migrate `cmd/exponential`'s two huh prompts from `github.com/charmbracelet/huh` v1.0.0 to `charm.land/huh/v2`. This removes bubbletea v1 from the binary, along with its package `init()` that queries the terminal's background colour and stalls every command for 5 s under a pty that doesn't answer.

## Why
Measured under `script -q /dev/null`: `xpo version` takes 5.03 s (0.02 s normally), and `xpo list` 5.06 s, each printing a stray `ESC ]11;?`. No xpo code can run before a dependency's package init, so the fix has to remove the dependency. Upstream drops the workaround in v2.

## How
1. `go get charm.land/huh/v2@v2.0.3`, then `go mod tidy`. This removes `github.com/charmbracelet/huh` and bubbletea v1 from `go.mod`.
2. `cmd/exponential/init.go` and `cmd/exponential/prompt.go`: change the import to `charm.land/huh/v2`. The API used (`NewMultiSelect`, `NewSelect`, `NewOption(...).Selected`, `Title`, `Options`, `Value`, `Run`) is unchanged in v2, as the prototype confirmed.
3. Regression test `TestBinaryDoesNotLinkBubbleteaV1` in `cmd/exponential`: read `debug.ReadBuildInfo()` of the test binary (which links the same deps as the CLI) and fail if `github.com/charmbracelet/bubbletea` (v1) appears.
4. Manual check: time `xpo version` under `script -q /dev/null`, and look for `ESC ]11;?` in its output.

## Decisions
- **Upgrade huh, don't replace it** with hand-rolled prompts. It's a small mechanical change, keeps the polished select UI, and follows upstream's own fix. lipgloss v1 and glamour stay as they are (different module path from lipgloss v2, so they coexist).
- **Option labels styled with lipgloss v1** (`ui.MutedStyle.Render`) are passed to huh v2 as plain strings with ANSI codes, which still render.

## Acceptance Criteria
- [ ] Under `script -q /dev/null`, `xpo version` and `xpo list` take well under 1 s and emit no `ESC ]11;?`.
- [ ] `TestBinaryDoesNotLinkBubbleteaV1` passes, and fails if bubbletea v1 is reintroduced.
- [ ] The interactive agent multi-select (`xpo init`) and `promptSelect` still work (checked in a real terminal at tophat).
- [ ] `make test` passes.
