# Walkthrough: Highlight issue IDs and make them clickable everywhere in transcripts

## What was built

Every issue ID in a chat transcript now shows as a chip. A chip has a rounded, accent-tinted background and light accent text. A click opens the issue, and hovering it shows the issue preview card. Chips appear in the agent's answers, the user's messages and the tool rows. The issue description, the spec and walkthrough tabs, and comments use the same code, so issue references look the same everywhere.

The work also fixed bug xpo-b83f1c: before this change, plain issue IDs were never linked on a real board.

## How the pieces fit together

```
text ──► splitIssueIds (utils/issue-refs.ts)
            │
            ├── remarkIssueRefs (utils/remark-issue-refs.ts) ──► mdast link nodes
            │        └── IssueMarkdown ──► react-markdown + MarkdownLink (a)
            │                                   └── IssueRef
            └── IssueRefText (plain text: user messages, tool rows) ──► IssueRef
```

### 1. Tokenizer: `web/src/utils/issue-refs.ts`

- `issueIdPattern(prefix)` matches `<prefix>-[a-f0-9]{6}` between word boundaries.
- `splitIssueIds` returns plain strings and `{ id }` parts in order.
- `issueHref` and `issueIdFromHref` convert between an ID and `#/issues/<id>`.

All other parts use these functions, so there is one match rule.

### 2. Remark plugin: `web/src/utils/remark-issue-refs.ts`

The old `linkifyIssueIds` was a regex replace on the raw Markdown before parsing. That broke code spans (the reader saw `[xpo-…](#/issues/…)` as literal text) and wrapped existing links a second time. The plugin works on the parsed tree (mdast) instead:

- **`text` nodes** become text and `link` nodes.
- **`link`, `linkReference` and `definition` nodes** are skipped (`SKIP`), so IDs in existing links and autolinks stay as they are.
- **Fenced `code` nodes** have a `value` and no text children, so the visitor never changes them.
- **`inlineCode` that is only an ID** becomes a plain ID link. A chip inside a code box looked doubled.
- **`inlineCode` with an ID among other text** (for example `xpo show xpo-…`) keeps the single code span. The plugin sets `data.hName = "code"` and `data.hChildren` to text and `<a>` hast nodes. `mdast-util-to-hast` uses `hChildren` as the element's children, so the code element contains the link.

`unist-util-visit`, `@types/mdast` and `@types/hast` are now direct dependencies. They were already in the tree through react-markdown.

### 3. Components: `web/src/components/IssueRef/`

- **`IssueRef({ id, label? })`**
  - Looks the issue up in the `useIssues()` cache.
  - **Known issue:** an `<a class="issue-ref">` with a hover preview. It uses `createHoverIntent` with the same 250 ms delay as the Relations chips.
  - **Unknown ID:** a gray `issue-ref-missing` span with the tooltip "Issue not found". This happens only after the list has loaded; while it loads the chip stays a link, so IDs do not flash as missing.
  - **`label`:** a Markdown link with its own text, such as `[the epic](#/issues/…)`, renders as a chip with that text in the body font.
- **`MarkdownLink`**: the react-markdown `a` component.
  - An `#/issues/<id>` href renders `IssueRef`, with `label` set when the link text is not the ID itself.
  - Other hrefs render a plain `<a>`. Only `href` and `title` are passed on, so react-markdown's `node` prop does not reach the DOM.
- **`IssueMarkdown`**: wraps react-markdown with `remarkGfm`, `remarkBreaks`, `remarkIssueRefs` (the prefix comes from `useConfig()`) and `MarkdownLink`. All five Markdown call sites use it.
- **`IssueRefText`**: for plain text (user messages and tool-row details). It uses `splitIssueIds` and renders an `IssueRef` for each ID.
- **`IssuePreview`**: moved here unchanged from `Relations.tsx`, with an optional `placement`. Inline chips use `bottom-start`; Relations keeps `right-start`.

