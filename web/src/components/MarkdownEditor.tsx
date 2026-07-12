import { useEditor, EditorContent } from "@tiptap/react";
import { DOMParser as PMDOMParser } from "@tiptap/pm/model";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Link from "@tiptap/extension-link";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import { Table } from "@tiptap/extension-table";
import { TableRow } from "@tiptap/extension-table-row";
import { TableCell } from "@tiptap/extension-table-cell";
import { TableHeader } from "@tiptap/extension-table-header";
import { Markdown as TiptapMarkdown } from "tiptap-markdown";
import { useRef, useCallback, useEffect, useMemo } from "react";

interface MarkdownEditorProps {
  value: string;
  onChange?: (markdown: string) => void;
  onSave?: () => void;
  onCancel?: () => void;
  placeholder?: string;
  autoFocus?: boolean;
  clickEvent?: { clientX: number; clientY: number } | null;
  className?: string;
}

export default function MarkdownEditor({
  value,
  onChange,
  onSave,
  onCancel,
  placeholder = "Add a description...",
  autoFocus = false,
  clickEvent = null,
  className = "",
}: MarkdownEditorProps) {
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;
  const onCancelRef = useRef(onCancel);
  onCancelRef.current = onCancel;
  const suppressBlurSave = useRef(false);
  const dirty = useRef(false);

  const handleSave = useCallback(() => {
    suppressBlurSave.current = true;
    onSaveRef.current?.();
  }, []);

  const editorRef = useRef<ReturnType<typeof useEditor>>(null);
  const initialValue = useRef(value);

  const extensions = useMemo(() => [
    StarterKit.configure({
      heading: { levels: [1, 2, 3, 4] },
    }),
    Placeholder.configure({ placeholder }),
    Link.configure({ openOnClick: false }),
    TaskList,
    TaskItem.configure({ nested: true }),
    Table.configure({ resizable: false }),
    TableRow,
    TableCell,
    TableHeader,
    TiptapMarkdown.configure({
      html: false,
      breaks: true,
      transformPastedText: true,
      transformCopiedText: true,
    }),
  ], [placeholder]);

  const editor = useEditor({
    extensions,
    content: initialValue.current,
    autofocus: autoFocus && !clickEvent ? "end" : false,
    editorProps: {
      attributes: {
        class: "outline-none",
      },
      handleKeyDown: (_view, event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          suppressBlurSave.current = true;
          onCancelRef.current?.();
          return true;
        }
        if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
          event.preventDefault();
          handleSave();
          return true;
        }
        return false;
      },
      handleTextInput: () => { dirty.current = true; return false; },
      handlePaste: (_view, event) => {
        dirty.current = true;
        const ed = editorRef.current;
        if (!ed) return false;
        const clipHtml = event.clipboardData?.getData("text/html") || "";
        const clipText = event.clipboardData?.getData("text/plain") || "";
        if (!clipHtml && !clipText) return false;
        const richTag = /<(h[1-6]|strong|em|b|i|a\s|ul|ol|li|table|blockquote|pre|img|hr)\b/i;
        let insertHtml: string;
        if (clipHtml && richTag.test(clipHtml)) {
          const wrapper = document.createElement("div");
          wrapper.innerHTML = clipHtml;
          const { state } = ed.view;
          const fragment = PMDOMParser.fromSchema(state.schema).parse(wrapper);
          const storage = ed.storage as Record<string, any>;
          const md = storage.markdown.serializer.serialize(fragment) as string;
          insertHtml = storage.markdown.parser.parse(md) as string;
        } else if (clipText) {
          const storage = ed.storage as Record<string, any>;
          insertHtml = storage.markdown.parser.parse(clipText) as string;
        } else {
          return false;
        }
        const wrapper = document.createElement("div");
        wrapper.innerHTML = insertHtml;
        const { state } = ed.view;
        const slice = PMDOMParser.fromSchema(state.schema).parseSlice(wrapper);
        ed.view.dispatch(state.tr.replaceSelection(slice));
        return true;
      },
      handleDrop: () => { dirty.current = true; return false; },
    },
    onUpdate: ({ editor: ed }) => {
      if (!dirty.current) return;
      const md = (ed.storage as Record<string, any>).markdown.getMarkdown() as string;
      onChangeRef.current?.(md);
    },
  });

  editorRef.current = editor;

  useEffect(() => {
    if (!editor || !clickEvent) return;
    editor.commands.focus();
    const pos = editor.view.posAtCoords({ left: clickEvent.clientX, top: clickEvent.clientY });
    if (pos) {
      editor.commands.setTextSelection(pos.pos);
    }
  }, [editor, clickEvent]);

  return (
    <div className={className}>
      <div
        onBlur={(e) => {
          if (suppressBlurSave.current) { suppressBlurSave.current = false; return; }
          if (e.currentTarget.contains(e.relatedTarget)) return;
          onSaveRef.current?.();
        }}
      >
        <EditorContent editor={editor} />
      </div>
      {(onSave || onCancel) && (
        <p className="text-xs text-[var(--color-text-muted)] mt-6 select-none">⌘ Enter to save · Esc to cancel</p>
      )}
    </div>
  );
}
