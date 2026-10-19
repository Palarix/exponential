# Spec: Releases — data model, MCP and CLI

> **Read the spec of the parent epic xpo-4c92d4 first. It is authoritative.** It records the design decisions and the reasons for them. This spec covers only the backend part, and copies the decisions that apply here.

## What
The backend for releases: the stored model, the projection, Current/Next with placeholders, complete and delete, stats and burndown data, the server API, a `release` MCP tool, a `xpo release` CLI command, and a `release` field on issue `add`/`update`/`list`. There is no web UI in this story.

## Why
All other stories of xpo-4c92d4 (xpo-9a7a77, xpo-974eba, xpo-325f73, xpo-e6ff99) depend on this data model and API.

## Decisions copied from the epic
- **Copy-on-create, not live inheritance.** A new child copies its parent's release on create, if that release is OPEN. Later changes to the parent never change the child. Do **not** compute an "effective release" from the parent. Note: `EffectiveCycleID` is cycle *rollover*, not parent inheritance. Do not copy that pattern for releases.
- **States:** `OPEN` → `RELEASED`. This change is final. There is no reopen and no DROPPED or CANCELED state.
- **Delete:** only for OPEN releases. All issues, done issues included, move to Next or are removed from the release.
- **Complete:** open issues move to Next (default) or are removed from the release. Done issues stay.
- **Only OPEN releases accept assignments.** Assigning to a RELEASED release fails.
- **Reopen:** a reopened issue keeps its RELEASED release. Nothing moves automatically.
- **`#N` is the ID.** The number is sequential and is never reused. The label is optional and unique among OPEN releases, regardless of case. The display is the label, or `#7` if there is no label.
- **Refs:** `#7` (number), a `rel-` ID, a label, `@current`, `@next`, `none`. A number ref always starts with `#`; anything else is a label lookup. Labels may not start with `#` or `@` and may not be `none` (or start with `rel-`).
- **Current/Next** are the first and second OPEN releases in the manual order. If fewer than 2 releases are OPEN, placeholders fill the slots. A placeholder is not stored. The first write saves it.
- **Locked Next:** while Current is a placeholder, writes to the Next placeholder fail with "save Current first". The API marks that slot `locked`.
- **Leaf issue** = an issue with no children. **Scope** = leaf issues that are not CANCELED or DUPLICATE. Only scope counts in stats and burndowns.
- **started_at** is set the first time a saved release becomes Current. A reorder does not reset it.
- **Bulk moves make one feed entry** (epic decision 19).

## Decisions for this story
1. **Release events go into `issues.db`, not into a new file.** The event log already has locking, ordering, git staging, the file watcher (`server/watch.go`) and merge handling. A separate `releases.db` would need all of that wired again. Release events use new event types and IDs with a `rel-` prefix and a random hex suffix (e.g. `rel-3fa9c1`). The `#N` number is in the create payload. `ProjectIssues` ignores the new types. Alternative: a separate `releases.db`. Rejected because of the wiring cost.
2. **A new `release` MCP tool with an `operation` parameter** (`list`, `create`, `update`, `reorder`, `complete`, `delete`). This follows the `spec` and `artifact` tools. Issue membership stays on `add`/`update`/`list` as a `release` field.
3. **The final stats are a snapshot in the complete event.** `archive` moves the events of done issues to `archive.db`. Without a snapshot, a past release would lose those issues from its stats. The complete payload stores the counts by status and the leaf issue IDs of the scope.
4. **Complete and delete write all events under one events lock.** The moves of issues and the release event go in one batch. A crash cannot leave half the issues moved. Issue `add` and `update` use the same batch write, so an update and its cascades also land together.
5. **The counter is derived, not stored separately.** The next number is the highest number in any release create event, plus 1. Create events stay in the log after a delete, so numbers are never reused.
6. **`release_id` in an add/update payload may hold any ref.** It is resolved inside the locked write; the stored event always holds the `rel-` ID (or ""). The web app can send `@current`/`@next` directly. Code that writes issue events without going through `AddIssue`/`applyUpdate` must not put a ref there.
7. **Where "move to Next" sends issues:** from Current, to the old Next (the new Current); from Next, to the release after it; from a later release, to Next. A placeholder target is saved. Because Current is saved whenever a release is being completed or deleted, the target is never a locked Next.
8. **`with_children` moves all open descendants**, at any depth, whose release differs from the new one.
9. **A RELEASED release's label, goal and target date can still be edited.** Its order and state cannot change, and it cannot be deleted.
10. **Caused events.** `model.Event` gets an optional `cause`: the ID of the release (complete/delete) or the epic issue (`with_children`) whose operation wrote the event. The activity feed, the timeline and the inbox skip caused events; each issue's own history keeps them, with a note such as "when 1.5 was released". Collapse never folds events with different causes together.
11. **Bulk entries carry their counts.** The complete and delete payloads store `moved` (count) and `moved_to` (release ID, "" when removed). The epic's update stores `children_moved: {count, release_id}`; the release ID is there because the epic's own `release_id` is pruned when it is unchanged. With `with_children` and an unchanged epic release, the epic still gets an update that carries only `children_moved`.
12. **Release complete and delete are activity entries and team-wide inbox entries**, like MERGE. Release create and update are not. Their `issue_id` is the `rel-` ID; the entry carries the release display name (`issue_title`, `release`) and `moved_to`.
13. **Counts by status keep CANCELED/DUPLICATE out entirely** (counts, total, snapshot leaf IDs, burndown scope). `done` = DONE.
14. **`xpo release show <ref>`** prints a release with its stats and issues.

