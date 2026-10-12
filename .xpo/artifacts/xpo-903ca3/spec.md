# Relations sidebar overhaul: Relationships button, per-kind sections, hover preview, context menu

## What

Rework how relations are shown and edited in the issue detail sidebar (`PropertySidebar.tsx`):

1. **Relationships property button** replaces the Parent property button. It shows a flag icon and reads "No relations", "1 Relationship" or "N Relationships". Clicking it opens the add popover. It is the only way to add a relation; the Relations card loses its "+ Add relation" button and the Modal.
2. **Add popover**: step 1 is a kind menu. It shows six entries by default: Parent of…, Sub-issue of…, Related to…, Blocked by…, Blocking…, Duplicate of…. A divider and a **More ▸** submenu hold the less common wordings: Depends on…, Dependency of…, Duplicated by…. When the issue has a parent, the menu starts with **Remove parent** followed by a divider; both are hidden otherwise. Step 2 is an issue search picker in the same popover.
3. **Relations card with one section per kind**, labelled from *this* issue's point of view (if B blocks A, A shows B under **Blocked by** and B shows A under **Blocking**). The parent gets no section, because the title's "Sub-issue of" subtitle already shows it. Empty sections are hidden, and the whole card is hidden when there are no links. An icon-only button in the card's top-right corner (Lucide `Waypoints`, with tooltip "Show in dependency graph") opens the dependency graph focused on this issue (`#/dependencies/<id>`).
4. **Chips**: each related issue is a chip (status icon + title) with an `×` that appears on hover. Hovering for 250 ms opens a mini preview card.
5. **Right-click context menu** on a chip: *Change to ▸*, *Remove relation*, divider, *Copy link*.

This also delivers the original ask of this issue: change an existing link's kind in place, under the one-link-per-pair rule from xpo-c93b11.

## Why

Today the only way to change a link's kind is to remove it and add it again, because the picker hides already-linked issues. A flat list of `kind title` rows is also hard to scan. With relations grouped by kind and every action on the chip, the card works like Linear's.

## How

### Kinds and sections

Each row in `issue.dependencies` (stored **and** derived) is grouped by its `kind`, which is already from this issue's side. Section order:

| Source | Section label | Add-menu label |
|---|---|---|
| `issue.parent_id` | — (shown in the "Sub-issue of" subtitle) | Sub-issue of… |
| (sets picked issue's parent) | — (shown in the main-pane sub-issues table) | Parent of… |
| `relates_to` | Related | Related to… |
| `blocked_by` | Blocked by | Blocked by… |
| `blocks` | Blocking | Blocking… |
| `depends_on` | Depends on | More ▸ Depends on… |
| `dependency_of` | Dependency of | More ▸ Dependency of… |
| `duplicates` | Duplicate of | Duplicate of… |
| `duplicated_by` | Duplicated by | More ▸ Duplicated by… |

Neither the parent nor sub-issues get a section in the card: the subtitle shows the parent and the main-pane table lists sub-issues.

All seven link kinds can be created from the UI. `depends_on`, `dependency_of` and `duplicated_by` sit behind **More ▸** so the default menu stays short.

### Relationships button

The count is the parent (if any) plus the number of chips the card shows (each linked issue per section, so a `relates_to` stored on both sides counts once). A sub-issue with no links reads "1 Relationship".

### Add popover

- Anchored to the Relationships button. Uses the shared `Popover` and `Menu`/`MenuItem`, keyboard navigable, with no shortcut hints for now.
- Step 2 shows a search input and a candidate list (status icon, ID, title). Titles are truncated to one line. `Esc` goes back to step 1, and `Esc` again closes the popover.
- Link kinds: candidates exclude this issue and every issue already linked in either direction (stored or derived).
- Sub-issue of…: same candidates as today's parent picker (excludes self and descendants). Picking one sets `parent_id`.
- Parent of…: excludes self, ancestors and current sub-issues. Picking one sets the picked issue's `parent_id` to this issue via `saveUpdate`'s `alsoUpdate`.

### Chips

- Status icon + title. No background at rest; the chip gets a background on hover, together with the `×`. Clicking navigates to the linked issue (today it goes to the dependency graph).
- `×` appears on hover/focus and removes the link from whichever side stores it.
- A missing target renders `<id> (deleted)` and gets no preview.

### Hover preview

Opens after 250 ms of continuous hover or keyboard focus, and closes on leave. It shows the ID, assignee avatar and name, the title (truncated to one line), a divider, then status and priority. Positioning reuses `Popover`/`computePopoverPosition`, so no new dependency.

### Context menu

Built on the existing `Menu` + `SubMenu` primitives.

- **Change to ▸**: on link chips, the same layout as the add menu without the parent entries. It shows Related to, Blocked by, Blocking and Duplicate of, then **More ▸** with Depends on, Dependency of and Duplicated by. The chip's current kind is left out. Choosing one sends the full link set with the old link replaced by the new kind (the `update` "replace the set" path, so xpo-c93b11's rule holds).
- **Remove relation**: same as `×`.
- **Copy link**: copies the linked issue's web URL.

