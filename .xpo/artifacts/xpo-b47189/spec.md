# Issue-ref chips: fixed line height, centered chips

## What

An issue-ref chip must never change the height of the line it sits in. Inside that fixed line, the chip should read as vertically centered next to the text around it. That has to hold in every place chips render:

- chat prose: `.prose-exponential`, 14–15px at `line-height: 1.75`
- the user message bubble (`text-sm`)
- the Work Mode divider (`text-xs`, hairlines on either side)
- `IssueRefText` in tool rows and approval cards
- chips inside inline code (`code .issue-ref`)
- the activity comment prose (14px)

## Why

Other apps keep a line at its set height whatever inline element it holds. Ours didn't:

- **Baseline alignment** (main, xpo-1d9d5d): the chip sits low, and in the `text-sm` bubble it stretches the line by 0.19px.
- **`vertical-align: middle`:** stretches tight lines by up to 0.55px, and the divider label floats above the hairlines.
- **1em + weight 200:** stretches lines by up to 1.84px.

## Decision (picked on the tophat page)

**Variant D, the inline chip, at 0.875em / weight 400.** It was the only variant measured at `+0px` in every context at both sizes. It also lines up exactly with the divider hairlines (0.00px) and has the smallest chip-vs-text offset (0–0.5px).

The rule: the chip is an **inline box, not an inline-flex box**. An inline box adds its own `line-height` to the line, and its padding and background paint outside that height without growing the line box. That is the approach GitHub and Linear use for mentions.

```css
.issue-ref {
  display: inline;
  font-size: 0.875em;
  line-height: 1;            /* below every parent line height, so the line never grows */
  padding: 0.15em 0.45em;    /* the tint; vertical padding paints outside the line box */
  vertical-align: baseline;  /* shares the text's baseline */
  box-decoration-break: clone; /* a wrapped chip keeps its tint and radius on each piece */
}
.issue-ref > svg {
  display: inline;
  vertical-align: middle;
  margin-right: 0.35em;      /* replaces the flex gap */
  translate: 0 -0.06em;      /* `middle` sits a hair low on the mono x-height */
}
code .issue-ref { font-size: inherit; padding: 0.15em 0.3em; }
```

- The `.issue-ref-text { align-self: baseline }` rule goes away (no flex container any more). The span stays in the markup as the label hook.
- The icon nudge flips from `+0.06em` to `-0.06em`: in the flex layout the icon sat high at the box center, while `vertical-align: middle` puts it a hair low.
- The missing-issue chip (`span.issue-ref-missing`) uses the same rules.

## Tests

- `IssueRef.test.tsx` markup is unchanged (icon, then `.issue-ref-text`).
- New CSS test (same style as `link-color.test.ts`): the `.issue-ref` rule is `display: inline` with `line-height: 1` and `vertical-align: baseline`, and no `inline-flex`. The svg rule uses `vertical-align: middle`.

## Acceptance criteria

- [ ] In every context listed, a line with a chip is the same height as a line without one (tophat overlay shows `+0px`)
- [ ] The chip reads as centered in prose, the bubble and the divider, and the divider hairlines line up with label and chip
- [ ] Chips inside inline code still fit the code span
- [ ] Chips still look distinct from plain links (status icon + tint)
- [ ] Tophat page removed before commit; `make test` passes

## Out of scope

- A "fixed line height" rule for other inline elements, such as inline code spans. A follow-up if wanted.
