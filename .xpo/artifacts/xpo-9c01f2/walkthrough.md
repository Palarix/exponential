### Rationale

Walkthroughs are now persisted as first-class artifacts via `WriteWalkthrough`, so dumping the full glamour-rendered markdown to the terminal is redundant — the user already has a permanent, retrievable copy. A one-liner pointing to `xpo artifact show` keeps terminal output tight and consistent with the other `✔` status lines that `xpo drive` already prints.

### Changes

- **`internal/exponential/drive.go`**: Replaced `log.walkthrough(walkthrough)` with `log.ok(fmt.Sprintf("Walkthrough saved → xpo artifact show %s --walkthrough", issue.ID))` — the user now sees a single status line instead of a multi-page markdown dump.
- **`internal/exponential/drive.go`**: Removed the `driveLogger.walkthrough` method (glamour-based terminal rendering) and its helper `wordWrap` — no remaining callers.
- **`internal/exponential/drive.go`**: Removed the `github.com/charmbracelet/glamour` import — only consumer was the deleted method.

### How to verify

1. `make build` — confirms the glamour import removal doesn't break compilation.
2. `make test` — all tests pass (verified in test output above).
3. Run `xpo drive` on a test issue and confirm:
   - Terminal prints `✔ Walkthrough saved → xpo artifact show <id> --walkthrough` instead of rendered markdown.
   - Running the printed command actually displays the walkthrough.
4. Pipe `xpo drive` output to a file (non-TTY) — the `log.ok` path works in both TTY and non-TTY modes since it uses the same codepath as the other status lines.

### What to look out for

The `glamour` dependency may now be unused project-wide. Worth checking whether `go mod tidy` drops it — if so, that's a free reduction in binary size. Not blocking, but worth a follow-up.