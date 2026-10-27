## What
Reading `editor.view` before `EditorContent` has mounted the ProseMirror view must not crash the Agent view. A focus that comes too early must run once the view is mounted. The composer must not throw its editor away when only its placeholder changes. Opening an existing chat shows "Reply…" from the start, without first flashing the new-chat placeholder.

## Why
In TipTap 3, `editor.view` is a proxy until the view is mounted, and the proxy throws on most keys. The `editor` object can exist before then: on the first render, and after `useEditor` rebuilds the editor because a dependency changed. `ChatComposer` focuses the input from a mount effect and from `focusSignal`. When either one runs in that window, `view.focus()` throws, and the error boundary takes over the view. The crash only happens sometimes because whether the effect wins depends on timing.

The rebuild happens on every page load. `AgentView` first renders the composer with "Ask a question, or describe what to build…", then switches the placeholder to "Reply…" once the thread has loaded. `placeholder` was a dependency of `extensions`, so `useEditor` destroyed the editor that had just been focused and built a new one. The focus went with the old editor, and the rebuild opened the window for the crash.

The switch itself is a visible flicker. The placeholder came from `isEmptyThread(thread)`, and an existing chat's thread is empty until its history arrives.

## How
- **Placeholder choice (`chat-thread-utils.ts`):** `asksFirstMessage(thread, chatId)` is true for a new chat (no `chatId`), or for a chat whose history has loaded (`thread.loaded`) and has nothing in it. While an existing chat loads, it is false, so the composer says "Reply…". `AgentView` and `ChatPopup` choose their placeholder with it. The other uses of `empty` (the home screen, the popup's suggestions) are unchanged.
- **No rebuild on placeholder change (`ChatMarkdownInput`):**
  - `extensions` is built once.
  - The editor's `aria-placeholder` attribute holds the placeholder. `Placeholder` gets a function that reads that attribute from the editor's options. A ref won't do here: the React refs lint rule forbids it.
  - An effect on `placeholder` calls `editor.setOptions({ editorProps: { ...current, attributes } })` with the new `aria-label`/`aria-placeholder`. `setOptions` updates the view's attributes and redraws it, so the placeholder decoration shows the new text. It runs only when the view is mounted, since `setOptions` touches the view.
- **Readiness check:** in TipTap 3, `editor.isDestroyed` is `true` until a view is mounted (`editorView?.isDestroyed ?? true`). `viewMounted(editor)` = `!editor.isDestroyed`.
- **Shared module** `web/src/components/tiptap-view.ts`:
  - `whenViewMounted(editor, fn): () => void` runs `fn` now if the view is mounted. Otherwise it runs `fn` once on the editor's `mount` event. It returns a cancel function.
  - `focusAt(editor, click)` focuses the editor and puts the caret at the click's coordinates through `whenViewMounted`.
- **Focus and blur (`ChatMarkdownInput`), kept as a safety net:**
  - `focus()`: focus right away if the view is mounted. Otherwise set a `wantFocus` ref.
  - An effect on `editor` applies a wanted focus right away if the view is already mounted, and on every `mount` event.
  - `blur()` clears `wantFocus`, and blurs `view.dom` only if the view is mounted.
- **`MarkdownEditor` click effect:** returns `focusAt(editor, clickEvent)`, so the effect's cleanup cancels a placement that is still waiting.

## Acceptance criteria
- [ ] Reloading the Agent view repeatedly never shows the error boundary
- [ ] The composer still gets focus on load once the editor is mounted
- [ ] Opening an existing chat shows "Reply…" right away, with no flash of the new-chat placeholder
- [ ] A test calls `focus()`/`blur()` before the view is mounted and nothing throws
- [ ] A test shows that a `focus()` call made before mount focuses the editor once it mounts, and that a `blur()` call in between cancels it
- [ ] A test shows that changing the placeholder keeps the same editor and shows the new placeholder and aria attributes
- [ ] A test covers `asksFirstMessage`: new chat, an existing chat while loading, a loaded empty chat, and a chat with turns
- [ ] `focusAt` doesn't throw before mount, and places the caret after mount
- [ ] `make test` passes
