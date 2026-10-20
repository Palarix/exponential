# Spec: Releases

> Each story has its own spec with concrete Flow steps. This epic spec is authoritative for design decisions. Where a story spec and this spec disagree, this spec wins. Update both.

## What / Why
Today all issues sit in one flat list. Cycles are timeboxes, and a release doesn't fit one. Label filters work, but every new release label means editing the filters. Releases become a first-class concept for grouping work that belongs together, so you can focus on it without the noise of the rest of the backlog.

Releases are **scope-bounded, not time-bounded**. A target date is optional and only for display. Releases don't gate starting work and aren't tied to building binaries or git tags.

## Model
- **Release:** number (`#N`, sequential, the stable ID), optional label (a version like "1.5.0" or free text like "Team Mode"), goal (one line), state, order among open releases, optional target date, started_at, released_at.
- **State:** `OPEN` → `RELEASED`. This change is final. There is no reopen and no third state.
- **Delete:** an OPEN release can be deleted. A RELEASED release cannot be deleted.
- **Issue:** has at most one `release_id`. Every issue stores its own value. There is no live inheritance from the epic.
- **Copy-on-create:** a new child copies the release of its parent when it is created. After that, the child's release is independent of the parent.
- **Leaf issue:** an issue with no children. Only leaf issues count in stats and burndowns.
- **Scope:** the leaf issues of a release that are not CANCELED or DUPLICATE. Canceled and duplicate issues leave the scope, so a release whose remaining issues are done reads as complete.
- **Current / Next:** the first and second OPEN releases in order. Completing or deleting Current makes Next the new Current automatically.
- **started_at:** the board sets it the first time a saved release becomes Current. A later reorder does not reset it.
- **Placeholders:** Current and Next always exist. If fewer than 2 releases are OPEN, the board shows placeholder slots with the next free numbers. A placeholder is not stored. The board saves it on the first write: an issue assignment, a label, a goal or a target date.
- **Locked Next:** while Current is a placeholder, the Next placeholder is locked. Writes to it fail with "save Current first", and the UI shows it disabled. This keeps an empty Current from being created as a side effect of filling Next.

## Releases and cycles
Releases and cycles are independent. An issue can be in a cycle and in a release at the same time. Example: release "Team Mode" targets Q3, and each cycle before Q3 delivers part of its scope.

## Surfaces
| Story | Surface |
|---|---|
| xpo-f89e55 | Data model, copy-on-create, placeholders, complete and delete, stats snapshot, server API, `release` MCP tool + `xpo release` CLI, `release` on add/update/list (`#7`/label/`@current`/`@next`/`none`) |
| xpo-9a7a77 | Sidebar "Releases" entry (no sub-items), Releases view: one column of release rows (Create release, Upcoming, Next, Current, Past, No release) with progress and order warnings, reorder by drag, expandable releases with issue drag and drop, top bar issue search |
| xpo-974eba | Assignment: property picker (OPEN releases only), context menu (multi-select), `R` shortcut, "also move open children?" prompt (also on drops in the Releases view), reopen hint, release chips on rows and cards |
| xpo-325f73 | Release filter in Issues (`@current`/`@next`/`none`/`#N`, resolved when the filter runs); clicking an open release opens Issues with that filter |
| xpo-88f633 | Edit, Complete and Delete on open release rows in the Releases view, with the confirm dialogs |
| xpo-e6ff99 | Past releases expand in place (no detail page): the burndown, then the delivered issues |
| xpo-574cf5 | Current release section on the Dashboard: progress, target date, burndown |
| xpo-f5e7db | (Backlog) Retrospective notes at the very bottom of an expanded past release, below its issues |
| xpo-6178dc | (Related, outside the epic) Custom views: saved filters as tabs, e.g. "Release: Current" |

