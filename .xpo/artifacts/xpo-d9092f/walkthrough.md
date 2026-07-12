## What changed

`web/src/components/MarkdownEditor.tsx` — replaced the no-op `handlePaste` with a custom handler that intercepts all paste events and routes them through the markdown layer.

Added import: `DOMParser` from `@tiptap/pm/model` (aliased as `PMDOMParser`).

Added `editorRef` (a `useRef`) to reliably access the Tiptap editor instance from within the `handlePaste` closure.

## Why

The `tiptap-markdown` plugin (v0.9.0, which is the latest available release) configures `transformPastedText: true` via ProseMirror's `clipboardTextParser` hook. However, ProseMirror only invokes `clipboardTextParser` when the clipboard contains **no HTML**. Most paste sources (VS Code, browsers, terminals on macOS, notes apps) include both `text/html` and `text/plain`. When HTML is present, ProseMirror uses its DOM parser — which treats markdown syntax as literal text — and the `clipboardTextParser` hook is never called.

## How it works

The `handlePaste` handler:

1. Reads both `text/html` and `text/plain` from the clipboard.
2. If the HTML contains rich formatting tags (`h1`-`h6`, `strong`, `em`, `a`, `ul`, `ol`, `li`, `table`, `blockquote`, `pre`, `img`, `hr`):
   - Parses the HTML into a temporary ProseMirror document via `PMDOMParser`
   - Serializes that document to markdown via `tiptap-markdown`'s serializer
   - Re-parses the markdown back to HTML via the markdown parser
   - Inserts the result (this roundtrip normalizes the content, stripping unsupported elements)
3. Otherwise (plain text or thin HTML wrapper):
   - Parses the `text/plain` directly as markdown
4. In both cases, the final HTML is converted to a ProseMirror `Slice` and inserted via `view.dispatch(tr.replaceSelection(slice))`, bypassing tiptap-markdown's overridden `insertContentAt` command (which would double-parse the content).

## Edge cases

- **No text/plain and no HTML** (e.g. image paste): returns `false` to let ProseMirror handle it natively.
- **Copy/paste within the editor**: `transformCopiedText: true` serializes the selection to markdown on copy; our handler then parses it back on paste — clean roundtrip.
- **Shift+paste**: not intercepted by this handler (ProseMirror suppresses `handlePaste` for shift-pastes on some browsers; on others the plain text path handles it correctly).