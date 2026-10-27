# Spec: Milestones

> Each story has its own spec with concrete Flow steps. This epic spec is authoritative for design decisions. Where a story spec and this spec disagree, this spec wins. Update both.
>
> The feature was built as "Releases" and renamed to "Milestones" in xpo-d12b64 (see Decision 20). Story specs written before the rename still say "release".

## What / Why
Today all issues sit in one flat list. Cycles are timeboxes, and a milestone doesn't fit one. Label filters work, but every new milestone label means editing the filters. Milestones become a first-class concept for grouping work that belongs together, so you can focus on it without the noise of the rest of the backlog.

Milestones are **scope-bounded, not time-bounded**. A target date is optional and only for display. Milestones don't gate starting work and aren't tied to building binaries or git tags.

## Model
- **Milestone:** number (`#N`, sequential, the stable ID), optional label (a version like "1.5.0" or free text like "Team Mode"), goal (one line), state, order among open milestones, optional target date, started_at, completed_at.
- **State:** `OPEN` → `COMPLETED`. This change is final. There is no reopen and no third state.
- **Delete:** an OPEN milestone can be deleted. A COMPLETED milestone cannot be deleted.
- **Issue:** has at most one `milestone_id`. Every issue stores its own value. There is no live inheritance from the epic.
- **Copy-on-create:** a new child copies the milestone of its parent when it is created. After that, the child's milestone is independent of the parent.
- **Leaf issue:** an issue with no children. Only leaf issues count in stats and burndowns.
- **Scope:** the leaf issues of a milestone that are not CANCELED or DUPLICATE. Canceled and duplicate issues leave the scope, so a milestone whose remaining issues are done reads as complete.
- **Current / Next:** the first and second OPEN milestones in order. Completing or deleting Current makes Next the new Current automatically.
- **started_at:** the board sets it the first time a saved milestone becomes Current. A later reorder does not reset it.
- **Placeholders:** Current and Next always exist. If fewer than 2 milestones are OPEN, the board shows placeholder slots with the next free numbers. A placeholder is not stored. The board saves it on the first write: an issue assignment, a label, a goal or a target date.
- **Locked Next:** while Current is a placeholder, the Next placeholder is locked. Writes to it fail with "save Current first", and the UI shows it disabled. This keeps an empty Current from being created as a side effect of filling Next.

## Milestones and cycles
Milestones and cycles are independent. An issue can be in a cycle and in a milestone at the same time. Example: milestone "Team Mode" targets Q3, and each cycle before Q3 delivers part of its scope.

## Surfaces
| Story | Surface |
|---|---|
| xpo-f89e55 | Data model, copy-on-create, placeholders, complete and delete, stats snapshot, server API, `milestone` MCP tool + `xpo milestone` CLI, `milestone` on add/update/list (`#7`/label/`@current`/`@next`/`none`) |
| xpo-9a7a77 | Sidebar "Milestones" entry (no sub-items), Milestones view: one column of milestone rows (Create milestone, Upcoming, Next, Current, Completed, No milestone) with progress and order warnings, reorder by drag, expandable milestones with issue drag and drop, top bar issue search |
| xpo-974eba | Assignment: property picker (OPEN milestones only), context menu (multi-select), `r` shortcut, "also move open children?" prompt (also on drops in the Milestones view), reopen hint, milestone chips on rows and cards |
| xpo-325f73 | Milestone filter in Issues (`@current`/`@next`/`none`/`#N`, resolved when the filter runs); clicking an open milestone opens Issues with that filter |
| xpo-88f633 | Edit, Complete and Delete on open milestone rows in the Milestones view, with the confirm dialogs |
| xpo-e6ff99 | Completed milestones expand in place (no detail page): the burndown, then the delivered issues |
| xpo-d12b64 | Rename Releases to Milestones in every surface; `xpo doctor --fix` migrates existing boards once |
| xpo-574cf5 | Current milestone section on the Dashboard: progress, target date, burndown |
| xpo-f5e7db | Retrospective notes at the very bottom of an expanded completed milestone, below its issues |
| xpo-8995e4 | UI/UX polish of the Milestones view |
| xpo-6178dc | (Related, outside the epic) Custom views: saved filters as tabs, e.g. "Milestone: Current" |

