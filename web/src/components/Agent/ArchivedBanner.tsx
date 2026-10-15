import { Archive, ArchiveRestore } from "lucide-react";
import { Button } from "../ui";
import { useArchiveChat } from "./useArchiveChat";

/** Stands in for the composer of an archived chat, which is read only. */
export default function ArchivedBanner({ chatId, wide }: { chatId: string; wide?: boolean }) {
  const { unarchive } = useArchiveChat();
  return (
    <div className={wide ? "w-full max-w-3xl mx-auto px-4 pb-6" : "px-3 pb-3"}>
      <div role="status" className="flex items-center gap-3 px-4 py-3 rounded-[var(--radius-lg)] border border-[var(--color-border-ai-subtle)] bg-[var(--color-surface-ai-2)]">
        <Archive className="w-4 h-4 shrink-0 text-[var(--color-text-muted)]" />
        <span className="flex-1 min-w-0 text-sm text-[var(--color-text-secondary)]">This chat is archived</span>
        <Button variant="secondary" size="sm" icon={<ArchiveRestore size={14} />} onClick={() => void unarchive([chatId])}>Unarchive</Button>
      </div>
    </div>
  );
}
