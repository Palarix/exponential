# Walkthrough: Type check in make lint, and make frontend reports failed builds

## What was built and why

xpo-8fc8ba reached `main` with a type error in a test fixture. `make test` passed, because vitest strips types without checking them, and nothing else ran `tsc`. Only `make frontend` failed.

Two things changed:

- `make lint` now runs `tsc -b`, so `make test` and the pre-commit hook run it too.
- `make frontend` now fails when the build fails.

## How the pieces fit

- `web/package.json` has a new `typecheck` script, `tsc -b`. The `build` script calls it, so the build and the lint step run the same check.
- The `lint` target in the `Makefile` runs `typecheck` after eslint, and prints "tsc failed" on failure.
- The `frontend` target keeps the build's output in a variable. On failure, it prints all of the output and "frontend build failed", then exits. On success, it prints the last line, as before.

## Key decisions

- **The check is in `lint`, not only in `test`.** The user chose this. The pre-commit hook runs `make lint`, so a type error stops at its commit.
- **The cost is about 5 seconds per commit.** Measured at 6s cold and 5s warm. The hook runs the full `make lint` when a commit stages a Go file or a `web/` source file, so Go-only commits pay it too.

## Non-obvious things

- **The old `frontend` recipe hid failures.** It piped the build through `tail -1`, so the recipe saw the exit code of `tail`. A failed build with an old `web/dist` copied the old assets and exited 0. xpo-8fc8ba failed visibly only because `web/dist` did not exist, and `cp` failed.
- **`internal/server/static` stays intact on failure.** The `rm` and `cp` are later recipe lines, and make stops at the first failing line.
- **These checks are local only.** `.github/workflows/test.yml` runs only the Go tests and `go vet`. The hook needs `make setup`, which sets `core.hooksPath`.

## Acceptance criteria

- [x] A type error in a file under `web/src`, test files included, makes `make lint` and `make test` exit non-zero, and the output names "tsc failed". Evidence: a temporary `__typecheck_probe.test.ts`; both exited 2.
- [x] The pre-commit hook refuses a commit that stages a `web/` source file while such an error exists. Evidence: `git commit` with the probe staged exited 1 with "tsc failed", and HEAD did not move.
- [x] With an old `web/dist` and a type error, `make frontend` exits non-zero, prints the error, and leaves `internal/server/static` unchanged. Evidence: exit 2, TS2322 and "frontend build failed" in the output, same checksums before and after.
- [x] `make lint`, `make test` and `make frontend` pass on a clean tree. Evidence: all exit 0; 1614 frontend tests and the Go suite.
