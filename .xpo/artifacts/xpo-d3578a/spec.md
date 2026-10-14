# Activity chips: flex alignment

## What

Lay out activity system rows and their chips with flexbox, centering items on the cross axis, instead of the inline-text approach from xpo-ca763f.

## Why

The inline approach (label on the text baseline, icon nudged with `align-middle`) renders misaligned in the user's browser. The user confirmed in devtools that the flex layout below renders cleanly.

## How

- `activity-chips.tsx`: `StatusChip`, `EstimateChip`, `PriorityChip` become `inline-flex items-center`. The icon wrapper keeps `mr-1` for the 4px icon–label space and drops `inline-block align-middle`.
- `ActivityTimeline.tsx`: the system row text container (`data-activity="system"`) becomes `flex flex-wrap items-center gap-x-1`.
- Reduce the estimate triangle's lift from `-top-[1.5px]` to `-top-[0.25px]`. Under flex centering, 1.5px puts the triangle too high and no lift puts it too low (both confirmed by the user).
- Replace the comments that describe the old inline rationale.

## Decisions

- **Wrapping:** with flex, each text run and chip is a separate flex item. A long row (e.g. many labels) wraps between items, not word by word like a sentence. `flex-wrap` keeps long rows from overflowing. Chosen by the user.
- **Triangle lift 0.25px:** lucide's `Triangle` ink is not vertically centered in its box, so it still needs a small optical correction. The user tuned it by eye: 1.5px too high, 0px too low, then 0.5px, then 0.25px.

## Acceptance criteria

- [ ] Status, estimate and priority chips sit centered on the row text, with the icon centered on the label.
- [ ] The estimate triangle sits level with its label (0.25px lift).
- [ ] Words, chips and the `·` separator keep normal spacing (no doubled or missing spaces).
- [ ] Long rows wrap instead of overflowing.
- [ ] `make test` passes.