import { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { BubbleMenu } from "@tiptap/react/menus";
import type { Editor } from "@tiptap/react";
import {
  Bold, Italic, Strikethrough, Underline, Link, Unlink, Quote, Code,
  CodeSquare, Heading1, Heading2, Heading3, Heading4, Type,
  List, ListOrdered, ListChecks, ChevronDown, Check, X,
} from "lucide-react";
import Tooltip from "./ui/Tooltip";

interface FormattingBarProps {
  editor: Editor;
  menuRef: React.RefObject<HTMLDivElement | null>;
}

type OpenMenu = "textType" | "list" | "link" | null;

const textTypes = [
  { label: "Body Text", command: (e: Editor) => e.chain().focus().setParagraph().run(), check: (e: Editor) => e.isActive("paragraph") && !e.isActive("heading"), icon: Type },
  { label: "Heading 1", command: (e: Editor) => e.chain().focus().toggleHeading({ level: 1 }).run(), check: (e: Editor) => e.isActive("heading", { level: 1 }), icon: Heading1 },
  { label: "Heading 2", command: (e: Editor) => e.chain().focus().toggleHeading({ level: 2 }).run(), check: (e: Editor) => e.isActive("heading", { level: 2 }), icon: Heading2 },
  { label: "Heading 3", command: (e: Editor) => e.chain().focus().toggleHeading({ level: 3 }).run(), check: (e: Editor) => e.isActive("heading", { level: 3 }), icon: Heading3 },
  { label: "Heading 4", command: (e: Editor) => e.chain().focus().toggleHeading({ level: 4 }).run(), check: (e: Editor) => e.isActive("heading", { level: 4 }), icon: Heading4 },
];

const listTypes = [
  { label: "Bullet List", command: (e: Editor) => e.chain().focus().toggleBulletList().run(), check: (e: Editor) => e.isActive("bulletList"), icon: List },
  { label: "Numbered List", command: (e: Editor) => e.chain().focus().toggleOrderedList().run(), check: (e: Editor) => e.isActive("orderedList"), icon: ListOrdered },
  { label: "Checklist", command: (e: Editor) => e.chain().focus().toggleTaskList().run(), check: (e: Editor) => e.isActive("taskList"), icon: ListChecks },
];

function ToolbarButton({ icon: Icon, tooltip, active, onClick, className = "" }: {
  icon: typeof Bold;
  tooltip: string;
  active?: boolean;
  onClick: () => void;
  className?: string;
}) {
  return (
    <Tooltip content={tooltip}>
      <button
        type="button"
        onMouseDown={(e) => { e.preventDefault(); onClick(); }}
        className={`flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] transition-colors ${active ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]"} ${className}`}
      >
        <Icon className="w-3.5 h-3.5" />
      </button>
    </Tooltip>
  );
}

function Separator() {
  return <div className="w-px h-4 bg-[var(--color-border-subtle)] mx-1" />;
}

function DropdownPortal({ anchorRef, children, onClose }: {
  anchorRef: React.RefObject<HTMLElement | null>;
  children: React.ReactNode;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ top: 0, left: 0 });

  useEffect(() => {
    if (anchorRef.current) {
      const rect = anchorRef.current.getBoundingClientRect();
      setPos({ top: rect.bottom + 4, left: rect.left });
    }
  }, [anchorRef]);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node) &&
          anchorRef.current && !anchorRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose, anchorRef]);

  return createPortal(
    <div ref={ref} style={{ position: "fixed", top: pos.top, left: pos.left, zIndex: 200 }} className="min-w-40 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1">
      {children}
    </div>,
    document.body,
  );
}

const appendToBody = () => document.body;
const bubbleMenuOptions = { placement: "top" as const, offset: 8, inline: true };

