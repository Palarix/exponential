# Type check the frontend in make lint, and make frontend report failed builds

## What

1. `make lint` runs `tsc -b` after eslint. `make test` and the pre-commit hook run `make lint`, so both get the type check.
2. `make frontend` fails when the build fails, and it does not copy old assets.

## Why

`make test` runs `go vet`, eslint, vitest and the Go tests, but not `tsc`. vitest strips types without checking them. A type error therefore passes every check and fails only `make frontend`. xpo-8fc8ba reached `main` this way.

`make frontend` piped the build through `tail -1`, so the recipe saw the exit code of `tail`, not of the build. A failed build with an old `web/dist` copied the old assets and exited 0. In xpo-8fc8ba the build failed visibly only because `web/dist` did not exist, and `cp` failed.

## How

1. In `web/package.json`, add the script `"typecheck": "tsc -b"`.
2. In `web/package.json`, change `build` to `bun run typecheck && vite build`, so the two commands cannot drift apart.
3. In the `Makefile` `lint` target, add `bun run --silent typecheck` after eslint. On failure, it prints "tsc failed" and exits 1, like the other steps.
4. In the `Makefile` `frontend` target, keep the build's output in a variable. On failure, print all of it and exit 1. On success, print the last line, as before.
5. In `.githooks/pre-commit`, update the comment to name the type check.
6. In `CLAUDE.md`, update the `make lint` and "Before declaring implementation done" lines.
7. Add a CHANGELOG entry.

## Decisions

- **The check goes in `lint`, not only in `test`.** The user chose this. The pre-commit hook runs `make lint`, and it runs when a commit stages a Go file or a `web/` source file. The hook would have stopped xpo-8fc8ba at its commit. The cost is about 5 seconds on each such commit: measured at 6s cold and 5s warm on this repository.
- **`tsc -b`, not a separate config.** It is the command `make frontend` runs, and it covers test files.
- **A failed build prints all its output.** `tail -1` alone hides the error the user needs.

## Not in this issue

- `.github/workflows/test.yml` runs only the Go tests and `go vet`. It has no frontend steps.

## Acceptance criteria

- [ ] A type error in a file under `web/src`, test files included, makes `make lint` and `make test` exit non-zero, and the output names "tsc failed".
- [ ] The pre-commit hook refuses a commit that stages a `web/` source file while such an error exists.
- [ ] With an old `web/dist` and a type error, `make frontend` exits non-zero, prints the error, and leaves `internal/server/static` unchanged.
- [ ] `make lint`, `make test` and `make frontend` pass on a clean tree.
