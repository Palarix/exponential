# Spec: Board registry concurrent writes corrupt instances.json

## What

Make `internal/registry` safe when several `xpo board` processes read and write `instances.json` at once, let it recover from a corrupt file, and stop the server test from touching the developer's real registry.

## Why

`save` writes to a fixed `instances.json.tmp` and renames it, with no lock. Two boards that register or unregister together interleave their writes into the same tmp file and leave invalid JSON. From then on `List()` fails, `GET /api/instances` returns 500 on every board, and `TestHandleListInstances` fails `make test` on that machine, because it reads the real file.

## How

1. **Lock.** Add `withLock(fn)` in `registry.go`: an exclusive `gofrs/flock` lock on `instances.lock` next to `instances.json`, with a timeout (5 s) and a short retry delay, the same pattern as `storage.withEventsLock`. `Register`, `UnregisterByPID` and `List` run their whole load-modify-save inside it.
2. **Unique tmp file.** `save` writes through `os.CreateTemp(dir, "instances-*.json.tmp")` and renames that file over `instances.json`. It removes the tmp file if the write or rename fails. With the lock this is defence in depth, and it also protects against an older binary that writes without the lock.
3. **Recover from a corrupt file.** `load` returns an empty list and `corrupt = true` when the JSON does not parse. Callers then rewrite the file: `List` saves the live entries (empty) when the file was corrupt or when it pruned entries; `Register` and `UnregisterByPID` save anyway. A read error other than "not exist" is still returned.
4. **Test isolation.** The registry directory comes from a package variable that defaults to `<os.TempDir()>/xpo`. An exported `SetDirForTesting(dir string) (restore func())` swaps it. Registry tests and `TestHandleListInstances` (through `setupTestServer`) point it at `t.TempDir()`.

## Edge cases

- Lock timeout: `Register`/`UnregisterByPID`/`List` return an error; the board start already logs a failed `Register`, and `/api/instances` returns 500 as before.
- An empty file (0 bytes) counts as corrupt and is rewritten as `[]`.
- The lock file stays in the dir; it is never deleted (deleting it would race with other lockers).

## Acceptance criteria

- [ ] Concurrent `Register`/`UnregisterByPID` calls from many goroutines leave valid JSON that contains exactly the entries expected.
- [ ] Concurrent writers from separate processes (re-exec of the test binary) leave valid JSON.
- [ ] `List` on a corrupt `instances.json` returns an empty list without error and rewrites the file as valid JSON.
- [ ] `Register` on a corrupt file succeeds and leaves only the new entry.
- [ ] No `instances*.tmp` files are left behind after saves.
- [ ] `TestHandleListInstances` and the registry tests never read or write the real registry in the OS temp dir.
- [ ] `make test` passes.
