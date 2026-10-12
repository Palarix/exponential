# Spec: Highlight issue IDs and make them clickable everywhere in transcripts

## What

Show every issue ID in a chat transcript as an issue chip. A click on the chip opens the issue, and hovering it shows a preview. Chips appear in the agent's answers, the user's messages and the tool rows. The same Markdown plugin replaces `linkifyIssueIds` in the issue description, spec and walkthrough tabs and in comments.

## Why

- Agent answers mention many issue IDs, but the user's messages and the tool rows show them as plain text.
- `linkifyIssueIds` is a regex replace that runs before Markdown parsing:
  - An ID inside a code span becomes the literal text `[xpo-…](#/issues/xpo-…)`.
  - An ID that is already inside a link is wrapped a second time.
- The links use the normal link style, so they do not stand out as issue references.

## How

### 1. Tokenizer: `web/src/utils/issue-refs.ts`

- `issueIdPattern(prefix)` returns `\b(<escaped prefix>-[a-f0-9]{6})\b` with the `g` flag. `GET /api/config` sends the prefix without the hyphen (`xpo`), and the function also accepts a trailing hyphen. The old `linkifyIssueIds` did not add the hyphen, so plain IDs were never linked on a real board (bug xpo-b83f1c, fixed here). The frontend fallback prefix changes from `issue-` to `issue`, the same as the server.
- `splitIssueIds(text, prefix): Array<string | { id: string }>` splits text into plain parts and ID parts.
- `issueHref(id)` returns `#/issues/<id>`.
- `issueIdFromHref(href)` returns the ID for an `#/issues/<id>` href, and `null` for any other href.

### 2. Remark plugin: `web/src/utils/remark-issue-refs.ts`

`remarkIssueRefs({ prefix })` walks the mdast with `unist-util-visit`. That package is already installed through react-markdown; the plugin adds it as a direct dependency.

- **`text` node:** replace the node with text nodes and `link` nodes (url `issueHref(id)`).
- **`inlineCode` node that is only an ID:** replace it with a plain ID `link`, so it renders as the chip alone and not as a chip in a code box.
- **`inlineCode` node that contains an ID among other text:** keep the single code span. Set `data.hName = "code"` and set `data.hChildren` to text nodes and `a` elements. `mdast-util-to-hast` renders `hChildren` as the children of the code element, so the code styling stays.
- **Skipped:** text inside `link`, `linkReference` and `definition` nodes (no double wrap). `code` (fenced) and `html` nodes have no text children, so they stay plain.

### 3. Chip: `web/src/components/IssueRef.tsx`

- `IssueRef({ id, children? })` looks up the issue in `useIssueList()`.
  - **Known issue:** `<a href="#/issues/<id>">` styled as a chip, not as a link: an inline-block at 0.875em with `line-height: 1.6`, so all chips are the same height whatever the font and stay below the line height of the text around them; monospace, rounded (`--radius-md`), an accent-tinted background (`--color-issue-ref-bg`, 18% accent, 30% on hover), light accent text (`--color-issue-ref`, accent mixed with white). The tokens derive from `--color-accent-primary` and no underline. Hovering it for `PREVIEW_DELAY` shows `IssuePreview`.
  - **Unknown issue:** a muted, non-clickable gray chip (`surface-3` background, muted text, no underline) with the tooltip "Issue not found".
  - **Custom label** (`label` prop): a Markdown link with its own text, such as `[the epic](#/issues/…)`, is a chip that shows that text in the body font, at the same size and height as an ID chip. It still shows the preview.
- `IssuePreview` moves from `Relations.tsx` to a shared file (`components/IssuePreview.tsx`). Relations imports it from there. Its look does not change.
- `MarkdownLink` is the react-markdown `a` component. It renders `IssueRef` when `issueIdFromHref(href)` matches, with `label` set when the link text is not the ID itself. It renders a normal `<a>` (href and title only) for any other href.
- `IssueMarkdown({ children })` wraps react-markdown with `remarkGfm`, `remarkBreaks`, `remarkIssueRefs` (prefix from `useConfig()`) and `MarkdownLink`. All call sites use it. The now unused `prefix` props of `IssueDetail` and `ActivityTimeline` are removed.
- Inside a code span, the chip keeps the code's font size and uses less padding.
- `IssueRefText({ text })` renders plain text with `splitIssueIds` and an `IssueRef` for each ID. It is used where there is no Markdown.

### 4. Where it is used

- `ChatThread.tsx`:
  - `AnswerText`: `remarkIssueRefs` plus `components={{ a: MarkdownLink }}`.
  - User message: `IssueRefText`.
  - `WorkStep` detail: `IssueRefText`. The truncate wrapper stays.
- `IssueDetail.tsx` (description, spec, walkthrough) and `ActivityTimeline.tsx` (comments) use the plugin and `MarkdownLink` in place of `linkifyIssueIds`.
- `linkifyIssueIds` and its tests are deleted.

### 5. Navigation

The chip is a normal hash link. The hash router opens the issue in the main view. The popup, the pills and the Agent view keep their state because the provider is mounted above the router. A click in the full-page Agent view leaves that view (the user can return through the sidebar or the history), which is the same as any other link.

## Decisions

- **Chip style everywhere:** the issue description, specs and comments also get the chip, so issue references look the same in all places. The user confirmed this.
- **Unknown IDs:** muted plain text with a "not found" tooltip, not a link. A link to a missing issue goes to a "not found" page, which does not help.
- **No inline title:** the chip does not show the issue title inline. Agent text already names the issue, and the ID stays compact.

## Acceptance criteria

- [ ] IDs that match the project prefix are links to `#/issues/<id>` in the agent's answers, the user's messages and the tool rows.
- [ ] IDs in inline code are links, and the code styling stays. IDs in fenced code blocks stay plain text.
- [ ] IDs that are already inside a Markdown link are not wrapped again.
- [ ] Issue links are chips (accent-tinted rounded background, light accent text, no underline), not link-styled text. Hovering one shows the title and status with the shared `IssuePreview`.
- [ ] A click opens the issue in the main view. The popup and the Agent view keep their state.
- [ ] Unknown IDs show as a muted gray chip with a "not found" tooltip.
- [ ] Unit tests cover code spans, code blocks, existing links, punctuation around IDs and other prefixes.
- [ ] The issue description, specs, walkthroughs and comments use the plugin, and `linkifyIssueIds` is gone.
- [ ] Markdown links with custom text to an issue keep their text, as a chip.
- [ ] `make test` passes.
