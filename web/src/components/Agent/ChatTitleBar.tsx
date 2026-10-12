import { useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import { deleteChat, renameChat, type ChatMeta } from "../../api/chat";
import { Button, ConfirmDialog, Menu, MenuItem, Popover, useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";

interface ChatTitleBarProps {
  /** The chat; undefined for a new chat, which has no menu. */
  chat: ChatMeta | undefined;
  /** The title as shown when not renaming, e.g. a switcher button. */
  title: ReactNode;
  /** Runs after the chat is deleted. */
  onDeleted?: (id: string) => void;
}

/** A chat title that turns into an input for Rename, plus the "⋯" menu with Rename and Delete. */
export default function ChatTitleBar({ chat, title, onDeleted }: ChatTitleBarProps) {
  const { chatDeleted } = useAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();
  const menuBtnRef = useRef<HTMLButtonElement>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [draft, setDraft] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const saveTitle = async () => {
    if (!chat || draft === null) return;
    const next = draft.trim();
    setDraft(null);
    if (next === "" || next === chat.title) return;
    try {
      await renameChat(chat.id, next);
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
    } catch {
      showToast("Could not rename the chat", { variant: "error" });
    }
  };

  const remove = async () => {
    if (!chat) return;
    setConfirmDelete(false);
    try {
      await deleteChat(chat.id);
      chatDeleted(chat.id);
      onDeleted?.(chat.id);
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
    } catch {
      showToast("Could not delete the chat", { variant: "error" });
    }
  };

  return (
    <div className="flex items-center gap-1 min-w-0 flex-1">
      {draft !== null ? (
        <input
          autoFocus
          aria-label="Chat title"
          value={draft}
          maxLength={60}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => setDraft(null)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              void saveTitle();
            } else if (e.key === "Escape") {
              e.preventDefault();
              e.stopPropagation();
              setDraft(null);
            }
          }}
          className="flex-1 min-w-0 h-7 px-2 -ml-2 rounded-[var(--radius-sm)] bg-[var(--color-surface-2)] border border-[var(--color-border-focus)] text-sm font-medium text-[var(--color-text-primary)] outline-none"
        />
      ) : (
        <div className="min-w-0 flex">{title}</div>
      )}

      {chat && draft === null && (
        <button
          ref={menuBtnRef}
          type="button"
          aria-label="Chat actions"
          onClick={() => setMenuOpen((o) => !o)}
          className="shrink-0 p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]"
        >
          <MoreHorizontal className="w-4 h-4" />
        </button>
      )}
      {chat && menuOpen && (
        <Popover anchorRef={menuBtnRef} onClose={() => setMenuOpen(false)}>
          <Menu onClose={() => setMenuOpen(false)} aria-label="Chat actions">
            <MenuItem label="Rename" icon={<Pencil size={14} />} onClick={() => { setMenuOpen(false); setDraft(chat.title); }} />
            <MenuItem label="Delete" icon={<Trash2 size={14} />} destructive onClick={() => { setMenuOpen(false); setConfirmDelete(true); }} />
          </Menu>
        </Popover>
      )}

      {chat && confirmDelete && (
        <ConfirmDialog
          scope="agent.confirm-delete"
          title="Delete this chat?"
          description={`"${chat.title || "Untitled chat"}" will be deleted. This cannot be undone.`}
          onCancel={() => setConfirmDelete(false)}
        >
          <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(false)}>Cancel</Button>
          <Button variant="danger" size="sm" onClick={() => void remove()}>Delete</Button>
        </ConfirmDialog>
      )}
    </div>
  );
}