## Decisions
1. **Current/Next come from the order of open milestones**, not from pins or dates.
2. **Open milestones are worked on in the Issues view, through a Milestone filter, not on their own page.** Clicking an open milestone opens Issues on All Issues with that milestone filter, the way clicking a label in the Labels view does. `@current` and `@next` resolve each time the filter runs, so a saved filter (and later a custom view tab, xpo-6178dc) follows Current and Next as milestones complete. **No milestone has a detail page.** A completed milestone is read in place: clicking it expands its row into the milestone's record, top to bottom: the burndown, the issues it delivered, and later the retrospective notes (xpo-f5e7db). Its progress popover shows the final stats. A first detail page for completed milestones (xpo-e6ff99) was built and dropped: it showed nothing the Milestones view doesn't. A page comes back only if it tells the milestone's story (scope changes, timing, notes). The Current milestone's burndown also shows on the Dashboard. Open milestones are edited, completed and deleted from their rows in the Milestones view. (An earlier draft had built-in `Current`/`Next` tabs in Issues and a detail page with an embedded backlog for every milestone. That duplicated the Issues view, and custom views cover the tabs more generally.)
3. **Copy-on-create, not live inheritance.** This follows Linear: a sub-issue copies the parent's project and cycle when it is created. Jira does not inherit fix versions at all. Live inheritance caused two problems. Moving an epic also moved its done children out of a completed milestone. New children of an epic in a COMPLETED milestone joined a frozen milestone. With copy-on-create, both problems go away. Cycles in xpo have no parent inheritance either. `EffectiveCycleID` is cycle *rollover* (unfinished issues in a past cycle roll to the current cycle), so do not copy it for milestones.
4. **Changing an epic's milestone asks about its children.** In the web app, a prompt asks "Also move its N open children?" Done children never move. MCP and CLI change only the epic, unless the caller passes a `with_children` option. "Children" means all open descendants, at any depth.
5. **Completing a milestone with open issues:** the dialog defaults to moving them to the next open milestone, with "No milestone" as the alternative. Done issues stay as the milestone's record. If the target is a placeholder, the move saves it.
6. **Not the Pivotal approach:** no cut-line markers in the backlog. Membership is an explicit property.
7. **`#N` is the ID, the label is optional.** Milestones get sequential numbers like cycles. The UI shows the label, or `#7` if there is no label. The CLI and MCP accept `#7`, a label, `@current`, `@next` and `none`. A number ref always starts with `#`; anything else is a label, so labels like `2026` or `7` are allowed. Labels may not start with `#` or `@`, and may not be `none`. Labels are unique among OPEN milestones, regardless of case. Jira and Linear show versions and projects by name only. Milestone numbers only appear in milestone contexts, so `#7` does not clash with cycle `#7` in practice.
8. **Current and Next are always present, as placeholders if needed.** Placeholders are not stored, so the board does not fill with empty milestones. Placeholders also give `@next` and "Move to Next" a target at all times. A placeholder cannot be completed, reordered or deleted. Placeholders always sit after the saved OPEN milestones. While Current is a placeholder, Next is locked (see Model).
9. **Completing is final and needs confirmation.** The complete dialog is the confirm step. There is no reopen. Jira allows "unrelease". We do not.
10. **Delete, not Drop or Cancel.** Done issues are already merged to main, so the work of an abandoned milestone still lands with a later one. A cancelled state that keeps done issues would record a loss that did not happen. So an abandoned milestone is deleted. The delete dialog moves all its issues, done issues included, to the next open milestone (default) or to No milestone. This matches Jira's delete-version page. The issues' activity events keep the trace. Completed shows only milestones that were completed.
11. **Stats count leaf issues only, and only the scope.** The parent itself is never counted. This prevents double counting when children are in different milestones. Leaf means "has no children", not "has no epic label", so a childless epic counts. CANCELED and DUPLICATE issues are not counted. **Milestone issue lists are flat:** each list shows only the issues that are in it, as Jira's version page and Linear's lists do. A sub-issue's row shows its parent as a muted "Parent ›" breadcrumb, whose tooltip says where the parent is ("in 1.2", "no milestone"); a parent's row carries its sub-issue progress. A parent never shows as a stand-in in a list it is not in. (An earlier draft grouped children under a parent header in every milestone holding them; it hid where the parent actually was.)
12. **Burndown = open scope over time, from started_at.** The line goes up when scope is added mid-milestone, and goes down when an issue is done, canceled or marked duplicate. Linear's project graph works the same way, and it starts when a project moves to "Started". Becoming Current is our equivalent. Milestones that were never Current show stats but no burndown.
13. **Order warning:** if target dates contradict the manual order of open milestones, the Milestones view shows a warning badge on the affected milestones. The board does not reorder them.
14. **Only OPEN milestones accept assignments.** The picker, context menu, drag targets, MCP and CLI reject COMPLETED milestones. Jira allows this by default, and its users have asked for years to hide released versions.
15. **A reopened issue stays in its COMPLETED milestone.** This matches Jira, where nothing moves on reopen. The issue detail shows a hint: "In completed milestone 1.5. Move to Current?" The hint has a one-click action. Nothing moves automatically.
16. **Drag targets live in the Milestones view, not the sidebar.** No other sidebar entry has sub-items, so "Milestones" is a plain entry. In the Milestones view, the open milestones expand to list their issues, and a "No milestone" row at the bottom lists open issues with no milestone ("Unassigned" would read as "no assignee"). Issues drag between them; dropping on No milestone removes the milestone. Completed milestones accept no drops. The `r` shortcut stays the main path.
17. **Completing saves a stats snapshot.** `xpo archive` moves the events of done issues to `archive.db`. Without a snapshot, a completed milestone would lose those issues from its stats. The complete event stores the final counts, points and the leaf issue IDs.
18. **Milestone events live in `issues.db`.** Event types `MILESTONE_*` with `ms-` IDs reuse the existing locking, ordering, git staging, file watcher and merge handling. A separate file would need all of that again. Details are in the xpo-f89e55 spec.
19. **Bulk moves make one feed entry.** Completing or deleting a milestone, and changing an epic's milestone with its children, can move many issues at once. The activity feed and the inbox show one entry for the whole operation, such as "completed 1.5 · moved 12 open issues to #3" or "moved this issue and 10 open children to 1.5". The per-issue moves stay in each issue's own history, with the operation that caused them.
20. **Milestones, not Releases.** xpo groups work that belongs together. It does not know whether that work becomes a public build, image or deploy. "Release" claims something xpo does not track. "Milestone" pairs with Cycles (scope-boxed vs time-boxed), and keeps "Release" free for a later feature that ties a completed milestone to a shipped version. Labels name the theme ("Better Agent Chat"); the version is decided when the milestone ships. Boards with data from before the rename migrate once with `xpo doctor --fix` (xpo-d12b64).

