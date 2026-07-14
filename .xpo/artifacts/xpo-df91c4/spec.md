# Formatting Bar for Markdown Content

## Problem

The MarkdownEditor (TipTap-based) has no visible formatting controls. Users must know markdown syntax or keyboard shortcuts to apply formatting. A floating toolbar on text selection would make formatting discoverable and accessible.

## Requirements

1. **Show on text selection** — a floating toolbar appears above the current selection when the user selects text in any MarkdownEditor instance.
2. **Hide when selection is empty** — the toolbar disappears when the selection collapses or the editor loses focus.
3. **Formatting actions**:
   - **Text type** dropdown: Body Text, Heading 1, Heading 2, Heading 3, Heading 4
   - **Inline marks**: Bold, Italic, Strikethrough, Underline (toggle buttons)
   - **Link**: toggle, with a URL input popover when activating
   - **Quote**: toggle blockquote
   - **Inline Code**: toggle code mark
   - **Code Block**: toggle code block
   - **List** dropdown: Bullet List, Numbered List, Checklist
4. **Active state** — buttons reflect whether the current selection already has that formatting applied (highlighted/active appearance).
5. **Tooltips** — each button shows a tooltip with the action name and keyboard shortcut (where applicable).
6. **Visual separators** — logical groups separated by thin vertical dividers (inline marks | link | block formats | lists).

## Design Decisions

### TipTap BubbleMenu

Use TipTap's `BubbleMenu` component (exported from `@tiptap/react`). It handles:
- Positioning relative to the selection (above by default)
- Showing/hiding based on selection state
- Repositioning on scroll/resize

This avoids building custom selection-tracking and positioning logic.

### Separate component

Create `FormattingBar.tsx` as a standalone component that receives the TipTap `editor` instance as a prop. This keeps `MarkdownEditor.tsx` focused on editor lifecycle and the formatting bar self-contained.

### Underline extension

Install `@tiptap/extension-underline`. StarterKit doesn't include it. Note: markdown has no native underline syntax, but `tiptap-markdown` will serialize it as `<u>` HTML tags inline, which round-trips correctly.

### Link input

When the user clicks the Link button:
- If text is already a link → unset the link.
- If text is not a link → show a small popover below the button with a URL text input. Enter confirms, Escape cancels. The popover should not interfere with the BubbleMenu positioning.

### Text type and List dropdowns

These open a small dropdown panel below the respective button. Selecting an option applies the command and closes the dropdown. Only one dropdown can be open at a time.

### Blur handling

Clicking toolbar buttons must not trigger the editor's `onBlur` → `onSave` behavior. The `BubbleMenu` renders inside the editor's DOM container by default, so clicks on it should be caught by the existing `e.currentTarget.contains(e.relatedTarget)` blur guard in `MarkdownEditor.tsx`. If not, the `suppressBlurSave` ref can be used.

## Implementation Outline

### 1. Install dependency

```
bun add @tiptap/extension-underline
```

### 2. `web/src/components/FormattingBar.tsx`

- Import `BubbleMenu` from `@tiptap/react` and icons from `lucide-react`.
- Import `Tooltip` from `./ui/Tooltip`.
- Accept `editor` as a prop (type: `Editor` from `@tiptap/react`).
- Render a `<BubbleMenu>` containing icon buttons grouped with separators.
- Each button calls the corresponding `editor.chain().focus().toggle*().run()` command.
- Use `editor.isActive(...)` to set active styling on buttons.
- Text type button: shows current type label (e.g. "Body", "H1"), opens a dropdown on click.
- List button: icon button, opens a dropdown with three list options.
- Link button: toggles between set/unset. When setting, renders a small URL input below.
- Manage local state for open dropdowns (`textType | list | link | null`).

### 3. `web/src/components/MarkdownEditor.tsx`

- Import `Underline` from `@tiptap/extension-underline`.
- Add `Underline` to the extensions array.
- Import and render `<FormattingBar editor={editor} />` alongside `<EditorContent>`.

### 4. Styling

- Toolbar container: `bg-[var(--color-surface-3)]`, border, rounded, shadow, `flex items-center gap-0.5 p-1`.
- Buttons: small square ghost buttons (~28px), `rounded-[var(--radius-sm)]`, hover and active states using design tokens.
- Active button: `bg-[var(--color-hover-surface-3)]` with `text-[var(--color-text-primary)]`.
- Separators: `w-px h-4 bg-[var(--color-border-subtle)] mx-1`.
- Dropdowns: absolute positioned below the trigger, same surface/border styling as the toolbar.

## Acceptance Criteria

- [ ] Selecting text in the MarkdownEditor shows the formatting bar above the selection.
- [ ] Deselecting text (click away or collapse selection) hides the bar.
- [ ] Bold, Italic, Strikethrough, Underline toggle correctly and show active state.
- [ ] Text type dropdown changes between body/headings and shows current type.
- [ ] Link button sets/unsets links, with URL input when setting.
- [ ] Quote, Inline Code, Code Block toggle correctly.
- [ ] List dropdown switches between bullet, numbered, and checklist.
- [ ] Formatting bar appears in both IssueDetail description editor and NewIssueModal.
- [ ] Clicking toolbar buttons does not trigger editor blur/save.
- [ ] Tooltips show on hover for each button.
