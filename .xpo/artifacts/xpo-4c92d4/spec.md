# Spec: Releases

## What / Why
Today all issues sit in one flat list. Cycles are timeboxes, and a release doesn't fit one. Label filters work, but every new release label means editing the filters. Releases become a first-class concept for grouping work that belongs together, so you can focus on it without the noise of the rest of the backlog.

Releases are **scope-bounded, not time-bounded**. A target date is optional and only for display. Releases don't gate starting work and aren't tied to building binaries or git tags.

## Model
- **Release:** id, name (a version like "1.5" or free text like "Team Mode"), goal (one line), state `OPEN` → `RELEASED` | `DROPPED`, order among open releases, optional target date, released_at.
- **Issue:** has at most one `release_id`. Epics pass their release to their children, and a child can override it (same as `CycleID`/`EffectiveCycleID`).
- **Current / Next:** the first and second OPEN releases in order. Completing Current makes Next the new Current automatically.

## Surfaces
| Story | Surface |
|---|---|
| xpo-f89e55 | Data model, server API, MCP + CLI (`release` on add/update/list, `@current`/`@next`/`none`) |
| xpo-9a7a77 | Sidebar "Releases" entry and list view: Current/Next cards with sparklines and stats, Upcoming (reorderable), Past timeline |
| xpo-e6ff99 | Release detail: header, stats, embedded backlog, right toolbar with Complete release as the primary button |
| xpo-974eba | Assignment: property picker, context menu (multi-select), `R` shortcut, drag onto a release in the sidebar |
| xpo-325f73 | `Current (x)` / `Next (y)` tabs in Issues, relative filter tokens |

## Decisions
1. **Current/Next come from the order of open releases**, not from pins or dates.
2. **The Current/Next tabs show the full scope**, grouped by status, with the Done group collapsed.
3. **Epics pass their release to their children**, and a child can override it.
4. **A release has only a short goal line** for now. A full markdown release doc is out of scope.
5. **Completing a release with open issues:** the dialog defaults to moving them to Next, with "Remove from release" as the alternative. Done issues stay as the release's record.
6. **Not the Pivotal approach:** no cut-line markers in the backlog. Membership is an explicit property.

## Out of scope
- Git tag integration (auto-assigning merged issues to tags), agent-drafted release notes, release gating via links. These are possible follow-ups.

## Acceptance criteria
- [ ] Releases can be created, ordered, completed and dropped from the web app, MCP and CLI.
- [ ] Issues can be assigned to a release from the picker, the context menu (bulk), the `R` shortcut and drag. Children inherit from epics.
- [ ] The Releases view lists Current, Next, Upcoming and Past releases with sparklines and stats.
- [ ] The release detail page shows stats and an embedded backlog, and completes a release with the move-to-Next dialog.
- [ ] The Current/Next tabs appear only when those releases exist and have issues. `release:@current`/`@next`/`none` filters work.
- [ ] `make test` passes.
