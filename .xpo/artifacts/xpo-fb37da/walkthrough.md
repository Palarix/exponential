# Walkthrough: Board registry concurrent writes corrupt instances.json

## What was wrong

`internal/registry` keeps one `instances.json` in `<os temp dir>/xpo`, shared by every running `xpo board`. Each board registers on startup and unregisters on shutdown, and `List()` prunes dead PIDs. All three ran load-modify-save with no lock, and `save` always wrote to the same `instances.json.tmp` before renaming it. When two boards wrote at the same moment, both wrote into that tmp file, and the renamed result could be invalid JSON. `load` then failed on every call, so `/api/instances` returned 500 on every board until someone fixed the file by hand. `TestHandleListInstances` read the real file, so a corrupt registry also failed `make test`.

## What changed

All of it is in `internal/registry/registry.go`.

- **`withLock(fn)`** takes an exclusive `gofrs/flock` lock on `instances.lock` in the registry dir (5 s timeout, 5 ms retry), the same pattern as `storage.withEventsLock`. `List`, `Register` and `UnregisterByPID` run their whole load-modify-save inside it. flock locks belong to the open file, so goroutines in one process also exclude each other. The lock file is never deleted: deleting it would let two processes hold locks on different inodes.
- **`save`** writes through `os.CreateTemp(dir, "instances-*.json.tmp")`, chmods it to 0644, and renames it over `instances.json`. On any error it deletes the tmp file. With the lock, a unique name isn't strictly needed. It protects against an older binary that still writes without the lock. An empty list is written as `[]`, not `null`.
- **`load`** returns `(entries, corrupt, err)`. JSON that doesn't parse, including an empty file, loads as an empty list with `corrupt = true`. `List` rewrites the file when it is corrupt or when it pruned entries. `Register` and `UnregisterByPID` always save, so they repair a corrupt file too. A read error other than "not exist" is still returned.
- **The directory** is now a package variable (`registryDir`, guarded by a mutex) that defaults to `<os temp dir>/xpo`. **`SetDirForTesting(dir)`** swaps it and returns a restore function. The registry is created with `MkdirAll` inside `withLock`, so a test dir doesn't need to exist in advance.

## Tests

- `internal/registry/registry_test.go` (new):
  - register, list and unregister round trip
  - recovery from a corrupt file for `List`, `Register` and `UnregisterByPID` (an extra `]`, an empty file, garbage)
  - 50 goroutines registering, listing and unregistering at once
  - 8 child processes (re-execs of the test binary through `TestHelperProcess`) that each register and list 20 times, wait for a release file, then unregister. The parent checks that exactly those PIDs are listed and that the file ends empty and valid.
  - every test checks that no `.tmp` file is left behind
- `setupTestServer` in `internal/server/handlers_test.go` points the registry at the test's temp dir.
- `TestHandleListInstances` now expects an empty list. `_UsesTestRegistry` proves the handler reads the redirected dir, and `_CorruptRegistry` proves a corrupt file returns 200.

## Acceptance criteria

- [x] Concurrent `Register`/`UnregisterByPID` calls from many goroutines leave valid JSON with exactly the expected entries: `TestConcurrentGoroutineWritesStayValid` (passes with `-race`).
- [x] Concurrent writers from separate processes leave valid JSON: `TestConcurrentProcessWritesStayValid`.
- [x] `List` on a corrupt file returns an empty list without error and rewrites valid JSON: `TestListRecoversFromCorruptFile`, and `TestHandleListInstances_CorruptRegistry` at the API level.
- [x] `Register` on a corrupt file succeeds and leaves only the new entry: `TestRegisterRecoversFromCorruptFile`.
- [x] No `instances*.tmp` files are left behind: `assertNoTmpFiles` in the round-trip and concurrency tests.
- [x] Tests never touch the real registry: `setupTestServer` and `useTempDir` redirect it. After `make test`, the real registry dir had no `instances.lock` and the timestamp of `instances.json` hadn't changed.
- [x] `make test` passes.
