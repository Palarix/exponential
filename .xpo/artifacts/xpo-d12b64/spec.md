# Milestones: rename Releases to Milestones, with a one-time data migration

## What

Rename the Releases feature to Milestones in every surface: data model, events, MCP, CLI, server API, web app and docs. The web app uses Lucide's `Milestone` icon (a sign post) instead of `Package`. Boards that already have release data get a one-time migration through `xpo doctor --fix`, which rewrites `issues.db`, `archive.db` and drafts in place.

## Why

xpo groups work that belongs together. It does not know whether that work becomes a public build. "Release" claims something xpo does not track. "Milestone" names only what xpo knows. It pairs with Cycles (time-boxed vs scope-boxed), and it keeps the word "Release" free for a later feature that ties a milestone to a shipped version.

Releases is still in `[Unreleased]`, so we rename in place, with no compatibility layer (see the no-backward-compat rule). This board and other projects of the user already have release data. Those boards need the migration once. Parent epic: xpo-4c92d4.

## Acceptance Criteria

- [ ] The code, MCP tools, CLI, API, web app, xpo skill and README contain no "release" for this feature. The CHANGELOG has one combined Milestones entry.
- [ ] The MCP tool `milestone` replaces `release`. `add`, `update` and `list` take `milestone` instead of `release`.
- [ ] `xpo milestone list|show|create|update|reorder|complete|delete` replaces `xpo release`. `--milestone` replaces `--release`.
- [ ] `/api/milestones` replaces `/api/releases`. Issue JSON has `milestone_id` and `milestone`.
- [ ] A finished milestone is completed: state `COMPLETED`, `completed_at`, and "Completed" in the web app.
- [ ] The web app has a **Milestones** view, sidebar entry, filter, picker, chip, context menu and palette command.
- [ ] Every milestone surface in the web app shows Lucide's `Milestone` icon. No surface uses `Package` for milestones.
- [ ] `xpo doctor` reports "Milestone data uses the old Release names" when it finds old names in `issues.db`, `archive.db` or a draft.
- [ ] `xpo doctor --fix` migrates the files and creates a `.bak` copy of each changed file.
- [ ] After the migration, the board projects the same milestones, assignments, stats, burndowns and feeds as before, with new names.
- [ ] A second `xpo doctor --fix` run changes nothing.
- [ ] With old names still in the log, every read command fails with: "This board has Release data from before 1.6. Run `xpo doctor --fix` to migrate it."
- [ ] This board is migrated, and the migrated `issues.db` is committed with the change.

## Flow

### Rename

1. Rename the model in `internal/model/types.go`. `EventTypeRelease*` becomes `EventTypeMilestone*` (`MILESTONE_CREATE`, `_UPDATE`, `_COMPLETE`, `_DELETE`).
2. Change `ReleaseIDPrefix` from `rel-` to `ms-`.
3. Change `ReleaseReleased`/`"RELEASED"` to `MilestoneCompleted`/`"COMPLETED"`. Rename `ReleasedAt`/`released_at` to `CompletedAt`/`completed_at`.
4. Rename the JSON keys `release_id` to `milestone_id` in `CreatePayload`, `UpdatePayload` and `ChildrenMove`.
5. Rename `release` to `milestone` in `IssueDraftPayload` and the activity entry.
6. Rename `internal/exponential/release.go`, `internal/server/releases.go` and `cmd/exponential/release.go`, and their tests.
7. Rename the MCP tool, its parameters and its description in `internal/mcpserver/tools.go`.
8. Rename the endpoints in `internal/server/router.go` and the response types.
9. Rename `web/src/components/Releases/` to `Milestones/`, with its files, types, query keys, client calls and copy.
10. Replace the `Package` icon with Lucide's `Milestone` icon, at the same sizes. The uses are in `Sidebar.tsx`, `CommandPalette.tsx`, `ReleaseChip.tsx`, `FilterMenu.tsx`, `PropertySidebar.tsx` (2), `NewIssueModal.tsx` and `ContextMenu.tsx`.
11. Change the web copy for finished milestones: "Past releases" becomes "Completed", and "shipped" or "released" becomes "completed".
12. Rename the filter key `releases` to `milestones`. Saved filters with the old key are dropped.
13. Rename the localStorage key for the progress mode. The old key is ignored.
14. Rename the view ID and route to `#/milestones`.
15. Update the xpo skill text, the Work Mode prompt, the README and the `[Unreleased]` CHANGELOG entries.

### Migration

16. Add `internal/storage/migrate_milestones.go` with `MigrateMilestones(dir string) (MigrationResult, error)`.
17. Rewrite each event line under the events lock. Change these:
    - `type`: `RELEASE_*` becomes `MILESTONE_*`.
    - `id` and `cause`: a leading `rel-` becomes `ms-`.
    - `payload.release_id` and `payload.children_moved.release_id` become `milestone_id`, with `rel-` becoming `ms-`.
    - `payload.moved_to` on milestone events: `rel-` becomes `ms-`.
18. Leave all other bytes on a line unchanged, including key order and timestamps.
19. After the rewrite, remove exact duplicate lines. Keep the first one.
20. Write each file through the existing temp-file-and-rename path. Write `<file>.bak` first.
21. Rewrite `release` to `milestone` in `.xpo/drafts/*.json`.
22. Add `HasOldReleaseData(dir)`. `ReadEvents` calls it and returns the error from the AC.
23. Add the doctor check and its AutoFix in `cmd/exponential/doctor.go`.
24. Run the migration on this board. Commit the result with the change.

## Decisions

- **The migration lives in `xpo doctor --fix`, not in a separate script.** Doctor already finds and fixes board problems (`.gitattributes`, stale skills). It runs under the events lock and uses the tested rewrite path. Other projects get the fix when they upgrade xpo, with no script to find.
- **The migration is explicit, not automatic on load.** Rewriting a committed, shared file without being asked would surprise the user. The read error tells the user what to run.
- **The migration rewrites strings in place, not by parse and re-marshal.** The bytes of lines without release data stay the same, so the git diff shows only real changes.
- **`ms-` replaces `rel-`.** Mixed prefixes would keep the old name alive in data forever.
- **A finished milestone is completed (`COMPLETED`), confirmed by the user.** "Completed" says only what xpo knows. The value is uppercase, like `OPEN` and the issue statuses.
- **Lucide's `Milestone` icon replaces `Package`, at the user's request.** A sign post marks a point on the way. A package suggests a shipped bundle, which is the meaning the rename removes. The installed `lucide-react` already exports `Milestone`.
- **Saved filters and the progress-mode setting are dropped, not migrated.** They are per-browser conveniences, and the feature has not shipped.
- **The shortcuts stay `r` (picker) and `g v` (view).** `m` is Add comment in the issue detail, and `g m` is My Issues.

## Edge Cases

- **HIGH:** a branch from before the migration merges into a migrated main. `merge=union` keeps the branch's old lines next to the new ones. Doctor then reports old names again. A second `--fix` converts the old lines, and the dedupe step removes the copies. A test covers this.
- **HIGH:** a write runs while the migration runs. The migration holds the events lock for the whole read-rewrite.
- **MEDIUM:** an issue title, description or comment contains "RELEASE_" or "rel-". The rewrite changes only the named keys, never free text. A test covers this.
- **MEDIUM:** `archive.db` does not exist. The migration skips it.
- **LOW:** a `.bak` file already exists. The migration overwrites it.

## Assumptions

- Nobody runs a released xpo build with Releases, so no shipped client reads these names.
- The other projects with release data run this repo's local builds and can run `xpo doctor --fix` once.