## Decisions
1. **Current/Next come from the order of open releases**, not from pins or dates.
2. **Open releases are worked on in the Issues view, through a Release filter, not on their own page.** Clicking an open release opens Issues on All Issues with that release filter, the way clicking a label in the Labels view does. `@current` and `@next` resolve each time the filter runs, so a saved filter (and later a custom view tab, xpo-6178dc) follows Current and Next as releases complete. **No release has a detail page.** A past release is read in place: clicking it expands its row into the release's record, top to bottom: the burndown, the issues it delivered, and later the retrospective notes (xpo-f5e7db). Its progress popover shows the final stats. A first detail page for past releases (xpo-e6ff99) was built and dropped: it showed nothing the Releases view doesn't. A page comes back only if it tells the release's story (scope changes, timing, release notes). The Current release's burndown also shows on the Dashboard. Open releases are edited, completed and deleted from their rows in the Releases view. (An earlier draft had built-in `Current`/`Next` tabs in Issues and a detail page with an embedded backlog for every release. That duplicated the Issues view, and custom views cover the tabs more generally.)
3. **Copy-on-create, not live inheritance.** This follows Linear: a sub-issue copies the parent's project and cycle when it is created. Jira does not inherit fix versions at all. Live inheritance caused two problems. Moving an epic also moved its done children out of a completed release. New children of an epic in a RELEASED release joined a frozen release. With copy-on-create, both problems go away. Cycles in xpo have no parent inheritance either. `EffectiveCycleID` is cycle *rollover* (unfinished issues in a past cycle roll to the current cycle), so do not copy it for releases.
4. **Changing an epic's release asks about its children.** In the web app, a prompt asks "Also move its N open children?" Done children never move. MCP and CLI change only the epic, unless the caller passes a `with_children` option. "Children" means all open descendants, at any depth.
5. **Completing a release with open issues:** the dialog defaults to moving them to Next, with "Remove from release" as the alternative. Done issues stay as the release's record. If Next is a placeholder, the move saves it.
6. **Not the Pivotal approach:** no cut-line markers in the backlog. Membership is an explicit property.
7. **`#N` is the ID, the label is optional.** Releases get sequential numbers like cycles. The UI shows the label, or `#7` if there is no label. The CLI and MCP accept `#7`, a label, `@current`, `@next` and `none`. A number ref always starts with `#`; anything else is a label, so labels like `2026` or `7` are allowed. Labels may not start with `#` or `@`, and may not be `none`. Labels are unique among OPEN releases, regardless of case. Jira and Linear show versions and projects by name only. Release numbers only appear in release contexts, so `#7` does not clash with cycle `#7` in practice.
8. **Current and Next are always present, as placeholders if needed.** Placeholders are not stored, so the board does not fill with empty releases. Placeholders also give `@next` and "Move to Next" a target at all times. A placeholder cannot be completed, reordered or deleted. Placeholders always sit after the saved OPEN releases. While Current is a placeholder, Next is locked (see Model).
9. **Completing is final and needs confirmation.** The complete dialog is the confirm step. There is no reopen. Jira allows "unrelease". We do not.
10. **Delete, not Drop or Cancel.** Done issues are already merged to main, so the work of an abandoned release still ships with a later release. A cancelled state that keeps done issues would record a loss that did not happen. So an abandoned release is deleted. The delete dialog moves all its issues, done issues included, to Next (default) or removes them from the release. This matches Jira's delete-version page. The issues' activity events keep the trace. Past shows only releases that shipped.
11. **Stats count leaf issues only, and only the scope.** The parent itself is never counted. This prevents double counting when children are in different releases. Leaf means "has no children", not "has no epic label", so a childless epic counts. CANCELED and DUPLICATE issues are not counted. **Release issue lists are flat:** each list shows only the issues that are in it, as Jira's version page and Linear's lists do. A sub-issue's row shows its parent as a muted "Parent ›" breadcrumb, whose tooltip says where the parent is ("in 1.2", "no release"); a parent's row carries its sub-issue progress. A parent never shows as a stand-in in a list it is not in. (An earlier draft grouped children under a parent header in every release holding them; it hid where the parent actually was.)
12. **Burndown = open scope over time, from started_at.** The line goes up when scope is added mid-release, and goes down when an issue is done, canceled or marked duplicate. Linear's project graph works the same way, and it starts when a project moves to "Started". Becoming Current is our equivalent. Releases that were never Current show stats but no burndown.
13. **Order warning:** if target dates contradict the manual order of open releases, the Releases view shows a warning badge on the affected releases. The board does not reorder them.
14. **Only OPEN releases accept assignments.** The picker, context menu, drag targets, MCP and CLI reject RELEASED releases. Jira allows this by default, and its users have asked for years to hide released versions.
15. **A reopened issue stays in its RELEASED release.** This matches Jira, where nothing moves on reopen. The issue detail shows a hint: "In released release 1.5. Move to Current?" The hint has a one-click action. Nothing moves automatically.
16. **Drag targets live in the Releases view, not the sidebar.** No other sidebar entry has sub-items, so "Releases" is a plain entry. In the Releases view, the open releases expand to list their issues, and a "No release" row at the bottom lists open issues with no release ("Unassigned" would read as "no assignee"). Issues drag between them; dropping on No release removes the release. Past releases accept no drops. The `R` shortcut stays the main path.
17. **Completing saves a stats snapshot.** `xpo archive` moves the events of done issues to `archive.db`. Without a snapshot, a past release would lose those issues from its stats. The complete event stores the final counts and the leaf issue IDs.
18. **Release events live in `issues.db`.** New event types with `rel-` IDs reuse the existing locking, ordering, git staging, file watcher and merge handling. A separate file would need all of that again. Details are in the xpo-f89e55 spec.
19. **Bulk moves make one feed entry.** Completing or deleting a release, and changing an epic's release with its children, can move many issues at once. The activity feed and the inbox show one entry for the whole operation, such as "released 1.5 · moved 12 open issues to #3" or "moved this issue and 10 open children to 1.5". The per-issue moves stay in each issue's own history, with the operation that caused them.

