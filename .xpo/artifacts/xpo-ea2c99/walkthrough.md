# Walkthrough: Agent chat page load sometimes crashes with "The editor view is not available"

## The problem in one paragraph
Since xpo-c35d83, the chat composer has been a TipTap 3 editor. In TipTap 3, `editor.view` is a stand-in until `EditorContent` mounts the real ProseMirror view, and the stand-in throws on most property reads. The `editor` object exists before that, so any `editor.view.focus()` call that lands too early reaches the error boundary. One thing made this window open on every page load: `AgentView` renders the composer with the new-chat placeholder, then switches to "Reply…" once the chat's history arrives. `placeholder` was a dependency of the editor's `extensions`, so `useEditor` destroyed the editor that had just been focused and built a new one. That rebuild dropped the focus, and a focus call made during the rebuild crashed.

## What changed

### 1. The editor is no longer rebuilt for a placeholder (`ChatMarkdownInput.tsx`)
This change removes the window itself.
- `extensions` is memoized once, with no dependencies.
- The editor's `aria-placeholder` attribute is the single source of truth for the placeholder text. `Placeholder` is configured with a function, `placeholderOf(editor)`, that reads it from `editor.options.editorProps.attributes`.
- An effect on `[editor, placeholder]` calls `editor.setOptions({ editorProps: { ...current, attributes: attributesFor(placeholder) } })`. `setOptions` sets the view's props and calls `updateState`, and that recomputes decorations. So the placeholder decoration and the `aria-label`/`aria-placeholder` attributes update in place. It only runs when the view is mounted, because `setOptions` touches the view.

**Why not a ref for the text?** The React refs lint rule (`react-hooks/refs`) flags a ref read inside a function passed to an extension during render. Reading the attribute avoids the ref and keeps one source of truth.

**Why not `useEditor(..., [])` and let TipTap sync the options?** With empty deps, `useEditor` compares options on every render and calls `setOptions` when they differ. `content: value` changes on each keystroke, so the composer would redraw the view on every keystroke. The explicit effect redraws only when the placeholder changes.

### 2. Focus and blur wait for a mounted view (safety net)
`web/src/components/tiptap-view.ts` is shared by both TipTap editors:
- `viewMounted(editor)` is `!editor.isDestroyed`. TipTap 3 defines `isDestroyed` as `editorView?.isDestroyed ?? true`, so it is true until a view exists.
- `whenViewMounted(editor, fn)` runs `fn` now, or once on the editor's `mount` event. It returns a cancel function.
- `focusAt(editor, click)` focuses the editor and places the caret at the click's coordinates through `whenViewMounted`. `MarkdownEditor`'s click effect returns it, so the effect's cleanup cancels a placement that is still waiting.

`ChatMarkdownInput`'s imperative handle no longer reads `editor.view` blindly:
- `focus()` focuses right away when the view is mounted. Otherwise it sets `wantFocus`.
- An effect on `editor` applies a wanted focus right away if the view is already mounted, and on every `mount` event.
- `blur()` clears `wantFocus` and blurs only a mounted view.

### 3. No placeholder flicker for existing chats (`chat-thread-utils.ts`)
With the rebuild gone, the placeholder switch was still visible as a flicker. `asksFirstMessage(thread, chatId)` returns true only for a new chat (no `chatId`), or for a chat whose history has loaded (`thread.loaded`) and has nothing in it. While an existing chat loads, the composer counts it as a reply. `AgentView` and `ChatPopup` choose their placeholder with it. The other uses of an empty thread (the home screen, the popup's suggestions) still use `isEmptyThread`.

## Non-obvious bits
- **The view mounts twice over.** With `immediatelyRender`, TipTap mounts the view into a detached `div` when the editor is created, and `EditorContent` moves the DOM onto the page in `componentDidMount`. "Mounted" doesn't mean "on the page". The first version only guarded mounting, and tophat showed that focus still went missing. The real fix was to stop rebuilding the editor (section 1).
- **Testing "before mount".** In jsdom, `new Editor()` without an element still mounts, because TipTap's default element is a fresh `div`. The helper tests pass `element: null`. The composer tests call `editor.unmount()` to get back into the not-mounted state.

## Acceptance criteria
- [x] Reloading the Agent view repeatedly never shows the error boundary. The user tophatted it and didn't see "Something went wrong".
- [x] The composer still gets focus on load once the editor is mounted. The first tophat showed focus missing, which led to the rebuild fix. The second tophat confirmed focus on load.
- [x] Opening an existing chat shows "Reply…" right away, with no flash of the new-chat placeholder. Confirmed in the final tophat ("lgtm works").
- [x] A test calls `focus()`/`blur()` before the view is mounted and nothing throws: `ChatMarkdownInput.test.tsx`, "does not throw on focus or blur before the view is mounted". Before the fix it failed with the production error message.
- [x] A test shows that a `focus()` call made before mount focuses the editor once it mounts, and that a `blur()` call in between cancels it: "focuses once the view mounts when asked before" and "does not focus on mount when blurred after asking".
- [x] A test shows that changing the placeholder keeps the same editor and shows the new placeholder and aria attributes: "keeps its editor when the placeholder changes, and shows the new one". It failed before the change.
- [x] A test covers `asksFirstMessage`: new chat, an existing chat while loading, a loaded empty chat, and a chat with turns. See the `asksFirstMessage` block in `chat-thread-utils.test.ts`.
- [x] `focusAt` doesn't throw before mount, and places the caret after mount: `tiptap-view.test.ts`, "puts the caret where the click was once the view mounts…". The user also confirmed in tophat that clicking into an issue description puts the caret where they clicked.
- [x] `make test` passes: exit 0, lint (`go vet` + eslint) and all frontend and Go tests.
