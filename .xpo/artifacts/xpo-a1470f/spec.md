# Activity as a "Timeline"

## What
Draw a vertical timeline rail through the Activity list in the issue detail view (`web/src/components/IssueDetail/ActivityTimeline.tsx`). The rail joins the rows' avatars so the feed reads as one continuous timeline instead of a stack of separate lines and cards.

## Why
Today, system events (status changes, merges, artifacts) and comment cards are separated only by whitespace. A rail makes the order and continuity of events easy to see, much like Linear and GitHub.

## How
- The rail is a 1px vertical line in `--color-border-default` (`subtle` is almost invisible on the dark background), centered on the avatar column. System rows get a 17px left padding, so their avatar center (25px) lines up with comment avatars, which sit inside a 1px card border plus 16px of padding.
- Each row renders its own connector segments, with no measuring of the DOM:
  - a segment **above** the avatar (from the row top to the avatar center), except on the first row
  - a segment **below** the avatar (from the avatar center to the row bottom, reaching the next row), except on the last row
- To keep segments touching, vertical spacing between rows comes from top padding on each row's wrapper (12px for system rows, 16px for comments, as today), not from `space-y-*` margins. A row's above-segment spans that gap.
- Segments stop 2px short of system-row avatars rather than running behind them. An opaque ring wouldn't work, because avatars without an email address are transparent with a dashed border.
- Comment cards keep their opaque `surface-1` background and border. The rail runs into the top and out of the bottom of the card, ending at the card border, and is not drawn inside the card.
- A pure helper `timelineConnectors(index, count) → { above, below }` in `activity-timeline-utils.ts` decides which segments a row gets. It is unit tested with vitest.
- Comment bodies use a new `.prose-activity` class (in `index.css`, layered on `.prose-exponential`) that sets the font size to 14px (`0.875rem`). The "Leave a comment…" textarea also uses 14px, so what you type matches how it renders. 14px falls between the `text-sm` (13px) and `text-base` (15px) tokens; the user picked it after trying 13px and 14px.
- The rail follows the sort order (Newest/Oldest): it is based on display position, not on time.

## Edge cases
- 0 entries: no rail.
- 1 entry: no segments.
- A system row whose content wraps onto several lines: the avatar has its own fixed column, so wrapped lines indent to line up with the actor name and never overlap the rail. The below-segment stretches to the bottom of the row.

## Acceptance criteria
- [ ] A continuous vertical line connects the avatars of consecutive activity entries, both system rows and comment cards
- [ ] No line above the first entry and none below the last entry
- [ ] The line doesn't show through avatars or through comment card bodies
- [ ] Spacing between entries looks the same as today
- [ ] Works in both sort orders (the app has only a dark theme)
- [ ] Comment bodies and the comment textarea render at 14px; the issue description stays at 15px
- [ ] `timelineConnectors` is unit tested; `make test` passes

## Decisions
1. The rail ends at the last entry and doesn't continue down to the composer (confirmed by the user).
2. Comment avatars stay inside the card, and the rail runs behind the opaque card (confirmed by the user).