export default function FormattingBar({ editor, menuRef }: FormattingBarProps) {
  const [openMenu, setOpenMenu] = useState<OpenMenu>(null);
  const [linkUrl, setLinkUrl] = useState("");
  const linkInputRef = useRef<HTMLInputElement>(null);
  const textTypeBtnRef = useRef<HTMLButtonElement>(null);
  const listBtnRef = useRef<HTMLButtonElement>(null);
  const openMenuRef = useRef<OpenMenu>(null);
  useEffect(() => { openMenuRef.current = openMenu; }, [openMenu]);

  const closeMenus = useCallback(() => setOpenMenu(null), []);

  useEffect(() => {
    const handler = () => {
      if (editor.state.selection.empty) setOpenMenu(null);
    };
    editor.on("selectionUpdate", handler);
    return () => { editor.off("selectionUpdate", handler); };
  }, [editor]);

  useEffect(() => {
    const onBlur = ({ event }: { event: FocusEvent }) => {
      if (openMenuRef.current) return;
      const related = event.relatedTarget as Node | null;
      if (related && menuRef.current?.contains(related)) return;
      if (menuRef.current) menuRef.current.style.visibility = "hidden";
    };
    editor.on("blur", onBlur);
    return () => { editor.off("blur", onBlur); };
  }, [editor, menuRef]);

  const toggleMenu = useCallback((menu: OpenMenu) => {
    setOpenMenu((prev) => prev === menu ? null : menu);
    if (menu === "link") {
      const existing = editor.getAttributes("link").href || "";
      setLinkUrl(existing);
    }
  }, [editor]);

  useEffect(() => {
    if (openMenu === "link" && linkInputRef.current) linkInputRef.current.focus();
  }, [openMenu]);

  const applyLink = useCallback(() => {
    const url = linkUrl.trim();
    if (url) {
      editor.chain().focus().setLink({ href: url }).run();
    } else {
      editor.chain().focus().unsetLink().run();
    }
    setOpenMenu(null);
    setLinkUrl("");
  }, [editor, linkUrl]);

  const removeLink = useCallback(() => {
    editor.chain().focus().unsetLink().run();
    setOpenMenu(null);
    setLinkUrl("");
  }, [editor]);

  const activeTextType = textTypes.find((t) => t.check(editor));
  const ActiveTextIcon = activeTextType?.icon || Type;

  const activeListType = listTypes.find((t) => t.check(editor));
  const ActiveListIcon = activeListType?.icon || List;

  const isLinkMode = openMenu === "link";

  return (
    <BubbleMenu
      ref={menuRef}
      editor={editor}
      appendTo={appendToBody}
      style={{ zIndex: 100 }}
      options={bubbleMenuOptions}
    >
      <div className="flex items-center gap-0.5 p-1 bg-[var(--color-surface-3)] border-1 border-[var(--color-border-overlay)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)]">
        {isLinkMode ? (
          <>
            <Link className="w-3.5 h-3.5 text-[var(--color-text-muted)] ml-1 shrink-0" />
            <input
              ref={linkInputRef}
              type="url"
              value={linkUrl}
              onChange={(e) => setLinkUrl(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") { e.preventDefault(); applyLink(); }
                if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); closeMenus(); editor.commands.focus(); }
              }}
              placeholder="Paste or type a URL..."
              className="w-56 h-7 text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none px-1.5"
            />
            <Tooltip content="Apply">
              <button type="button" onMouseDown={(e) => { e.preventDefault(); applyLink(); }} className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] text-[var(--color-accent-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
                <Check className="w-3.5 h-3.5" />
              </button>
            </Tooltip>
            {editor.isActive("link") && (
              <Tooltip content="Remove link">
                <button type="button" onMouseDown={(e) => { e.preventDefault(); removeLink(); }} className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] text-[var(--color-error)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
                  <Unlink className="w-3.5 h-3.5" />
                </button>
              </Tooltip>
            )}
            <Tooltip content="Cancel">
              <button type="button" onMouseDown={(e) => { e.preventDefault(); closeMenus(); editor.commands.focus(); }} className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">
                <X className="w-3.5 h-3.5" />
              </button>
            </Tooltip>
          </>
        ) : (
          <>
            {/* Text type dropdown */}
            <Tooltip content="Text type">
              <button
                ref={textTypeBtnRef}
                type="button"
                onMouseDown={(e) => { e.preventDefault(); toggleMenu("textType"); }}
                className={`flex items-center gap-0.5 h-7 px-1.5 rounded-[var(--radius-sm)] transition-colors text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] ${openMenu === "textType" ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]" : ""}`}
              >
                <ActiveTextIcon className="w-3.5 h-3.5" />
                <ChevronDown className="w-2.5 h-2.5" />
              </button>
            </Tooltip>
            {openMenu === "textType" && (
              <DropdownPortal anchorRef={textTypeBtnRef} onClose={closeMenus}>
                {textTypes.map((t) => (
                  <button
                    key={t.label}
                    type="button"
                    onMouseDown={(e) => { e.preventDefault(); t.command(editor); closeMenus(); }}
                    className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${t.check(editor) ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                  >
                    <t.icon className="w-4 h-4" />
                    {t.label}
                  </button>
                ))}
              </DropdownPortal>
            )}

            <Separator />

            {/* Inline marks */}
            <ToolbarButton icon={Bold} tooltip="Bold (⌘B)" active={editor.isActive("bold")} onClick={() => editor.chain().focus().toggleBold().run()} />
            <ToolbarButton icon={Italic} tooltip="Italic (⌘I)" active={editor.isActive("italic")} onClick={() => editor.chain().focus().toggleItalic().run()} />
            <ToolbarButton icon={Strikethrough} tooltip="Strikethrough (⌘⇧S)" active={editor.isActive("strike")} onClick={() => editor.chain().focus().toggleStrike().run()} />
            <ToolbarButton icon={Underline} tooltip="Underline (⌘U)" active={editor.isActive("underline")} onClick={() => editor.chain().focus().toggleUnderline().run()} />

            <Separator />

            {/* Link */}
            <ToolbarButton icon={Link} tooltip="Link (⌘K)" active={editor.isActive("link")} onClick={() => toggleMenu("link")} />

            <Separator />

            {/* Block formats */}
            <ToolbarButton icon={Quote} tooltip="Quote" active={editor.isActive("blockquote")} onClick={() => editor.chain().focus().toggleBlockquote().run()} />
            <ToolbarButton icon={Code} tooltip="Inline Code (⌘E)" active={editor.isActive("code")} onClick={() => editor.chain().focus().toggleCode().run()} />
            <ToolbarButton icon={CodeSquare} tooltip="Code Block" active={editor.isActive("codeBlock")} onClick={() => editor.chain().focus().toggleCodeBlock().run()} />

            <Separator />

            {/* List dropdown */}
            <Tooltip content="List">
              <button
                ref={listBtnRef}
                type="button"
                onMouseDown={(e) => { e.preventDefault(); toggleMenu("list"); }}
                className={`flex items-center gap-0.5 h-7 px-1.5 rounded-[var(--radius-sm)] transition-colors text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] ${openMenu === "list" || activeListType ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]" : ""}`}
              >
                <ActiveListIcon className="w-3.5 h-3.5" />
                <ChevronDown className="w-2.5 h-2.5" />
              </button>
            </Tooltip>
            {openMenu === "list" && (
              <DropdownPortal anchorRef={listBtnRef} onClose={closeMenus}>
                {listTypes.map((t) => (
                  <button
                    key={t.label}
                    type="button"
                    onMouseDown={(e) => { e.preventDefault(); t.command(editor); closeMenus(); }}
                    className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${t.check(editor) ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                  >
                    <t.icon className="w-4 h-4" />
                    {t.label}
                  </button>
                ))}
              </DropdownPortal>
            )}
          </>
        )}
      </div>
    </BubbleMenu>
  );
}