## Flow
1. In `internal/model/types.go`, add the event types `RELEASE_CREATE`, `RELEASE_UPDATE`, `RELEASE_COMPLETE` and `RELEASE_DELETE`, and `Event.Cause`.
2. Add `ReleaseCreatePayload` (number, label, goal, target_date, sort_order).
3. Add `ReleaseUpdatePayload` with pointer fields (label, goal, target_date, sort_order, started_at).
4. Add `ReleaseCompletePayload` (open_issues: `move_to_next` or `remove`, moved, moved_to, stats snapshot) and `ReleaseDeletePayload` (issues, moved, moved_to).
5. Add a `Release` struct: ID, Number, Label, Goal, State, SortOrder, TargetDate, StartedAt, ReleasedAt, CreatedAt, Stats snapshot, Placeholder, Locked.
6. Add `ReleaseID` to `Issue`, `CreatePayload` and `UpdatePayload`; `Children` (`children_moved`, a `ChildrenMove{count, release_id}`) to `UpdatePayload`.
7. In `internal/exponential/projection.go`, project `ReleaseID` on CREATE and UPDATE, and fill `ReleaseName`.
8. Add `ProjectReleases(events)` in a new `internal/exponential/release.go`. It returns all releases, including deleted ones, for number allocation.
9. Add `CurrentNext(releases)`. It returns two slots and fills missing slots with placeholders (`Placeholder: true`, next free numbers). Next is `Locked` when Current is a placeholder.
10. Check that `storage/collapse.go` (`applyStateEvent`, `pruneNoopUpdates`) ignores the new event types, carries `release_id` and `children_moved`, and does not fold events with different causes. Add tests.
11. Check that `storage/archive.go` never archives release events. Add a test.
12. Add `ResolveRelease(ref)`. It accepts `#7`, a `rel-` ID, a label, `@current`, `@next` and `none`. A placeholder ref returns the placeholder.
13. Add `CreateRelease`, `UpdateRelease`, `ReorderRelease`, `CompleteRelease` and `DeleteRelease` to `internal/exponential/release.go`.
14. Make every write that targets a placeholder save it first: issue assignment, label, goal, target date. A write to a locked Next fails.
15. Set `started_at` when a saved release becomes Current: on create, reorder, complete, delete or placeholder save.
16. In `internal/exponential/add.go`, copy the parent's release when it is OPEN and the caller did not pass one.
17. In `internal/exponential/update.go`, accept `release`, resolve it, and reject a RELEASED target.
18. In `update.go`, add `with_children`. It moves the open descendants too, as caused events. Done children never move.
19. In `internal/exponential/list.go`, add a `release` filter with the same refs.
20. Add `ReleaseProgress(id)`: counts by status for scope issues, and burndown points (scope, open, completed per day) from `started_at`. Use the snapshot for RELEASED releases.
21. In `internal/jsonio/input.go`, add `Release` to the add, update and list inputs. Add `WithChildren` to the update input.
22. In `internal/jsonio/output.go` and `internal/server/responses.go`, add `release_id` and `release` (the display name).
23. In `internal/mcpserver/tools.go`, register the `release` tool. Add `release` to `add`, `update` and `list`.
24. Add `cmd/exponential/release.go` with the subcommands `list`, `show`, `create`, `update`, `reorder`, `complete` and `delete`. Model it on `cmd/exponential/cycle.go`.
25. Add `--release` to `cmd/exponential/add.go`, `update.go` and `list.go`. Add `--with-children` to `update.go`.
26. In `internal/server/router.go`, add `GET /api/releases` (Current, Next with placeholders, Upcoming, Past).
27. Add `GET /api/releases/{id}/progress`, modelled on `/api/cycles/{id}/progress`.
28. Add `POST /api/releases`, `PUT /api/releases/{id}`, `POST /api/releases/reorder`, `POST /api/releases/{id}/complete` and `DELETE /api/releases/{id}`.
29. Describe release changes in the activity timeline (`internal/ui/timeline.go`, `BuildTimeline`, `xpo history`, the server activity output and the inbox): issue moves name the release; bulk operations are one entry (`IsFeedEvent`); caused events are skipped in feeds and annotated in issue histories (`ReleaseNoter`).
30. Run `make test`.

