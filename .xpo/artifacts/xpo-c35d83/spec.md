# Agent chat: user messages render as Markdown, composer formats Markdown as you type

## What
1. User messages in the transcript render as Markdown with issue ID chips, like agent replies.
2. (Bonus) The composer is a small TipTap editor that formats Markdown as the user types and sends plain Markdown.

## Why
Users write lists, code and links to the agent. The bubble shows them as raw text, so the user's own message reads worse than the agent's reply.

## How

### Part 1: rendered user messages (`ChatThread.tsx`)
1. The user bubble (`data-user-message`) and the steer bubble (`data-steer-message`) render `IssueMarkdown` in place of `IssueRefText`. `IssueMarkdown` already has GFM, `remark-breaks` (a single newline stays a line break) and issue chips.
2. Drop `whitespace-pre-wrap` from both bubbles. Add `prose-exponential prose-activity prose-bubble`. The new `prose-bubble` modifier (`index.css`) sets a 1.25rem line height and drops the last block's bottom margin, so a one-line message keeps its current bubble height.
3. Unchanged: the agent-turn line (`data-agent-turn`, one truncated line), the queued-message chips (one truncated line of raw text), the turn rail's hover preview, and copy actions, which copy the raw Markdown.

### Part 2: Markdown composer
4. New `ChatMarkdownInput.tsx`: a TipTap editor with StarterKit (headings 1–3, its Link with no open on click), TaskList/TaskItem, Placeholder and `tiptap-markdown` (`html: false`, `breaks: true`, paste and copy as Markdown). No tables and no formatting bar: the composer stays one slim box. Formatting comes from input rules (`**x**`, `*x*`, `` `x` ``, `- `, `1. `, `> `, `` ``` `` followed by a space, `# `) and the usual shortcuts (⌘B, ⌘I, ⌘E).
5. It is controlled by the draft store, as the textarea was. `value` is Markdown, and `onChange(markdown)` fires on each edit. When `value` changes from outside and differs from the last Markdown it reported, it calls `setContent(value)` without an update event and puts the caret at the end. Outside changes are another chat's draft, a prefill, edit-last, a queued message pulled back, and the draft cleared after send. The caret also starts at the end on mount.
6. It exposes `focus()` and `blur()` through a ref, so `ChatComposer` keeps its focus logic (mount, `focusSignal`, prefill, after send, after attach). Both act at once, as on a textarea; TipTap's own commands wait a frame.
7. Keys go through `handleKeyDown`:
   - Code fences come first. Enter or Shift+Enter at the end of a paragraph that is only `` ``` `` or `` ```lang `` turns it into a code block (with that language) and does not send. TipTap's own rule fires only on a space after the fence, so typing a fence and pressing Enter left plain text.
   - Then `ChatComposer`'s handler, unchanged except that it returns whether it took the key:
     - Enter (no Shift, not composing) sends, also inside a list or code block.
     - ↑ in an empty input pulls the last queued message or edits the last message (`composerArrowUp`).
     - Esc cancels edit mode, else blurs when `blurOnEscape` (`composerEscapeBlurs`).
   - Then Shift+Enter, in this order:
     - On a code block's last line, when that line is a closing `` ``` ``, it drops the fence and leaves the block.
     - In a list, a new list item.
     - In a code block, a newline.
     - Otherwise, a new paragraph.
8. Paste: files on the clipboard become attachments, as before. Text pastes as Markdown (`transformPastedText`). Dropped files are left to the composer, which attaches them.
9. The editor grows with its content up to 180px, then scrolls. The `wide` variant keeps its 64px minimum height.
10. `ChatComposer` sends `text.trim()` of the Markdown, as before. Drafts stay Markdown strings, so `composer-drafts.ts` doesn't change.
11. `typing()` in `chat-thread-utils.ts` also counts a contenteditable with text. Without that, an approval or question card would take focus from a composer the user is typing in.

## Decisions
- **Composer re-serializes Markdown.** `tiptap-markdown` writes its own form: `* item` becomes `- item`, and a literal `_` or `*` in plain text may come out escaped (`foo\_bar`). The agent reads both the same. The user accepted this. Edit-last loads the message's Markdown into the editor unchanged; the editor then shows it formatted.
- **A separate lighter component, not `MarkdownEditor`.** `MarkdownEditor` saves on blur, uses ⌘Enter to save and Esc to cancel, and has a formatting bar. The composer needs the opposite key map.
- **Shift+Enter starts a paragraph, not a line break.** List and quote input rules only fire at the start of a block. After a line break, `- ` would stay plain text. So a single Shift+Enter puts a blank line between the two lines in the sent Markdown, which renders the same in the bubble.
- **Enter on a lone fence opens a code block instead of sending.** A message that ends in a bare `` ``` `` line is never meant to go out like that, so Enter there means "start the code block" (found in tophat).
- **jsdom for editor tests.** The existing tests render static markup, where TipTap renders nothing. `jsdom` is a new dev dependency. Only `ChatMarkdownInput.test.tsx` and `ChatComposer.dom.test.tsx` opt in, with `// @vitest-environment jsdom`. The shared helpers are in `src/test-dom.tsx`.
- **`data-placeholder` on the editor wrapper.** Static markup has no editor, so the wrapper also carries the placeholder. The AgentView greeting test checks it there. The textbox has `aria-label` and `aria-placeholder`.

## Edge cases
- IME composition: Enter and ↑ do nothing while composing.
- An empty editor serializes to `""`, so `hasContent` and the Send button behave as before.
- A message that is only issue IDs or a URL still renders as one line in the bubble.
- Global shortcuts already ignore keys in a contenteditable (`utils/keyboard.ts`).
- ↓ at the end of a code block also leaves it (StarterKit's default).

## Acceptance Criteria
- [ ] A sent user message with Markdown shows rendered in the transcript (popup and full view), and so does a steer message.
- [ ] Issue IDs in user messages show as chips.
- [ ] (Bonus) The composer formats Markdown as the user types, including fenced code blocks opened and closed with `` ``` ``. The text sent to the agent is plain Markdown.
- [ ] Enter, Shift+Enter, drafts, attachments, queue/steer and edit-last-message work the same as before.
- [ ] `make test` passes.