### Writes

All mutations go through `saveUpdate`. Removing or changing a *derived* link must remove it from the other issue's stored set. The HTTP update path already does this (`reconcileLinks` in `internal/exponential/update.go`), so no backend change is needed.

## Edge cases

- **Legacy multi-link pairs** (from before xpo-c93b11): the target shows in every section it has a link in. *Change to* on such a chip replaces **all** links to that target with the single chosen kind.
- A symmetric `relates_to` stored on both sides shows once.
- The popover, preview and menu close on issue navigation and on `Esc`.

## Out of scope

- Keyboard shortcuts for relations (M then R/B/X/M). These go in follow-up xpo-7f18af.

## Acceptance criteria

- [ ] The Parent property button is replaced by a flag-icon Relationships button ("No relations" / "1 Relationship" / "N Relationships") that opens the add popover
- [ ] The add popover shows six entries by default (Parent of, Sub-issue of, Related to, Blocked by, Blocking, Duplicate of) plus More ▸ (Depends on, Dependency of, Duplicated by), then an issue search; already-linked issues are excluded for link kinds
- [ ] Issue titles in the picker and the preview stay on one line (truncated)
- [ ] Sub-issue of… sets this issue's parent; Parent of… makes the picked issue a sub-issue of this one
- [ ] With a parent set, the popover's first level starts with Remove parent and a divider (both hidden otherwise); Remove parent clears the parent
- [ ] The Relations card groups links into per-kind sections from this issue's perspective (no Parent section), hiding empty sections and hiding itself when empty
- [ ] A link stored on B (B blocks A) shows on A under "Blocked by" and on B under "Blocking"
- [ ] Chips have no background until hovered; a chip's `×` (shown on hover) removes the relation, whether stored or derived
- [ ] Hovering a chip for 250 ms or more shows the preview card (ID, assignee, title, status, priority); shorter hovers don't
- [ ] Right-clicking a chip opens Change to ▸ / Remove relation / Copy link
- [ ] Change to replaces the link in place, and the issue moves to the new section on both issues
- [ ] Clicking a chip navigates to the linked issue
- [ ] The Relations card's `Waypoints` icon button opens the dependency graph focused on this issue
- [ ] Frontend tests cover grouping (including derived and legacy pairs), the change-kind link-set computation and the button label
- [ ] `make test` passes

## Decisions (agreed with user)

- Parent moves into Relationships: the Parent property button becomes the Relationships button. The parent isn't repeated as a section, because the subtitle shows it, but it counts toward the button's number, and Remove parent sits on the popover's first level (user review).
- Sub-issues stay in the main-pane table only.
- There is a single entry point for adding (the Relationships button).
- The add menu and Change to show the common wordings by default (user mockup). Depends on, Dependency of and Duplicated by sit behind a More ▸ submenu, so all link kinds stay creatable.
- Chips show status + title; the ID is in the preview.
- Review feedback: titles in the picker and preview are truncated to one line; chips have no background at rest; the Relationships button shows a count.
- Chips now open the linked issue, so the dependency graph is reached through a `Waypoints` icon button on the Relations card (user review).
- Shortcuts are a follow-up (xpo-7f18af).
