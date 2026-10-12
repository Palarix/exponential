# Walkthrough: Relations sidebar overhaul

## What was built

The issue detail sidebar used to show relations as a flat list of `kind title` rows. You could only add one through a full-screen Modal, and the only way to change a link's kind was to remove it and add it again. Now:

- **The Parent property row became a Relationships button.** It shows a flag icon and reads "No relations", "1 Relationship" or "N Relationships", where the parent counts as one. It opens the add popover, which is now the only way to add a relation.
- **The add popover is a two-step menu.** Step 1 is the kind menu:
  - "Remove parent" and a divider come first, only when the issue has a parent.
  - Then Parent of…, Sub-issue of…, Related to…, Blocked by…, Blocking…, Duplicate of….
  - Then a divider and More ▸, which holds Depends on…, Dependency of… and Duplicated by….

  Step 2 is an issue search in the same popover.
- **The Relations card groups links into one section per kind**, worded from the issue you're viewing (B blocks A shows on A under "Blocked by" and on B under "Blocking"). The parent has no section because the "Sub-issue of" subtitle already shows it. A `Waypoints` icon button in the top-right opens the dependency graph focused on this issue.
- **Each linked issue is a chip.** It has no background until hovered; hovering shows a background and a `×`. After 250 ms of hover a preview card appears (ID, assignee, title, status, priority). Right-click opens Change to ▸ / Remove relation / Copy link. Clicking a chip opens that issue.

## How the pieces fit

```
PropertySidebar.tsx
 ├─ Relationships button ── relationshipsLabel()          (relation-utils.ts)
 │   └─ RelationsAddPopover ─ linkCandidates / subIssueOfCandidates / parentOfCandidates, addLink
 └─ RelationsCard ───────── groupRelations()
     └─ RelationChip ─ createHoverIntent() (hover-intent.ts) → IssuePreview
                      └─ RelationContextMenu ─ changeToKinds(), changeLinkKind(), removeLink()
```

- **`relation-utils.ts`** holds all the logic as pure functions, unit-tested in `relation-utils.test.ts`:
  - `groupRelations` dedupes per (kind, target), so a `relates_to` stored on both sides shows once, and a legacy pair with two kinds shows in both sections.
  - `changeLinkKind` drops *every* row to the target and puts one new row in place of the first. This is how legacy multi-link pairs collapse under the one-link-per-pair rule (xpo-c93b11).
- **`hover-intent.ts`** is a tiny timer controller, so the 250 ms behaviour can be tested with fake timers.
- **`Relations.tsx`** holds the components; they render and dispatch, and keep no rules of their own.

## Writes and links stored on the other issue

Every mutation goes through `saveUpdate({ dependencies })` and sends the issue's **full** link set, including derived rows (links stored on the other issue). No backend change was needed. `reconcileLinks` in `internal/exponential/update.go` already treats the payload as the desired set:

- A derived row that's still present stays on the other issue.
- A derived row that's missing is removed from the other issue.

So `×` on a "Dependency of" chip, or Change to on a derived link, updates the other issue correctly. I verified this through the API in the browser test.

## Non-obvious decisions

- **The two popover steps are separate `<Menu key=…>` instances.** The keyboard registry gives Escape to the most recently *registered* overlay. A single Menu instance reused across both steps registers before the Popover, so the Popover's Escape won and closed everything. Keying the steps remounts the Menu, so step 2's Escape goes back to step 1 and step 1's Escape closes the popover.
- **The `choosing` ref in `RelationsAddPopover`.** `Menu` calls `onClose` after every item click. Picking a kind must move to step 2, not close the popover, so `closeKindMenu` swallows that one close. An effect resets the ref on every kind change, because picks from the More flyout never call the parent's `onClose`.
- **`Popover` ignores mousedowns inside `[data-submenu-flyout]`.** Flyouts are portaled outside the popover's DOM, so without this, clicking a More ▸ entry counted as an outside click.
- **The preview and context menu render as *siblings* of the chip.** React events bubble through portals, so as children they would retrigger the chip's hover handlers.
- **`MenuItem` gains an optional `truncate` prop** for one-line titles in the picker. The issue ID sits in the icon slot because the suffix slot is a fixed 16px.
- **`relation-utils.ts` is not named `relations.ts`.** On the case-insensitive macOS filesystem that collides with `Relations.tsx`.
- **The initial menu focus lands on Remove parent for sub-issues.** It's the first item and `Menu` auto-focuses the first one. The user accepted this as is.

