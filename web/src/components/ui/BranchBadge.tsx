import type { BranchStats } from "../../api/types";
import { GitBranch } from "lucide-react";

interface BranchBadgeProps {
  stats: BranchStats;
}

export default function BranchBadge({ stats }: BranchBadgeProps) {
  const hasCommits = stats.commits > 0;
  const title = hasCommits
    ? `${stats.branch} — ${stats.commits} commit${stats.commits === 1 ? "" : "s"}, ${stats.files_changed} file${stats.files_changed === 1 ? "" : "s"}, +${stats.insertions} -${stats.deletions}`
    : `${stats.branch} — no commits yet`;

  return (
    <span
      className="inline-flex items-center gap-1.5 h-6 px-2 rounded-2xl border border-[var(--color-border-label)] text-xs text-[var(--color-text-secondary)] shrink-0 tabular-nums"
      title={title}
    >
      <GitBranch size={12} strokeWidth={1.5} />
      {stats.head_sha?.slice(0, 6)}
      {hasCommits && (
        <>
          <span className="text-[var(--color-border-label)]">|</span>
          {stats.commits}
        </>
      )}
    </span>
  );
}
