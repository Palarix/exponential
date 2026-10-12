# Walkthrough: Activity as a "Timeline"

## What was built
The Activity section of the issue detail view now reads as a timeline. A thin vertical rail runs through the avatar column and joins consecutive entries: system events (status changes, merges, artifacts) and comment cards. Comment bodies and the comment composer also move from 15px to 14px, which gives the section three sizes in a clear order: description, then discussion, then log.

## How the pieces fit

### Per-row segments, no DOM measuring
A single absolutely positioned line from the first avatar to the last would need DOM measurement, because the avatar positions depend on how text wraps. Instead, each entry draws its own short segments:

- `activity-timeline-utils.ts` exports `timelineConnectors(index, count) → { above, below }`. This is the only logic in the feature, and it is unit tested: no segment above the first entry, none below the last, and none at all for a single entry.
- In `ActivityTimeline.tsx`, each entry is a `relative` wrapper, and `RailSegment` draws an absolutely positioned 1px line at `left-[25px]`.
- **Why the segments meet.** The gap between entries used to come from `space-y-3` and `!mt-4` margins. It now comes from `paddingTop` on each wrapper (12px for system rows, 16px for comments, so the spacing is the same as before). Because the gap is inside the wrapper, a row's above-segment can span it, and the segments of neighboring rows touch.
- **Where the geometry comes from.** The constants `SYSTEM_GAP`, `COMMENT_GAP`, `AVATAR_TOP` (8 = `py-1.5` 6px + (20px line − 16px avatar)/2), `AVATAR_BOTTOM` and `RAIL_INSET` (2px clearance) are named at the top of the file. If any of these paddings or the avatar size change, update the constants with them.

### One avatar column
Comment avatars sit inside a card with a 1px border and 16px of padding, which puts their center at 25px. System rows used to have `px-4`, which put theirs at 24px. System rows now use `pl-[17px]`, so every avatar shares the same 25px center, and that's where the rail is drawn.

### Avatar column in system rows
System rows used to be a single `flex-wrap` line, so a wrapped second line (common at narrow widths) started under the avatar and its status icon sat on top of the rail. The avatar now has its own fixed column (`flex h-5 items-center`), and the name, event text and time wrap in a separate `flex-wrap` container. Wrapped lines now indent under the name. This was found while checking the layout in the browser, and the spec was updated to match.

### Comments
Comment cards keep their opaque `surface-1` background. A comment row draws only an above-segment, which ends at the card's top border, and the next row's above-segment starts at the card's bottom edge. The rail looks like it passes behind the card, without drawing anything inside it. The user chose this over the GitHub style of placing the avatar outside the card.

### Comment typography
`.prose-exponential.prose-activity` in `index.css` sets the size to `0.875rem` (14px). It's a compound selector because `.prose-exponential` is unlayered CSS, which beats Tailwind's layered utilities. A plain `text-[14px]` class would have had no effect, and that was confirmed during the experiment. Headings and code inside comments are sized in `em`, so they scale with it. The composer textarea uses `text-[0.875rem]`, so what you type looks like what gets posted.

## Key decisions
- **Stop short of avatars instead of masking with a ring.** Avatars without an email address render transparent with a dashed border, so an opaque ring wouldn't hide the line behind them. Instead, segments stop 2px before the avatar edge.
- **`--color-border-default`, not `-subtle`.** `subtle` (#1e1e1e) is almost invisible on the dark background.
- **The rail stops at the last entry** and doesn't continue to the composer (user decision).
- **14px is off the type scale.** The project's `text-sm` is 13px and `text-base` is 15px. The user picked 14px after trying 13px and 14px. If 14px becomes a deliberate step, promote it to a token (for example `--text-md`) and keep the activity prose and composer as its defined use, so the meaning doesn't drift.

## Acceptance criteria
- [x] A continuous vertical line connects the avatars of consecutive activity entries, both system rows and comment cards. *Evidence:* DOM measurements on demo issue upe-000001 put every avatar center and segment at x=501, with each wrapper's bottom equal to the next wrapper's top (no gaps).
- [x] No line above the first entry and none below the last. *Evidence:* `timelineConnectors` tests; in the measurements, the first row has only a below-segment and the last only an above-segment.
- [x] The line doesn't show through avatars or through comment card bodies. *Evidence:* segments end 2px from avatar edges (for example, avatar 405–421, segment starts at 423); comment rows draw no segment inside the card.
- [x] Spacing between entries looks the same as today. *Evidence:* wrapper padding reproduces the old 12px and 16px gaps.
- [x] Works in both sort orders (the app has only a dark theme). *Evidence:* segments depend on display index, not time.
- [x] Comment bodies and the comment textarea render at 14px; the issue description stays at 15px. *Evidence:* computed styles on upe-0000a8: `prose-activity` 14px (paragraphs 14px), textarea 14px, description 15px.
- [x] `timelineConnectors` is unit tested; `make test` passes. *Evidence:* 4 new tests, 607 frontend tests pass, `make test` exits 0.
