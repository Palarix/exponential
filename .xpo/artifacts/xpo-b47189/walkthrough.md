# Walkthrough: issue-ref chips keep line height fixed

## What changed

Issue-ref chips (`.issue-ref`) used to be `inline-flex` boxes with `line-height: 1.6`. A flex box contributes its whole border box to the line, so in tight contexts it stretched the line. In the `text-sm` bubble that was +0.19px, and with `vertical-align: middle` up to +1.84px. On top of that, the chip sat low on the baseline.

The chip is now a plain **inline box** (variant D from the tophat page):

```css
.issue-ref, .prose-exponential .issue-ref {
  display: inline;
  font-size: 0.875em;
  line-height: 1;
  vertical-align: baseline;
  padding: 0.15em 0.45em;
  box-decoration-break: clone;
  …
}
.issue-ref > svg {
  display: inline;
  vertical-align: middle;
  margin-right: 0.35em;
  translate: 0 -0.06em;
}
```

## Why this works

CSS builds a line box from each inline box's `line-height`, not from its padding or background. With `line-height: 1` at 0.875em, the chip's inline box is always shorter than the parent's line. Every context is at least 1.33 × the font size: `text-xs` is 12/18 and prose is 1.75. So the chip can never stretch the line. The vertical padding paints the tint above and below that box without moving anything. GitHub and Linear draw their mention pills the same way.

Because the chip shares the text's baseline, the label sits exactly where the surrounding text sits. That is why the divider hairlines line up with it (0.00px offset, measured).

## Details

- **Icon:** without flex, the icon is placed with `vertical-align: middle`, which centers it on the chip's x-height. That reads a hair low in JetBrains Mono, so the nudge flips from `+0.06em` (flex layout, where the icon sat high) to `-0.06em`. `margin-right` replaces the old flex `gap`.
- **`.issue-ref-text`:** its `align-self: baseline` rule is gone, since there is no flex container any more. The span stays in the markup as the label hook, so the `IssueRef.test.tsx` markup tests are unchanged.
- **Wrapping:** chips are `nowrap`, but a long custom label can still break. `box-decoration-break: clone` (plus the `-webkit-` prefix) gives each piece its own padding and radius.
- **Inside code:** the chip inherits the code's size and gets `padding: 0.15em 0.3em`.

## How it was chosen

A temporary `#/tophat` page (now deleted) rendered every chip context five ways, each next to a version without chips, and measured the line-height difference in headless Chrome:

| Variant | Worst line delta | Chip vs. hairline |
|---|---|---|
| A. main (inline-flex, baseline span) | +0.19px (+1.8px at 1em) | +0.40px |
| B. `vertical-align: middle` | +0.55px | +0.66px |
| C. 1em / 200 + middle | +1.84px | +0.06px |
| **D. inline chip** | **+0px** | **0.00px** |
| E. fixed-height inline-flex | +0px | +0.72px |

D won at both 0.875em and 1em. The user chose 0.875em / weight 400 so chips match inline code.

## Acceptance criteria

- [x] In every context, a line with a chip is the same height as a line without one: re-measured the shipped CSS before deleting the tophat page, `+0.00px` in all 8 contexts
- [x] The chip reads as centered in prose, the bubble and the divider, and the hairlines line up: chip vs. hairline 0.00px, chip vs. text 0–1.5px (prose)
- [x] Chips inside inline code still fit the code span: the inline-code row measured `+0.00px` with a 0.00px offset
- [x] Chips still look distinct from plain links: status icon and tint unchanged
- [x] Tophat page removed; `make test` passes (75 files, 1110 frontend tests, plus Go tests)

## Tests

`web/src/issue-ref-chip.test.ts` reads `index.css` raw (like `link-color.test.ts`). It checks that the chip rule is `display: inline` without `inline-flex`, with `line-height: 1`, `vertical-align: baseline` and `box-decoration-break: clone`, and that the icon uses `vertical-align: middle`.
