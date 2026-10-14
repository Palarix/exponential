## What changed
Rendered checklist items now match the editor. Wrapped lines start at the same x as the first line of text, and the checkbox sits at the same height as in the editor.

## How the editor does it
TipTap renders each task item as a flex row: `li > label > input` and `li > div > p`. The text lives in its own column, so wrapped lines stay in that column. The editor puts 1em between the checkbox and the text: 0.5em of label margin and 0.5em of input margin.

## Why the rendered view was different
react-markdown with remark-gfm emits `<li class="task-list-item"><input type="checkbox"> text</li>`. In a loose list the input and text sit inside a `<p>`. The checkbox was a plain inline box in the text, so wrapped lines started at the left edge, under the checkbox.

## The fix
The fix has two parts.

**1. Hanging indent (`web/src/index.css`).** The `li.task-list-item` rule gets `padding-left: calc(14px + 1em)`. This is the checkbox width plus the editor's 1em gap. The checkbox gets `margin-left: calc(-14px - 1em)` and `margin-right: 1em`. Its net horizontal advance is zero. The checkbox moves into the left gutter, and the first line of text starts at the padding edge, like the wrapped lines. The checkbox keeps `position: relative; top: 2px` and baseline alignment, so it stays at the same height as in the editor.

**2. Remove the space after the checkbox (`web/src/utils/rehype-task-list-spacing.ts`).** mdast-util-to-hast adds a separate `" "` text node after the task checkbox. With the hanging indent, this space moved the first line one space-width right of the wrapped lines. `rehypeTaskListSpacing` visits each checkbox `<input>` and removes a whitespace-only text node right after it. Spaces inside the item text stay. The plugin runs in every react-markdown renderer: `IssueMarkdown` and the two renderers in `MergeView`. If you add a new react-markdown renderer, add this plugin too. Without it, checklists in that renderer show the one-space offset again.

## Rejected approach
The first version made the checkbox `position: absolute` with auto offsets. An out-of-flow checkbox lets the leading space collapse, so no plugin is necessary. But Chrome puts an absolutely positioned inline box at the top of the line box, not on the text baseline. At `line-height: 1.75` the checkbox sat 4.5px higher than in the editor. Baseline alignment needs the checkbox to stay inline, so the space has to go in the markdown pipeline.

## Verification
- `rehype-task-list-spacing.test.tsx` covers tight lists, loose lists, spaces inside item text, and non-task lists.
- In the real board, measured against the editor: checkbox top 6.5px, checkbox left 0, text at 29px, and wrapped lines at 29px. These values are the same for tight and loose lists.
- `make test` passes.

## Acceptance criteria
- [x] Wrapped lines of a rendered checklist item start at the same x as the first line of text. Evidence: wrapped lines measured at 29px, the same as the first line.
- [x] The text indent in the rendered view matches the editor's indent. Evidence: text at 29px in both views.
- [x] The checkbox's vertical position matches the editor. Evidence: checkbox top 6.5px in both views.
- [x] Tight and loose task lists both render correctly. Evidence: unit tests for both list types and the board measurements.