## Iterations during review

1. The add menu shows the common wordings by default; the reverse kinds moved behind More ▸, not removed.
2. Titles are truncated to one line, and chips have no background at rest.
3. The Relationships button shows a count.
4. Since chips now open the issue, a graph button (`Waypoints`) went on the Relations card.
5. The Parent section was dropped (the subtitle shows it). Remove parent moved first into the Sub-issue of picker, then to the top of the first-level menu, and the parent counts toward the button number.

## Acceptance criteria

- [x] The Parent property button is replaced by a flag-icon Relationships button ("No relations" / "1 Relationship" / "N Relationships") that opens the add popover. Evidence: upe-00007d read "1 Relationship", upe-000016 "3 Relationships", and after removing the parent, "No relations".
- [x] The add popover shows six entries by default plus More ▸ (Depends on, Dependency of, Duplicated by), then an issue search; already-linked issues are excluded for link kinds. Evidence: the DOM dump of the menu; `linkCandidates` test.
- [x] Issue titles in the picker and the preview stay on one line (truncated). Evidence: browser screenshot with "Integrate Stripe payment processi…".
- [x] Sub-issue of… sets this issue's parent; Parent of… makes the picked issue a sub-issue of this one. Evidence: upe-00007d's `parent_id` became upe-000009 via Parent of…; candidate tests.
- [x] With a parent set, the popover's first level starts with Remove parent and a divider (both hidden otherwise); Remove parent clears the parent. Evidence: the menu order on upe-000016; on upe-00007d the subtitle disappeared after the click.
- [x] The Relations card groups links into per-kind sections from this issue's perspective (no Parent section), hiding empty sections and hiding itself when empty. Evidence: `groupRelations` tests; on upe-00007d (parent only) no card was rendered.
- [x] A link stored on B (B blocks A) shows on A under "Blocked by" and on B under "Blocking". Evidence: the "shows a link stored on the other issue under its inverse wording" test; after Change to, the API showed `upe-000008 blocks upe-000009 (derived)`.
- [x] Chips have no background until hovered; a chip's `×` (shown on hover) removes the relation, whether stored or derived. Evidence: the screenshot; `×` on derived "Dependency of upe-00000c" left both issues without the link.
- [x] Hovering a chip for 250 ms or more shows the preview card (ID, assignee, title, status, priority); shorter hovers don't. Evidence: `hover-intent` tests; preview text "upe-00000c / Sarah Chen / Implement user registration flow / Done / No priority".
- [x] Right-clicking a chip opens Change to ▸ / Remove relation / Copy link. Evidence: the a11y snapshot of the "Relation actions" menu; the nested More ▸ left out the current kind.
- [x] Change to replaces the link in place, and the issue moves to the new section on both issues. Evidence: Depends on → Blocked by on upe-000009; the API showed `blocked_by upe-000008` and the inverse on upe-000008.
- [x] Clicking a chip navigates to the linked issue. Evidence: chips are `<a href="#/issues/<id>">`.
- [x] The Relations card's `Waypoints` icon button opens the dependency graph focused on this issue. Evidence: clicking it navigated to `#/dependencies/upe-000009`; the svg class was `lucide-waypoints`.
- [x] Frontend tests cover grouping (including derived and legacy pairs), the change-kind link-set computation and the button label. Evidence: `relation-utils.test.ts` (24 tests) and `hover-intent.test.ts` (4).
- [x] `make test` passes. Evidence: exit 0; 603 frontend tests plus the Go suite.
