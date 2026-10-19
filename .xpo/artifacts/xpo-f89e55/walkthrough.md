# Walkthrough: Releases — data model, MCP and CLI

This story adds the backend for releases (epic xpo-4c92d4). Releases group issues that ship together. A release is numbered `#1`, `#2`, …, may have a label, and is OPEN until it is completed (RELEASED, final) or deleted. The web UI comes in the next stories; everything here is the model, the API and the CLI/MCP surfaces.

## How it is stored

Releases are event-sourced in `issues.db`, next to the issue events, with four new event types (`RELEASE_CREATE/UPDATE/COMPLETE/DELETE`) and IDs like `rel-3fa9c1`. Living in the same file means they get the existing lock, ordering, git staging, file watcher and `merge=union` handling for free. `ProjectIssues` never sees them as issues (their IDs never match an issue create).

`model.Release` is the projected state. `ProjectReleases(events)` (in `internal/exponential/release.go`) replays the release events and returns every release, deleted ones included, because the next number is "highest number ever + 1": numbers are never reused, and there is no stored counter. After a git merge, two branches may both have created `#4`; `renumberReleases` keeps the number on the older release and gives the newer one the next free number.

Issues carry one `ReleaseID` (in `CreatePayload`, `UpdatePayload` and `Issue`). Projection also fills `Issue.ReleaseName` (the label, or `#7`), so every output can show it without another lookup.

## Current, Next and placeholders

Current and Next are the first two OPEN releases in the manual order (`SortOrder`, fractional keys from `internal/sortorder`). They always exist: `CurrentNext` fills a missing slot with a placeholder, which has a number but no ID and is not stored. The first write to a placeholder (an issue assignment, a label, a goal, a target date) saves it.

While Current is a placeholder, the Next placeholder is **locked**: writes to it fail with "Next (#2) is locked: save Current first". Without the lock, saving Next first would make it the only saved release, and so Current. An earlier version saved an empty Current as a side effect; that was removed in review.

`started_at` is set the first time a saved release becomes Current (on create, placeholder save, reorder, complete or delete) and never reset. `releaseWriter.finish` checks this after every batch that touched a release.

## Writing: one batch, one lock

`storage.AppendEvents(collapse, build)` is new: it takes the events lock, reads the log, calls `build` on it, and appends everything `build` returns in one write (or one rewrite in collapse mode). A release complete that moves 40 issues lands whole or not at all, and no other process can change the log between the read and the write. `AddIssue` and `applyUpdate` now use it too, so an issue update and its cascades (parent auto-start, link cleanup) are also atomic.

`releaseWriter` builds a batch against that snapshot. Each release event it emits re-projects the releases, so later steps in the same batch see a placeholder that an earlier step saved. Its main helpers:

- `assign(ref, currentID)` resolves a ref for an issue, saves a placeholder, and rejects RELEASED releases (except re-sending the issue's own release, which prunes to a no-op).
- `save(placeholder)` stores a placeholder after the saved OPEN releases, and refuses a locked Next.
- `moveTarget(r)` decides where "move to Next" goes when `r` is completed or deleted: from Current to the old Next, from Next to the release after it, from a later release to Next. A placeholder target is saved; it is never a locked Next, because Current is saved whenever a release is being completed or deleted.
- `moveIssues(issues, dest, cause)` writes one issue UPDATE per issue, marked with the cause.

**Refs.** `release_id` in an add/update payload may hold any ref until the write resolves it: `#7`, a `rel-` ID, `@current`, `@next`, `none`, or a label. Only `#` starts a number ref, so `2026` or `7` are valid labels. Labels may not start with `#`, `@` or `rel-`, or be `none`, and are unique among OPEN releases regardless of case. The stored event always holds the `rel-` ID. This lets the web app send `@next` directly. Anything that writes issue events outside `AddIssue`/`applyUpdate` must not put a ref there.

## Copy-on-create and with_children

A new issue with no release copies its parent's release, if that release is OPEN. That happens once, in `AddIssue`; there is no live inheritance, so later changes to the parent never move the child (unlike `EffectiveCycleID`, which is cycle rollover and was deliberately not copied). An explicit `none` skips the copy. Re-parenting an existing issue keeps its release.

`update` with `with_children` (`UpdateOptions`) also moves the issue's open descendants, at any depth. Terminal children never move.

## Complete and delete

`CompleteRelease(ref, mode)`: open issues move to Next (`move_to_next`, default) or leave the release (`remove`); terminal issues stay as the record. The complete event stores a stats snapshot (counts by status and leaf IDs). `xpo archive` moves done issues' events to `archive.db`, and without the snapshot a past release would lose them from its stats.

`DeleteRelease(ref, mode)` moves all issues, done ones included, because an abandoned release's finished work still ships later (epic decision 10). Placeholders cannot be completed, reordered or deleted, and RELEASED releases cannot be deleted or reordered. Their label, goal and target date can still be edited.

## Stats and burndown

Only **scope** counts: leaf issues (no children) that are not CANCELED or DUPLICATE. This avoids double counting epics, and a release whose remaining issues are done reads as complete (8 done + 2 canceled = 8/8). `ReleaseCounts` uses the snapshot for RELEASED releases. `ComputeReleaseProgress` replays events into daily points (scope, open, completed) from `started_at` to `released_at` or today, in UTC days like cycle progress. The CLI and server pass archived events too, so a past release keeps its burndown.

## Feeds: one entry per bulk operation

A complete, delete or `with_children` move can touch many issues. To keep the feeds readable:

- `model.Event.Cause` marks the issue UPDATEs a bulk operation wrote (cause = the release, or the epic).
- `IsFeedEvent` (activity feed, `/api/timeline`, global `xpo history`) and the inbox skip caused events. Release complete/delete are feed entries, and team-wide inbox entries like merges.
- The complete/delete payloads carry `moved` and `moved_to`, and the epic's update carries `children_moved: {count, release_id}`. The release ID is there because the epic's own `release_id` is pruned when it did not change. The feed shows "released 1.5 · moved 3 open issues to 1.6" or "moved this issue and 1 open child to release 1.5".
- An issue's own history keeps the caused move, with a note: "moved this issue to release 1.6 when 1.5 was released".
- Collapse treats the cause as part of the actor, so a caused move never folds into a user's own edit.

`ReleaseNoter` names the releases an event refers to by their current display names; `ui.DescribeEventNamed` turns that into text.

## Surfaces

- **MCP:** a `release` tool (`list`, `create`, `update`, `reorder`, `complete`, `delete`); `release` on `add`/`update`/`list`; `with_children` on `update`; `release: null` clears.
- **CLI:** `xpo release [list|show|create|update|reorder|complete|delete]` with `--json`; `--release` on `add`/`update`/`list`; `--with-children` on `update`; the release line in `xpo show`.
- **Server:** `GET /api/releases` (current, next, upcoming, past, with counts and `locked`), `GET /api/releases/{ref}/progress`, `POST /api/releases`, `PUT /api/releases/{ref}`, `POST /api/releases/reorder`, `POST /api/releases/{ref}/complete`, `DELETE /api/releases/{ref}?issues=`. Issue responses have `release_id` and `release`; activity items have `release` and `moved_to`.

## Not done here

- `/api/draft` still answers a refused assignment with a generic 500 instead of a 400 that carries the message (for xpo-974eba).
- The web feed skips `RELEASE_*` items and updates that only change an issue's release until the UI stories render them.

## Acceptance criteria

- [x] Releases can be created, updated, reordered, completed and deleted from MCP, CLI and the server API. Evidence: `TestReleaseTool_Lifecycle`, `TestReleaseCLI_Lifecycle`, `TestReleasesAPI_Lifecycle`.
- [x] `GET /api/releases` always returns a Current and a Next; missing slots are placeholders; Next is locked while Current is a placeholder. Evidence: `TestReleasesAPI_EmptyBoardHasPlaceholders`, `TestCurrentNext_PlaceholdersFillMissingSlots`.
- [x] A write to a placeholder saves it; a placeholder cannot be completed, reordered or deleted; a write to a locked Next fails. Evidence: `TestUpdateRelease_PlaceholderIsSavedOnFirstWrite`, `TestPlaceholders_CannotBeCompletedReorderedOrDeleted`, `TestLockedNext_RejectsWritesUntilCurrentIsSaved`, `TestAddIssue_PlaceholderRefSavesIt`.
- [x] Numbers are sequential and never reused, also after a delete. Evidence: `TestCreateRelease_NumbersAreSequentialAndNeverReused`, `TestProjectReleases_DuplicateNumberGoesToLaterRelease`.
- [x] A duplicate label among OPEN releases fails; labels starting with `#` or `@`, or `none`, fail. Evidence: `TestCreateRelease_DuplicateOpenLabelFails`, `TestCreateRelease_RejectsLabelsThatLookLikeRefs`, `TestCreateRelease_NumericLabelsAreLabels`.
- [x] Complete moves open issues to Next or removes them; done issues stay; RELEASED cannot change again. Evidence: `TestCompleteRelease_MovesOpenIssuesToNextAndKeepsDone`, `TestCompleteRelease_RemoveClearsOpenIssues`, `TestCompleteRelease_MoveToPlaceholderNextSavesIt`.
- [x] Delete moves all issues to Next or removes them; RELEASED releases cannot be deleted. Evidence: `TestDeleteRelease_MovesAllIssuesIncludingDone`, `TestDeleteRelease_NextMovesToTheReleaseAfterIt`, `TestDeleteRelease_CurrentWithPlaceholderNext`, `TestDeleteRelease_RemoveClearsAllIssues`.
- [x] Assigning an issue to a RELEASED release fails. Evidence: `TestAddIssue_ReleasedReleaseRejected`, `TestUpdateIssue_ReleasedReleaseRejected`.
- [x] A new child copies its parent's OPEN release; a later parent change does not change the child. Evidence: `TestAddIssue_ChildCopiesParentsOpenRelease`, `TestAddIssue_ChildOfReleasedParentGetsNoRelease`, `TestUpdateIssue_ReparentKeepsRelease`.
- [x] `update` with `with_children` moves open descendants; done children never move. Evidence: `TestUpdateIssue_WithChildrenMovesOpenChildrenOnly`, `TestUpdateTool_WithChildren`.
- [x] `release` refs `#7`, a label, `@current`, `@next` and `none` work on add, update and list. Evidence: `TestResolveRelease_Refs`, `TestUpdateIssue_ReleaseRefs`, `TestListIssues_ReleaseFilter`.
- [x] Progress returns counts and a burndown from `started_at`, for scope issues only; a RELEASED release uses its snapshot. Evidence: `TestComputeReleaseProgress_CountsLeavesAndBurnsDown`, `TestComputeReleaseProgress_ReleasedUsesSnapshot`, `TestComputeReleaseProgress_NeverCurrentHasNoBurndown`.
- [x] Release events are not collapsed, pruned or archived. Evidence: `TestAppendEventCollapsed_KeepsReleaseEvents`, `TestArchiveEvents_NeverArchivesReleaseEvents`, `TestAppendEventCollapsed_DoesNotFoldAcrossCauses`.
- [x] The activity timeline describes release changes; complete, delete and `with_children` make one feed and inbox entry each. Evidence: `TestCompleteRelease_MovesAreCausedAndCounted`, `TestWithChildren_OneEntryForTheEpic`, `TestBuildInbox_ReleaseIsTeamWideAndMovesAreQuiet`, `TestActivity_CompleteIsOneEntry`, `TestDescribeEvent_ReleaseBulkEntries`. A CLI run in a scratch project showed "released 1.5 · moved 3 open issues to 1.6" and "moved this issue to release 1.6 when 1.5 was released".
- [x] `make test` passes (exit 0: lint, vitest, Go suite).