## Edge cases
- **HIGH:** A user completes or deletes a milestone by accident. Both need a confirm dialog, and neither can be undone.
- **HIGH:** A completed milestone's done issues are archived. The stats snapshot keeps its final stats.
- **HIGH:** A board has data from before the rename. Reads fail with a message to run `xpo doctor --fix`, which migrates it (xpo-d12b64).
- **MEDIUM:** A user deletes Current while Next is a placeholder. The placeholder becomes Current, and a new placeholder becomes Next.
- **MEDIUM:** A user deletes Next and keeps the default target. The issues move to the milestone after it, else the one before it. A new milestone is created only when no other open milestone exists.
- **MEDIUM:** Two OPEN milestones get the same label. The server rejects the second one.
- **MEDIUM:** A git merge brings two milestones with the same number. The earlier milestone keeps the number. The later one gets the next free number.
- **MEDIUM:** A user creates a child under an epic whose milestone is COMPLETED. The child gets no milestone, because only OPEN milestones are copied.
- **MEDIUM:** A reopened issue sits in a COMPLETED milestone. It counts in no OPEN milestone until the user moves it. The `milestone:@current` filter does not show it.
- **MEDIUM:** The board has no saved milestone and a user tries to put an issue in Next. The write fails with "save Current first"; nothing is saved.
- **MEDIUM:** A saved filter uses `@current` and Current is completed. The filter shows the new Current with no edit.
- **LOW:** A user moves an existing issue under a new parent. The issue keeps its own milestone. Copy-on-create applies only to new issues.
- **LOW:** The board never reuses the number of a deleted milestone.
- **LOW:** A placeholder becomes Current and is saved later. started_at is the time of the save.

