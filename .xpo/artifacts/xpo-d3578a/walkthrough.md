# Walkthrough: activity chips use flex alignment

## What changed and why

System rows in the issue activity timeline read as one sentence, for example "Nicolas changed status to ○ Planned · 2m ago". Status, estimate and priority values show as chips with an icon and a label.

xpo-ca763f laid these rows out as inline text. The chip label sat on the text baseline, and `inline-block align-middle` centered the icon. In the user's browser the result was visibly misaligned. The user tested a flex layout in devtools, and it rendered cleanly. This change adopts that layout.

## The pieces

```
ActivityTimeline.tsx     system row: flex flex-wrap items-center gap-x-1
  └─ describeEvent()     activity-describe.tsx (unchanged): text runs + chips
       └─ chips          activity-chips.tsx: inline-flex items-center
            └─ ChipIcon  mr-1, optional lift
```

- **Row** (`data-activity="system"`): `flex flex-wrap items-center gap-x-1`. Each text run, chip and separator becomes a flex item, and the row centers the items vertically.
- **Chips** (`StatusChip`, `EstimateChip`, `PriorityChip`): a shared `CHIP` class, `inline-flex items-center` plus the hover styles. The chip centers its icon on its label.
- **`ChipIcon`**: keeps `mr-1` for the 4px icon-to-label space. It no longer uses `inline-block align-middle`.

## Decisions

- **The gap replaces word spaces.** Flex trims whitespace at the edges of each item and drops whitespace-only text nodes. The `gap-x-1` (4px) gives about one word space at `text-sm`. The source still has its spaces, so the text content still reads as one sentence. The " and " separator span and " · " work the same way.
- **Long rows wrap between items.** With flex, a row cannot wrap word by word like a sentence. `flex-wrap` moves whole items to the next line, so a long row (many labels) does not overflow. The user chose this over a single non-wrapping line.
- **The triangle has a 0.25px lift.** The lucide `Triangle` ink is not vertically centered in its box. The old inline layout needed `-top-[1.5px]`. Under flex centering, 1.5px was too high and 0px was too low. The user tuned it by eye through 0.5px to 0.25px. The status and priority icons need no lift.

## Tests

`ActivityTimeline.test.tsx` checks that the row has `flex`, `flex-wrap`, `items-center` and `gap-x-1`. It checks that each chip has `inline-flex` and `items-center`, with its icon before the label. A separate test checks that the row text still reads as one sentence.

The old test "flows as text without a flex gap" passed without testing anything. Its regex expected `class` as the first attribute of the row, but `data-activity` comes first, so the class string was always empty. The new test matches `class` in any position.

## Acceptance criteria

- [x] Status, estimate and priority chips sit centered on the row text, with the icon centered on the label. Evidence: the user confirmed the rows visually; the row and chip class tests pass.
- [x] The estimate triangle sits level with its label (0.25px lift). Evidence: the user confirmed it as "pixel perfect".
- [x] Words, chips and the `·` separator keep normal spacing. Evidence: the user checked the rows; the "reads as one sentence" test passes.
- [x] Long rows wrap instead of overflowing. Evidence: the row has `flex-wrap`, checked by the class test.
- [x] `make test` passes. Evidence: 939 frontend tests and the Go suite pass, lint is clean.