Write the tests first (TDD) for steps 7–20 and 29.

## Edge cases
- **HIGH:** Complete or delete fails halfway. Decision 4 prevents this.
- **HIGH:** A release's done issues are archived. The stats snapshot (decision 3) keeps the final stats.
- **MEDIUM:** Current is deleted while Next is a placeholder. The placeholder becomes Current, and a new placeholder becomes Next.
- **MEDIUM:** Next is deleted with `move_to_next`. The issues move to the release after Next, or to a new placeholder that the move saves.
- **MEDIUM:** Two OPEN releases get the same label. The server rejects the second one with an error.
- **MEDIUM:** A git merge of `issues.db` brings two releases with the same number. The projection keeps the number on the earlier `created_at`. It gives the later release the next free number.
- **MEDIUM:** A child is created under an epic whose release is RELEASED. The child gets no release.
- **MEDIUM:** No release is saved and a write targets `@next`. It fails with "save Current first"; nothing is stored.
- **MEDIUM:** Completing a release moves 40 open issues. The feed and inbox show one "released" entry; the 40 issue histories each show the move.
- **LOW:** An existing issue moves under a new parent. Its release does not change.
- **LOW:** A label ref matches an OPEN and a RELEASED release. The OPEN release wins.
- **LOW:** A release is labelled `2026` or `7`. That is a label; `#7` still names release number 7.

## Assumptions
- Release sort order uses `internal/sortorder` keys, like issue sort order.
- Dates are days (`YYYY-MM-DD`), like cycle IDs. Burndown days are UTC days, like cycle progress.

## Acceptance criteria
- [ ] Releases can be created, updated, reordered, completed and deleted from MCP, CLI and the server API.
- [ ] `GET /api/releases` always returns a Current and a Next. Missing slots are placeholders; Next is locked while Current is a placeholder.
- [ ] A write to a placeholder saves it. A placeholder cannot be completed, reordered or deleted. A write to a locked Next fails.
- [ ] Numbers are sequential and never reused, also after a delete.
- [ ] A duplicate label among OPEN releases fails. Labels starting with `#` or `@`, or `none`, fail.
- [ ] Complete moves open issues to Next or removes them. Done issues stay. RELEASED cannot change again.
- [ ] Delete moves all issues to Next or removes them. RELEASED releases cannot be deleted.
- [ ] Assigning an issue to a RELEASED release fails.
- [ ] A new child copies its parent's OPEN release. A later parent change does not change the child.
- [ ] `update` with `with_children` moves open descendants. Done children never move.
- [ ] `release` refs `#7`, a label, `@current`, `@next` and `none` work on add, update and list.
- [ ] Progress returns counts and a burndown from `started_at`, for scope issues only (leaf, not CANCELED/DUPLICATE). A RELEASED release uses its snapshot.
- [ ] Release events are not collapsed, pruned or archived.
- [ ] The activity timeline describes release changes. Complete, delete and `with_children` make one feed and inbox entry each.
- [ ] `make test` passes.
