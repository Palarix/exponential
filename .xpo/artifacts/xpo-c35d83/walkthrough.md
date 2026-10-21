# Walkthrough: user messages render as Markdown, composer formats Markdown as you type

## What changed
- In the chat transcript, the user's own messages, and the messages they steer into a running turn, render as Markdown with issue ID chips, like the agent's replies.
- The composer is now a small TipTap editor. It formats Markdown as the user types and still hands plain Markdown to the agent.

## How the pieces fit

```
ChatComposer ── value (Markdown) ──▶ ChatMarkdownInput (TipTap + tiptap-markdown)
     ▲                                    │
     └──── onChange(Markdown) ◀───────────┘
     │
     └─ drafts store (composer-drafts.ts, unchanged: still Markdown strings)
```

### Transcript (`ChatThread.tsx`, `index.css`)
- The user bubble and the steer bubble render `IssueMarkdown`, the same renderer as `AnswerText`. It brings GFM, `remark-breaks` (a single newline stays a line break) and issue chips.
- A new `.prose-bubble` modifier sets a 1.25rem line height and drops the last block's bottom margin. Without it, prose's 1.75 line height would make every one-line bubble taller.
- The queued-message chips, the agent-turn line and the turn rail's preview still show raw text on one line. They are summaries, not messages.

### Composer input (`ChatMarkdownInput.tsx`)
- **A controlled editor.** TipTap owns its document, but the draft store owns the text. The input keeps `shown`, the last Markdown it reported or loaded. When `value` differs from `shown`, the change came from outside: a chat switch, a prefill, edit-last, a queued message pulled back, or a cleared draft after send. It then calls `setContent` without an update event and puts the caret at the end. When `value` is its own echo, nothing happens. Without this check, every keystroke would re-parse the document and move the caret.
- **Key order in `handleKeyDown`:**
  1. **Code fences.** Enter or Shift+Enter on a line that is only "```" or "```lang" opens a code block. TipTap's own rule fires only on a space after the fence, and Enter would have sent the bare fence (found in tophat).
  2. **`ChatComposer`'s handler.** Enter sends, ↑ edits or pulls from the queue, and Esc cancels edit or blurs. This is the old textarea logic, moved to native `KeyboardEvent`s. It now returns whether it took the key.
  3. **Shift+Enter**, in this order:
     - out of a code block after a closing "```"
     - a new list item in a list
     - a newline in a code block
     - otherwise a new paragraph
- **Shift+Enter makes a paragraph, not a `<br>`.** TipTap's list and quote input rules only fire at the start of a block. After a hard break, `- ` would stay plain text. The cost: one Shift+Enter puts a blank line in the sent Markdown, which renders the same.
- **Focus.** The ref handle's `focus` and `blur` call the view and the DOM directly. TipTap's commands wait a frame, and the composer's focus-after-send needs to act at once, as it did on the textarea.
- **Paste and drop.** Clipboard files go to `onPasteFiles` and become attachments. Text pastes as Markdown. Dropped files are refused by ProseMirror, so the composer's existing `onDrop` attaches them.
- **Placeholder.** It is set as `aria-label`/`aria-placeholder` on the textbox. The wrapper also carries `data-placeholder`, because static markup (and so the AgentView test) has no editor.

### Elsewhere
- `typing()` in `chat-thread-utils.ts` decides whether an approval or question card may take focus. It knew only textareas and inputs. It now also counts a contenteditable with text, or a card would steal focus mid-sentence. It reads the `contenteditable` attribute, since jsdom has no `isContentEditable`.
- Global shortcuts already skip contenteditable targets (`utils/keyboard.ts`), so typing `g` in the composer doesn't fire g-chords.

## Decisions
- **Re-serialized Markdown is fine.** `tiptap-markdown` writes its own form (`* item` → `- item`, `foo_bar` may become `foo\_bar`). The agent reads both the same. The user accepted this.
- **A new component, not `MarkdownEditor`.** `MarkdownEditor` saves on blur, uses ⌘Enter and Esc for save and cancel, and has a formatting bar. The composer needs the opposite key map and a slim box.
- **jsdom for two test files.** The suite renders static markup, where TipTap renders nothing. `jsdom` is a new dev dependency. Only `ChatMarkdownInput.test.tsx` and `ChatComposer.dom.test.tsx` opt in (`// @vitest-environment jsdom`). They share helpers in `src/test-dom.tsx`: layout stubs for ProseMirror, mounting, typing through input rules, and key presses.

## Follow-up
- xpo-7fdb71: syntax highlighting for code blocks in all Markdown.

## Acceptance criteria
- [x] A sent user message with Markdown shows rendered in the transcript (popup and full view), and so does a steer message. *Evidence:* "ChatThread user message Markdown" tests (lists, bold, code, line breaks, no `whitespace-pre-wrap`); the popup and the Agent view share `ChatThread`; user tophat.
- [x] Issue IDs in user messages show as chips. *Evidence:* both bubble tests assert `data-issue-ref`.
- [x] (Bonus) The composer formats Markdown as the user types, including fenced code blocks opened and closed with "```". The text sent to the agent is plain Markdown. *Evidence:* `ChatMarkdownInput.test.tsx` (bold, lists, open and close fences, Markdown reported); "sends the Markdown on Enter" composer test; user tophat ("composer understands markdown and is nice to use").
- [x] Enter, Shift+Enter, drafts, attachments, queue/steer and edit-last-message work the same as before. *Evidence:* `ChatComposer.dom.test.tsx` (per-chat drafts, Enter sends and clears, Shift+Enter, IME, ↑ edit-last, ↑ pulls the queue, edit loads Markdown, Esc cancels, edit sends as edit, Esc blurs); the paste-files test; the existing queue and attachment markup tests.
- [x] `make test` passes. *Evidence:* exit 0, 1,484 frontend tests plus the Go suite.