### 4. Styles: `web/src/index.css`

- Three tokens derive from the accent:
  - `--color-issue-ref`: 45% accent, mixed with white.
  - `--color-issue-ref-bg`: 18% accent.
  - `--color-issue-ref-bg-hover`: 30% accent.
  If the accent changes, the chips follow it.
- `.issue-ref` is an **inline-block at 0.875em with `line-height: 1.6`**:
  - With a plain inline element, the background height follows the font's own metrics. Monospace and sans chips then had different heights (16 to 19.8 px).
  - With a fixed line-height, every chip is 19.6 px in prose.
  - 0.875 × 1.6 = 1.4em, which is below the prose line height (24.5 px at 14 px). The line spacing does not change.
- The chip rules are placed after `.prose-exponential a` and repeat the `.prose-exponential` scope. Without that, the prose link color and underline would win.
- Inside a code span the chip keeps the code's font size and has less padding.

### 5. Prefix fix (xpo-b83f1c)

`GET /api/config` sends the prefix without the hyphen (`xpo`). The old regex was `<prefix>[a-f0-9]{6}`, so it needed `xpo-` and never matched on a real board. Only links that were written as Markdown links worked. The tests passed `"xpo-"`, so they did not catch it.

- `issueIdPattern` now strips a trailing hyphen and adds one, so both forms work.
- The frontend fallback prefix is `issue`, the same as the server's fallback.
- The tests use the server form.

### 6. Cleanup

- `linkifyIssueIds` and its tests are deleted.
- The `prefix` props on `IssueDetail` and `ActivityTimeline` were used only by `linkifyIssueIds`, so they are removed, together with the pass-through in `Inbox` and `ViewRouter`.

## Decisions from review

- **Chips everywhere:** the user chose the chip style for descriptions, specs and comments too, not only for chat.
- **Chips, not links:** blue link text looked wrong. Chips have a tinted background and no underline. Label links and unknown IDs are chips as well.
- **Brand colors:** the first version used the info blue. The tokens now derive from `--color-accent-primary`.
- **Same height:** the label chip used to inherit the full font size. All chips now use 0.875em with a fixed line-height.

## Acceptance criteria

- [x] IDs that match the project prefix are links to `#/issues/<id>` in the agent's answers, the user's messages and the tool rows. Evidence: Playwright found 12 chip links in the demo chat across all three places. A click went to `#/issues/upe-000008`.
- [x] IDs in inline code are links, and the code styling stays. IDs in fenced code blocks stay plain text. Evidence: the `remark-issue-refs.test.tsx` tests (inline code, code-only ID, fenced block) and the demo screenshot.
- [x] IDs that are already inside a Markdown link are not wrapped again. Evidence: the tests for links, reference links and autolinks.
- [x] Issue links are chips (accent-tinted rounded background, light accent text, no underline), not link-styled text. Hovering one shows the title and status with the shared `IssuePreview`. Evidence: the hover screenshot shows the preview for upe-000008.
- [x] A click opens the issue in the main view. The popup and the Agent view keep their state. Evidence: the chip is a hash link, and the agent provider is mounted above the router.
- [x] Unknown IDs show as a muted gray chip with a "not found" tooltip. Evidence: `IssueRef.test.tsx`, and Playwright found `upe-abcdef` as the only `.issue-ref-missing`.
- [x] Unit tests cover code spans, code blocks, existing links, punctuation around IDs and other prefixes. Evidence: `issue-refs.test.ts`, `remark-issue-refs.test.tsx` and `IssueRef.test.tsx`.
- [x] The issue description, specs, walkthroughs and comments use the plugin, and `linkifyIssueIds` is gone. Evidence: `grep linkifyIssueIds` finds nothing.
- [x] Markdown links with custom text to an issue keep their text, as a chip. Evidence: the `MarkdownLink` test and the demo ("the epic").
- [x] `make test` passes. Evidence: lint, 704 vitest tests and the Go tests.