## Out of scope
- Git tag integration (auto-assigning merged issues to tags, recording the shipped version on complete), agent-drafted release notes, gating via links. These are possible follow-ups.
- Reopening a COMPLETED milestone.
- Velocity-based completion forecasts on the burndown.
- Built-in Current/Next tabs in Issues. Custom views (xpo-6178dc) cover them.
- A milestone detail page (scope changes, timing, notes). Revisited if it becomes an issue.

## Acceptance criteria
- [ ] Milestones can be created, ordered, completed and deleted from the web app, MCP and CLI.
- [ ] Complete and Delete show a confirm dialog. A COMPLETED milestone cannot change state or be deleted.
- [ ] Deleting a milestone moves all its issues to another open milestone or to No milestone, as the user picks.
- [ ] Current and Next always exist. A placeholder is saved only on first write. A placeholder cannot be completed, reordered or deleted. Next is locked while Current is a placeholder.
- [ ] Milestones have a sequential `#N` that is never reused, and an optional label that is unique among OPEN milestones. The UI shows the label or `#7`.
- [ ] Issues can be assigned to an OPEN milestone from the picker, the context menu (bulk), the `r` shortcut and drag in the Milestones view. Assigning to a COMPLETED milestone fails.
- [ ] A new child copies its parent's OPEN milestone on create. Later changes to the parent do not change the child.
- [ ] Changing an epic's milestone in the web app asks whether to move its open children. Done children never move.
- [ ] Stats and burndowns count only issues with no children, and leave out CANCELED and DUPLICATE issues. Milestone issue lists are flat; a sub-issue shows its parent in a breadcrumb that says where the parent is.
- [ ] The burndown starts at started_at. A COMPLETED milestone keeps its final stats after `xpo archive`.
- [ ] Completing or deleting a milestone, or moving an epic with its children, makes one entry in the activity feed and inbox.
- [ ] A reopened issue in a COMPLETED milestone stays there, and its detail page offers "Move to Current".
- [ ] The Milestones view lists Current, Next, Upcoming and Completed milestones with burndowns and stats. It shows a warning badge when target dates contradict the order.
- [ ] The Issues view has a Milestone filter (`@current`/`@next`/`none`/each milestone) that follows Current and Next as they change. Clicking an open milestone opens Issues with that filter.
- [ ] Open milestones can be edited, completed and deleted from the Milestones view.
- [ ] Completed milestones expand in place to their burndown and delivered issues; their progress popover shows the final stats.
- [ ] No surface uses "release" for this feature, and boards from before the rename migrate with `xpo doctor --fix`.
- [ ] The Dashboard shows the Current milestone's progress and burndown.
- [ ] `make test` passes.
