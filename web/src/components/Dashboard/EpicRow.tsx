import type { EpicProgress, Issue } from "../../api/client";
import { CopyableId, StatusIcon, SubProgress } from "../ui";

/**
 * One Active Epics row. The row is a div, not a button: it holds CopyableId,
 * which is a button itself, and buttons can't nest. Mouse clicks anywhere on
 * the row open the epic; the status and title sit in a real button so the row
 * stays keyboard-reachable.
 */
export default function EpicRow({ epic, issue, onOpen }: { epic: EpicProgress; issue?: Issue; onOpen: () => void }) {
  return (
    <div onClick={onOpen} className="flex items-center gap-3 w-full px-5 py-2 cursor-pointer transition-colors hover:bg-[var(--color-hover-surface)]">
      <button type="button" onClick={(e) => { e.stopPropagation(); onOpen(); }} className="flex items-center gap-3 min-w-0 text-left">
        <StatusIcon status={issue?.status || "PLANNED"} size={14} isInferred={issue?.is_inferred} />
        <span className="text-sm text-[var(--color-text-primary)] truncate">{epic.title}</span>
      </button>
      <CopyableId id={epic.issue_id} className="text-xs shrink-0 tabular-nums" />
      {epic.stale && <span className="text-xs uppercase tracking-wider font-medium px-2 py-1 rounded border shrink-0 text-[var(--color-warning)] border-[var(--color-warning)]/40 bg-[var(--color-warning)]/10">STALE</span>}
      <span className="ml-auto flex items-center gap-2 text-xs text-[var(--color-text-muted)] shrink-0">
        <SubProgress done={epic.children_done} total={epic.children_total} />
        {epic.children_done}/{epic.children_total}
      </span>
    </div>
  );
}
