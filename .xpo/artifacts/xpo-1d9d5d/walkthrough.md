# Walkthrough: Prose links: brighter, readable link color

## What changed and why
Links in rendered Markdown (`.prose-exponential a`) used `--color-accent-primary` (`#5e6ad2`), the fill of primary buttons. As text on our dark surfaces it reached only about 3.2–3.9:1 contrast. Links now have their own token, `--color-link: #adbbff`, at about 9–11:1.

A bright blue link sits close in color to the issue-ID chips' accent-tint text. We tried yellow and neutral chips on a tophat page, and the user picked a different cue: chips keep their colors and start with the issue's status icon. Now a chip reads as "an issue" by its shape, so it doesn't need a color of its own.

## The pieces

### `web/src/index.css`
- **`--color-link`** sits next to the accent tokens. It is a plain hex value, not a `color-mix` of the accent. The first proposal (accent mixed 60% with white) was dropped on the tophat page in favor of `#adbbff`, which the user chose.
- **`.prose-exponential a`** uses the token with `text-decoration: none`, and **`.prose-exponential a:hover`** adds the underline. The underline is the hover state, so no separate hover color is needed.
- **`.issue-ref:hover`** now sets `text-decoration: none`. Without it, chips inside prose would pick up the link hover underline. `.prose-exponential a:hover` has lower specificity than `.prose-exponential .issue-ref:hover`, but the chip rule has to name the property for that to help.
- **Chip layout** changed from `inline-block` to `inline-flex` with `align-items: center` and `gap: 0.35em`. With inline-flex, the container's baseline comes from its first item, and an SVG has no text baseline, so the chip would drop out of line with the text around it. To prevent that, the label sits in a **`.issue-ref-text`** span with `align-self: baseline`. That span is the only item using baseline alignment, so it sets the chip's baseline, and the icon still centers in the chip's height (the 0.875em × 1.6 line box, the same as before).
- **`.issue-ref > svg { translate: 0 0.06em; }`** is an optical nudge. The JetBrains Mono line box centers a little below the visual middle of the ID, and the user found the exactly centered icon a hair high. We measured in headless Chromium and tuned on the tophat page: first the icon sat level with the digits (too high), then at the line-box center, and finally 0.06em lower, which the user approved. `translate` moves only the painted icon, so the layout and the chip height don't change.

### `web/src/components/IssueRef/IssueRef.tsx`
When the issue is in the loaded issue list, the chip renders `<StatusIcon status={issue.status} size={11} />` before the label. Chips for missing issues (the muted `issue-ref-missing` span) and chips whose issue list is still loading have no icon. The icon appears once the data arrives. `IssueRef` backs every chip (prose, `IssueRefText` in chat titles, activity, approval cards), so they all get the icon.

## Tests
- **`web/src/link-color.test.ts`** (new) reads `index.css?raw`:
  - It checks `--color-link` for at least 4.5:1 WCAG contrast against every `--color-surface*` hex token, so a new, lighter surface would fail the test.
  - It checks that the link rule uses the token without an underline, that hover underlines, and that chips don't underline on hover.
- **`IssueRef.test.tsx`:** the markup assertions now expect the `.issue-ref-text` span. New cases check that a known issue's chip starts with its status icon (`--color-status-done`), that there's no icon while issues are loading, and that missing chips have no icon.

## Acceptance criteria
- [x] Links have at least 4.5:1 contrast. Evidence: `#adbbff` is about 9–11:1 across surfaces; `link-color.test.ts` checks every surface token.
- [x] Links have a visible hover state. Evidence: the underline on hover; tested.
- [x] The color comes from a token. Evidence: `.prose-exponential a` uses `var(--color-link)`; tested.
- [x] Issue-ref chips still look distinct from plain links. Evidence: the status icon and the tinted chip background; the user approved on the tophat page.
- [x] The status icon is vertically centered in the chip. Evidence: inline-flex centering plus a 0.06em optical nudge, measured in Chromium; the user approved it ("perfect").
- [x] `make test` passes. Evidence: 1106 frontend tests plus the Go suite, exit 0.
