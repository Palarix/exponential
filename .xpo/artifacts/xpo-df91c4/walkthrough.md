# Walkthrough: Formatting Bar for Markdown Content

## Overview

A floating formatting toolbar appears when text is selected in any TipTap-based MarkdownEditor. It provides visual formatting controls as an alternative to keyboard shortcuts and raw markdown syntax.

## New file: `web/src/components/FormattingBar.tsx`

This is the core of the feature. It receives the TipTap `editor` instance and a `menuRef` (owned by MarkdownEditor) as props.

### BubbleMenu setup

The component renders a TipTap `BubbleMenu` (from `@tiptap/react/menus`), which handles showing/hiding based on the editor's selection state and positioning via Floating UI.

Three configuration choices worth understanding:

1. **`appendTo={appendToBody}`** — the menu element is appended to `document.body` instead of the editor's parent. Without this, the menu gets clipped by overflow containers (e.g. the modal body's `overflow-y: auto`). The `appendToBody` function is defined outside the component as a stable reference to prevent the BubbleMenu plugin from re-dispatching on every render.

2. **`style={{ zIndex: 100 }}`** — the New Issue modal uses `z-50`; the menu must float above it.

3. **`options={bubbleMenuOptions}`** with `inline: true` — Floating UI's `inline` middleware stabilizes positioning against inline selection rects. Without it, the menu shifts by a pixel or two when formatting changes the text dimensions (e.g. regular → bold).

### Blur handling

Appending to `document.body` breaks the BubbleMenu's built-in blur detection. The default blur handler checks `this.element.parentNode.contains(relatedTarget)` — when the parent is `body`, that's always true, so the menu never hides on blur.

To fix this, a `useEffect` listens for the editor's `blur` event and manually sets `visibility: hidden` on the menu element. Two guards prevent false hides:
- If a dropdown menu is open (`openMenuRef.current` is truthy), blur is ignored — the dropdown interaction needs to complete.
- If the blur target is inside the menu element (`menuRef.current.contains(related)`), it's an internal focus move, not a real blur.

### Toolbar content

The toolbar has two modes, toggled by the `isLinkMode` flag:

**Normal mode** — the full formatting toolbar:
- **Text type**: dropdown button showing the current type icon (Body/H1–H4). Opens a `DropdownPortal` with options. Active type is highlighted with accent color.
- **Inline marks**: Bold, Italic, Strikethrough, Underline — simple toggle buttons via `ToolbarButton`. Each calls `editor.chain().focus().toggle*().run()`.
- **Link**: toggle button that switches to link mode.
- **Block formats**: Quote, Inline Code, Code Block — toggle buttons.
- **List**: dropdown button (Bullet/Numbered/Checklist).

**Link mode** — the toolbar content swaps entirely to a URL input row:
- A link icon, a text input (`w-56 h-7`), a confirm button (checkmark), a remove button (unlink icon, only shown when selection is already a link), and a cancel button (X).
- Enter confirms, Escape cancels. Both return focus to the editor.
- This inline approach avoids focus/blur issues that plagued an earlier dropdown-based design. The link input is inside the BubbleMenu's DOM element, so MarkdownEditor's blur handler recognizes it as "still inside the editor" via `bubbleMenuRef`.

### DropdownPortal

Text type and list dropdowns render via `DropdownPortal` — a small component that uses `createPortal` to render to `document.body` with `position: fixed` and `zIndex: 200`. This avoids clipping from the BubbleMenu's container. It positions itself below the trigger button using `getBoundingClientRect()` and closes on outside clicks.

### ToolbarButton

A reusable component for simple toggle buttons. Uses `onMouseDown` with `e.preventDefault()` instead of `onClick` — this prevents the button from stealing focus from the editor. Wraps in a `Tooltip` for keyboard shortcut hints.

## Changes to `web/src/components/MarkdownEditor.tsx`

### Underline extension

Added `@tiptap/extension-underline` (new dependency) to the extensions array. StarterKit doesn't include Underline.

### FormattingBar rendering

The `FormattingBar` component is rendered alongside `EditorContent` inside the blur-handling div:

```tsx
{editor && <FormattingBar editor={editor} menuRef={bubbleMenuRef} />}
<EditorContent editor={editor} />
```

### Blur guard for BubbleMenu

A `bubbleMenuRef` is created in MarkdownEditor and passed to FormattingBar. The blur handler checks it:

```tsx
if (bubbleMenuRef.current?.contains(e.relatedTarget as Node)) return;
```

This prevents the editor from saving and exiting edit mode when focus moves to the BubbleMenu (e.g. the link URL input). Without this, clicking the link button in the IssueDetail description editor would immediately exit edit mode.

### onTransaction for formatting changes

Previously, the `dirty` ref was only set by `handleTextInput`, `handlePaste`, and `handleDrop`. Programmatic changes from the formatting bar (like `setLink`) never set it, so `onUpdate` skipped calling `onChange` and the changes were lost on blur.

The fix adds an `onTransaction` handler:

```tsx
onTransaction: ({ transaction }) => {
  if (transaction.docChanged) dirty.current = true;
},
```

Now any transaction that changes the document (including formatting commands) sets `dirty`, ensuring `onUpdate` fires `onChange` and the parent component receives the updated markdown.

## What was left out

Comments use a plain `<textarea>`, not the TipTap MarkdownEditor. A follow-up issue (xpo-7d3c63) tracks migrating comments to MarkdownEditor so they also get the formatting bar.
