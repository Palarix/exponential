# Walkthrough: rename Releases to Milestones, with a one-time data migration

## What was built

The Releases feature is now Milestones in every surface, and `xpo doctor --fix` migrates boards that already hold release data. Releases had not shipped, so the rename happens in place, with no compatibility layer.

## The rename

The rename was done with ordered substitutions over the files that use the feature. The order matters:

1. `ReleaseReleased` → `MilestoneCompleted`, `RELEASED` → `COMPLETED`, `ReleasedAt`/`released_at` → `CompletedAt`/`completed_at`, and the words `released`/`Released` → `completed`/`Completed`. These run first, so "released" never turns into "milestoned".
2. `RELEASE_` → `MILESTONE_`, and a `rel-` at a word boundary → `ms-`.
3. Then `Releases`, `releases`, `Release`, `release` and `RELEASE` become the matching milestone forms.

Files that use "release" in other senses were left out: the Makefile's release targets, CI, the commit-msg hook, licences, `gitlock.go` ("the lock is released"), `setup_test.go` (a branch named `release`), the README's GitHub Releases link, and the stress-generator and demo text.

Some changes needed a person, not the substitution:

- **Wording:** "ship together" became "are done together"; "past milestones" in the CLI and MCP list became "completed milestones"; "shipped" and the mangled "milestone date" or "to its milestone" became "completed" or "completion".
- **Icon:** Lucide's `Milestone` replaces `Package` in 8 places. It is imported as `MilestoneIcon` because the API type is already called `Milestone`.
- **Internal names:** the slot value `past` (`board.past`, `SlotPast`) stays. It is internal; only the text users see says "completed".
- **gofmt:** the repo was not gofmt-clean before. Files that were already unformatted were not reformatted, so the diff shows only the rename.

Saved filters with the old `releases` key are dropped without breaking anything: `parseStoredFilters` spreads them over `EMPTY_FILTERS`, so `milestones` falls back to `[]`. The progress-mode setting has a new localStorage key, and the old one is ignored.

## The migration

`internal/storage/migrate_milestones.go`:

- **Matching:** old names are matched as JSON keys with their quotes, for example `"release_id":` and `"type":"RELEASE_`. In free text the quotes are escaped (`\"release_id\"`), so a backslash sits right after the key name and the pattern can't match. This is why titles, descriptions and comments that mention `RELEASE_` or `rel-` stay untouched. A test covers it.
- **`migrateLine`:** rewrites `type`, `id`, `cause` and `release_id` (with the value's `rel-` → `ms-`), including the copy inside `children_moved`. On milestone events it also rewrites `moved_to`. All other bytes stay as they are, so the git diff shows only the renamed keys.
- **`migrateLog`:** rewrites each line, then drops any rewritten line that is an exact copy of a line seen earlier. This handles a pre-migration branch merged with `merge=union`: its old lines become copies of main's new lines. It writes `<file>.bak` first (overwriting an old one), then the new file through `rewriteFile`, which writes a temp file and renames it. A file with nothing to change is left alone, with no `.bak`, so a second run changes nothing.
- **Lock:** `MigrateMilestones` does all of this under the events lock. A missing `archive.db` is skipped. Drafts get `"release":` → `"milestone":`.
- **Readers:** `ReadEvents` and `ReadArchivedEvents` check each line with `lineHasOldReleaseData` and return `ErrOldReleaseData`. The check is four `bytes.Contains` calls per line, which costs little.
- **Doctor:** `doctorMilestoneData` in `cmd/exponential/doctor.go` runs after the issues.db validity check. Without `--fix` it reports an error, because reads fail. With `--fix` it migrates and lists the changed files under "Fixed locally". Doctor is exempt from the config checks in `PersistentPreRunE` and never calls `ReadEvents` before this check, so it can still run on an old board.

Deviation from the spec: `MigrateMilestones()` and `HasOldReleaseData()` take no `dir`. Like every other storage function, they use `XpoDir()`, so the lock and the files are always in the same directory.

## Acceptance criteria

- [x] No "release" for this feature in the code, MCP, CLI, API, web app, skill or README. Evidence: `git grep -i releas` finds only the migration's old-name constants and doctor's messages about them, plus unrelated uses (Makefile, CI, hooks, the GitHub Releases link). The skill and the Work Mode prompt never mentioned the feature. The CHANGELOG has one combined Milestones entry.
- [x] The MCP tool `milestone` replaces `release`, and `add`/`update`/`list` take `milestone`. Evidence: `internal/mcpserver/tools.go`, `milestone_tools_test.go`.
- [x] `xpo milestone …` and `--milestone`. Evidence: `TestMilestoneCLI_Lifecycle`.
- [x] `/api/milestones`, plus `milestone_id` and `milestone` in issue JSON. Evidence: `internal/server/milestones_test.go`.
- [x] A finished milestone is `COMPLETED`, has `completed_at`, and shows "Completed" in the web app. Evidence: `model.MilestoneCompleted`, the server test asserting `State == "COMPLETED"`, and `MilestoneRow`'s date title.
- [x] Milestones view, sidebar entry, filter, picker, chip, context menu and palette command. Evidence: `views.ts` and the components under `web/src/components/Milestones`; vitest passes (1573 tests).
- [x] Every milestone surface shows Lucide's `Milestone` icon. Evidence: no `Package` left in `web/src`; `MilestoneChip.test.tsx` asserts `lucide-milestone`.
- [x] `xpo doctor` reports "Milestone data uses the old Release names". Evidence: `TestDoctor_MigratesOldReleaseData`.
- [x] `--fix` migrates and writes a `.bak` of each changed file. Evidence: the same test, and `TestMigrateMilestones_ArchiveAndDrafts` and `_OverwritesAnOldBak`.
- [x] Same projections after the migration. Evidence: on a copy of this board's real `issues.db`, the old binary's `release list`, `list`, `release show #1–#3` and `history`, mapped to the new names, equal the new binary's output after `doctor --fix`. The only differences are free text, which the migration leaves alone on purpose. The CLI test also checks that the migrated file is byte-identical to the original new-format file.
- [x] A second `--fix` changes nothing. Evidence: `TestMigrateMilestones_SecondRunChangesNothing` and the CLI test.
- [x] Reads fail with the AC message. Evidence: `TestReadEvents_OldReleaseDataFails` and the CLI test (`xpo list`).
- [ ] This board is migrated and the migrated `issues.db` is committed. Deferred, as agreed at handoff: migrating the hub's live `issues.db` before the merge would break the board running on the old build. After the merge: `make build`, run `xpo doctor --fix` on main, and commit `issues.db`.

## Tests

- `internal/storage/migrate_milestones_test.go`: key-only rewrite with untouched free text, second run, union-merge duplicates, archive and drafts, missing archive, `.bak` overwrite, and the read error.
- `cmd/exponential/milestone_test.go` `TestDoctor_MigratesOldReleaseData`: end to end through the binary.
