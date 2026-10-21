import { useEffect, useImperativeHandle, useMemo, useRef, type Ref } from "react";
import { useEditor, EditorContent, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { Selection, type Transaction } from "@tiptap/pm/state";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import { Markdown as TiptapMarkdown } from "tiptap-markdown";

export interface ChatMarkdownInputHandle {
  focus(): void;
  blur(): void;
}

interface ChatMarkdownInputProps {
  /** The draft, as Markdown. */
  value: string;
  /** The draft after each edit, as Markdown. */
  onChange(markdown: string): void;
  placeholder: string;
  /** Runs first on each keydown; returns true when it handled the key. */
  onKeyDown?(event: KeyboardEvent): boolean;
  /** Files on the clipboard; the text, if any, is not pasted. */
  onPasteFiles?(files: File[]): void;
  className?: string;
  ref?: Ref<ChatMarkdownInputHandle>;
}

function markdownOf(editor: Editor): string {
  return (editor.storage as unknown as { markdown: { getMarkdown(): string } }).markdown.getMarkdown();
}

/** Each draft the input shows opens with the caret at its end. */
function caretToEnd({ tr }: { tr: Transaction }): boolean {
  tr.setSelection(Selection.atEnd(tr.doc));
  return true;
}

/**
 * Enter on a line of only ``` or ```lang opens a code block there. TipTap's
 * own rule waits for a space after the fence; a lone fence is never meant
 * to be sent.
 */
function openFence(editor: Editor): boolean {
  const { $from, empty } = editor.state.selection;
  if (!empty || $from.parent.type.name !== "paragraph" || $from.parentOffset !== $from.parent.content.size) return false;
  const m = /^```([\w-]*)$/.exec($from.parent.textContent);
  if (!m) return false;
  return editor
    .chain()
    .deleteRange({ from: $from.start(), to: $from.end() })
    .setCodeBlock(m[1] ? { language: m[1] } : undefined)
    .run();
}

/** Shift+Enter after a closing ``` on a code block's last line drops the fence and leaves the block. */
function closeFence(editor: Editor): boolean {
  const { $from, empty } = editor.state.selection;
  if (!empty || $from.parent.type.name !== "codeBlock" || $from.parentOffset !== $from.parent.content.size) return false;
  const text = $from.parent.textContent;
  if (!/(^|\n)```$/.test(text)) return false;
  const cut = text.endsWith("\n```") ? 4 : 3;
  return editor
    .chain()
    .deleteRange({ from: $from.pos - cut, to: $from.pos })
    .exitCode()
    .run();
}

/**
 * Shift+Enter: out of a code block after a closing fence, a new list item
 * in a list, a newline in a code block, else a
 * new paragraph, so `- ` on the next line starts a list. Enter is the
 * composer's: it sends.
 */
function shiftEnter(editor: Editor): boolean {
  if (closeFence(editor)) return true;
  return editor.commands.first(({ commands }) => [
    () => commands.splitListItem("taskItem"),
    () => commands.splitListItem("listItem"),
    () => commands.newlineInCode(),
    () => commands.splitBlock(),
  ]);
}

/**
 * The agent chat's input: formats Markdown as the user types and reports
 * plain Markdown. It follows value like a textarea: a value it did not
 * report itself (another draft, an edit, a cleared input) replaces the content.
 */
export default function ChatMarkdownInput({ value, onChange, placeholder, onKeyDown, onPasteFiles, className, ref }: ChatMarkdownInputProps) {
  const callbacks = useRef({ onChange, onKeyDown, onPasteFiles });
  useEffect(() => {
    callbacks.current = { onChange, onKeyDown, onPasteFiles };
  }, [onChange, onKeyDown, onPasteFiles]);
  // The Markdown the editor holds, so value changes from outside stand out.
  const shown = useRef(value);

  const extensions = useMemo(
    () => [
      StarterKit.configure({ heading: { levels: [1, 2, 3] }, link: { openOnClick: false } }),
      Placeholder.configure({ placeholder }),
      TaskList,
      TaskItem.configure({ nested: true }),
      TiptapMarkdown.configure({ html: false, breaks: true, transformPastedText: true, transformCopiedText: true }),
    ],
    [placeholder],
  );

  const editor = useEditor(
    {
      extensions,
      content: value,
      immediatelyRender: typeof window !== "undefined",
      editorProps: {
        attributes: { class: "outline-none", "aria-label": placeholder, "aria-placeholder": placeholder, "aria-multiline": "true", role: "textbox" },
        handleKeyDown: (view, event) => {
          const ed = (view.dom as HTMLElement & { editor?: Editor }).editor;
          const enter = event.key === "Enter" && !event.isComposing && !event.altKey && !event.metaKey && !event.ctrlKey;
          if (enter && ed && openFence(ed)) return true;
          if (callbacks.current.onKeyDown?.(event)) return true;
          if (enter && event.shiftKey) return !!ed && shiftEnter(ed);
          return false;
        },
        handlePaste: (_view, event) => {
          const files = Array.from(event.clipboardData?.files ?? []);
          if (files.length === 0 || !callbacks.current.onPasteFiles) return false;
          callbacks.current.onPasteFiles(files);
          return true;
        },
        // Dropped files are the composer's; it turns them into attachments.
        handleDrop: (_view, event) => (event.dataTransfer?.files.length ?? 0) > 0,
      },
      onCreate: ({ editor: ed }) => ed.commands.command(caretToEnd),
      onUpdate: ({ editor: ed }) => {
        const md = markdownOf(ed);
        shown.current = md;
        callbacks.current.onChange(md);
      },
    },
    [extensions],
  );

  useEffect(() => {
    if (!editor || value === shown.current) return;
    shown.current = value;
    editor.chain().setContent(value, { emitUpdate: false }).command(caretToEnd).run();
  }, [editor, value]);

  useImperativeHandle(
    ref,
    () => ({
      // Right away, as a textarea does; the focus and blur commands wait a frame.
      focus: () => editor?.view.focus(),
      blur: () => editor?.view.dom.blur(),
    }),
    [editor],
  );

  // The wrapper names the placeholder too: before the editor exists, as in static markup, it is all there is.
  return <EditorContent editor={editor} className={className} data-placeholder={placeholder} />;
}