## Edge cases
- **HIGH:** A user completes or deletes a release by accident. Both need a confirm dialog, and neither can be undone.
- **HIGH:** A completed release's done issues are archived. The stats snapshot keeps its final stats.
- **MEDIUM:** A user deletes Current while Next is a placeholder. The placeholder becomes Current, and a new placeholder becomes Next.
- **MEDIUM:** A user deletes Next and picks "Move to Next". The issues move to the release after it, or to a new placeholder that the move saves.
- **MEDIUM:** Two OPEN releases get the same label. The server rejects the second one.
- **MEDIUM:** A git merge brings two releases with the same number. The earlier release keeps the number. The later one gets the next free number.
- **MEDIUM:** A user creates a child under an epic whose release is RELEASED. The child gets no release, because only OPEN releases are copied.
- **MEDIUM:** A reopened issue sits in a RELEASED release. It counts in no OPEN release until the user moves it. The `release:@current` filter does not show it.
- **MEDIUM:** The board has no saved release and a user tries to put an issue in Next. The write fails with "save Current first"; nothing is saved.
- **MEDIUM:** A saved filter uses `@current` and Current is completed. The filter shows the new Current with no edit.
- **LOW:** A user moves an existing issue under a new parent. The issue keeps its own release. Copy-on-create applies only to new issues.
- **LOW:** The board never reuses the number of a deleted release.
- **LOW:** A placeholder becomes Current and is saved later. started_at is the time of the save.

## Out of scope
- Git tag integration (auto-assigning merged issues to tags), agent-drafted release notes, release gating via links. These are possible follow-ups.
- Reopening a RELEASED release.
- Velocity-based completion forecasts on the burndown.
- Built-in Current/Next tabs in Issues. Custom views (xpo-6178dc) cover them.
- A release detail page (scope changes, timing, release notes). Revisited if it becomes an issue.

## Acceptance criteria
- [ ] Releases can be created, ordered, completed and deleted from the web app, MCP and CLI.
- [ ] Complete and Delete show a confirm dialog. A RELEASED release cannot change state or be deleted.
- [ ] Deleting a release moves all its issues to Next or removes them from the release, as the user picks.
- [ ] Current and Next always exist. A placeholder is saved only on first write. A placeholder cannot be completed, reordered or deleted. Next is locked while Current is a placeholder.
- [ ] Releases have a sequential `#N` that is never reused, and an optional label that is unique among OPEN releases. The UI shows the label or `#7`.
- [ ] Issues can be assigned to an OPEN release from the picker, the context menu (bulk), the `R` shortcut and drag in the Releases view. Assigning to a RELEASED release fails.
- [ ] A new child copies its parent's OPEN release on create. Later changes to the parent do not change the child.
- [ ] Changing an epic's release in the web app asks whether to move its open children. Done children never move.
- [ ] Stats and burndowns count only issues with no children, and leave out CANCELED and DUPLICATE issues. Release issue lists are flat; a sub-issue shows its parent in a breadcrumb that says where the parent is.
- [ ] The burndown starts at started_at. A RELEASED release keeps its final stats after `xpo archive`.
- [ ] Completing or deleting a release, or moving an epic with its children, makes one entry in the activity feed and inbox.
- [ ] A reopened issue in a RELEASED release stays there, and its detail page offers "Move to Current".
- [ ] The Releases view lists Current, Next, Upcoming and Past releases with burndowns and stats. It shows a warning badge when target dates contradict the order.
- [ ] The Issues view has a Release filter (`@current`/`@next`/`none`/each release) that follows Current and Next as they change. Clicking an open release opens Issues with that filter.
- [ ] Open releases can be edited, completed and deleted from the Releases view.
- [ ] Past releases expand in place to their burndown and delivered issues; their progress popover shows the final stats.
- [ ] The Dashboard shows the Current release's progress and burndown.
- [ ] `make test` passes.
