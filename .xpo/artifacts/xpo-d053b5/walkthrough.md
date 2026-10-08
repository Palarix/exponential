# Walkthrough: Drop bubbletea v1's import-time terminal query

## What was built and why

Under a pseudo-terminal that doesn't answer terminal queries (`script -q /dev/null …`, some agent harnesses and CI wrappers), every `xpo` command stalled about 5 s and printed a stray `ESC ]11;?`:

| | before | after |
|---|---|---|
| `xpo version` | 5.03 s, 1 query | 0.39 s, 0 |
| `xpo list` | 5.06 s, 1 query | 0.06 s, 0 |

## Root cause

A goroutine dump showed the process blocked in `runtime.doInit`:

```
bubbletea.init.0 (tea_init.go:21)
  → lipgloss.HasDarkBackground
  → termenv.termStatusReport → readNextResponse → waitForData(5s)
```

`cmd/exponential` imported `github.com/charmbracelet/huh` v1 for two interactive selects. huh v1 imports bubbletea v1, and bubbletea v1's **package `init()`** calls `lipgloss.HasDarkBackground()`. termenv writes OSC 11 and `CSI 6n` to the terminal and waits `OSCTimeout` (5 s) for a reply. This happens on every process start, before `main`, so there's no flag or env-var hook in xpo code that can avoid it. Upstream's own comment says "This workaround will be removed in v2", and it is still present in v1.3.10.

Note: termenv only queries when stdout is a terminal it can control. Piped or `/dev/null` output and `CI=…` are unaffected, which is why this only showed up under a pty.

## The fix

- `cmd/exponential/init.go` (agent multi-select) and `cmd/exponential/prompt.go` (`promptSelect`) now import `charm.land/huh/v2`. The API they use (`NewSelect`, `NewMultiSelect`, `NewOption(...).Selected`, `Title`, `Options`, `Value`, `Run`) is identical in v2, so the call sites didn't change.
- `go.mod`: huh v1 and bubbletea v1 are gone; `charm.land/huh/v2 v2.0.3` brings bubbletea v2 and lipgloss v2 as indirect deps. lipgloss v1 and glamour stay in use elsewhere; the module paths differ, so both versions coexist.
- **Transitive fallout:** huh v2 requires `charmbracelet/x/ansi` v0.11.x, whose `ansi.Style` API changed. That broke `x/cellbuf` v0.0.13, used by lipgloss v1. `x/cellbuf` was bumped to v0.0.15, which is compatible. Minor bumps came along: `go-runewidth`, `go-colorful`, `uax29`.

## Regression guard

`TestBinaryDoesNotLinkBubbleteaV1` (`cmd/exponential/deps_test.go`) reads `debug.ReadBuildInfo()` of the test binary, which links the same dependency graph as the CLI. It fails if `github.com/charmbracelet/bubbletea` (v1) shows up again, for example via a new dependency on huh v1 or bubbles v1. It failed before the change and passes after.

## Correction recorded

The original report said `xpo init --yes` hung for 180 s after writing files. An exact replay exits in about 1 s, so that was most likely a later step in the same shell chain. The confirmed problem is the 5 s pty stall fixed here.

## Acceptance Criteria

- [x] Under `script -q /dev/null`, `xpo version` / `xpo list` take well under 1 s and emit no `ESC ]11;?`. *Evidence:* 0.39 s / 0.06 s, 0 queries (table above).
- [x] `TestBinaryDoesNotLinkBubbleteaV1` passes, and fails with bubbletea v1 linked. *Evidence:* it failed on main's dependencies, passes after the migration.
- [x] The interactive agent multi-select and `promptSelect` still work. *Evidence:* checked by the user in a real terminal ("lgtm").
- [x] `make test` passes.
