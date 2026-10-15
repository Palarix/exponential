# Prose links: brighter, readable link color

## What
Give markdown prose links (`.prose-exponential a`) their own light color token, drop the underline at rest, and add a status icon to issue-ref chips, so chips stay easy to tell apart from plain links.

## Why
Links used `--color-accent-primary` (`#5e6ad2`), the primary button fill. As text on the dark surfaces it reads dim, at about 3.2–3.9:1 contrast, below WCAG AA (4.5:1). A brighter blue link is close to the chips' accent-tint text, so the chips need another cue besides color.

## How
In `web/src/index.css`:

- New token `--color-link: #adbbff`, next to the accent tokens.
- `.prose-exponential a` uses `var(--color-link)`, with `text-decoration: none`.
- `.prose-exponential a:hover` adds the underline. That is the hover state.
- Issue-ref chips keep their accent-tint colors. `.issue-ref:hover` sets `text-decoration: none`, so chips don't pick up the link underline.
- `.issue-ref` changes from `inline-block` to `inline-flex` with `align-items: center` and `gap: 0.35em`, so the status icon centers on the chip. The text sits in a `.issue-ref-text` span with `align-self: baseline`, so the chip still lines up with the surrounding text's baseline.
- `.issue-ref > svg { translate: 0 0.06em; }`: at the chip's exact center the icon read a hair high, so this moves it down (tuned on the tophat page).

In `IssueRef.tsx`: once the issue has loaded, the chip leads with `<StatusIcon status={issue.status} size={11} />`. Chips for missing issues, and chips whose issue list is still loading, have no icon.

Contrast for `#adbbff`: about 9–11:1 on every `--color-surface*`.

## Decisions (made on the tophat page)
- Links: `#adbbff`, no underline at rest, underline on hover. This replaced the earlier accent-tint 60/40 proposal.
- Chips: the yellow and neutral variants were rejected. Chips keep the accent tint and get the status icon, so they read as issues without a separate color.

## Tests
- `web/src/link-color.test.ts`: `--color-link` has at least 4.5:1 contrast against every `--color-surface*` hex token. The link rule uses the token, has no underline at rest, and underlines on hover. Chips don't underline on hover.
- `IssueRef.test.tsx`: a chip for a loaded issue renders its status icon. A missing-issue chip has no icon.

## Acceptance criteria
- [ ] Links in the chat transcript and markdown prose have at least 4.5:1 contrast against the background
- [ ] Links have a visible hover state (underline)
- [ ] The color comes from a token, not a hard-coded value in the rule
- [ ] Issue-ref chips still look distinct from plain links (status icon + chip background)
- [ ] The status icon is vertically centered in the chip
- [ ] `make test` passes
