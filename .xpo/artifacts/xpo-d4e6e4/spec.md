## What
When a checklist item wraps in the rendered markdown view, the extra lines should line up with the first line of text, and the checkbox should sit exactly where it does in the editor.

## Why
Right now the rendered view puts the wrapped lines back at the left edge, under the checkbox. The editor and the rendered view don't match, and long checklists are harder to scan.

## How
**Hanging indent (CSS, `web/src/index.css`):**
- `.prose-exponential li.task-list-item` gets `padding-left: calc(14px + 1em)`: the checkbox width plus the editor's two 0.5em gaps.
- The checkbox stays inline (`position: relative; top: 2px`, baseline-aligned like before) with `margin-left: calc(-14px - 1em)` and `margin-right: 1em`. Its net horizontal advance is zero, so the first line starts at the padding edge, the same x as the wrapped lines.

**Remove the space after the checkbox (`rehypeTaskListSpacing`, `web/src/utils/rehype-task-list-spacing.ts`):**
- mdast-to-hast inserts a separate `" "` text node after the task checkbox. That space pushed the first line one space-width to the right of the wrapped lines.
- A small rehype plugin removes the whitespace-only text node right after a checkbox `<input>`. It's wired into every react-markdown renderer: `IssueMarkdown` and the two in `MergeView`.

**Rejected:** making the checkbox `position: absolute` so the space collapses. An absolutely positioned inline box's static position is the top of the line box, not the text baseline. At the 1.75 line-height the checkbox ended up 4.5px higher than in the editor.

## Acceptance criteria
- [ ] Wrapped lines of a rendered checklist item start at the same x as the first line of text.
- [ ] The text indent in the rendered view matches the editor's indent.
- [ ] The checkbox's vertical position matches the editor.
- [ ] Tight and loose task lists both render correctly.
