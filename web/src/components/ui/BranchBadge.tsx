import type { BranchStats } from "../../api/types";
import { GitBranch } from "lucide-react";

interface BranchBadgeProps {
  stats: BranchStats;
}

export default function BranchBadge({ stats }: BranchBadgeProps) {
  const hasCommits = stats.commits > 0;
  const hasUncommitted = !hasCommits && stats.has_uncommitted;
  const title = hasCommits
    ? `${stats.branch} — ${stats.commits} commit${stats.commits === 1 ? "" : "s"}, ${stats.files_changed} file${stats.files_changed === 1 ? "" : "s"}, +${stats.insertions} -${stats.deletions}`
    : hasUncommitted
    ? `${stats.branch} — uncommitted changes: ${stats.files_changed} file${stats.files_changed === 1 ? "" : "s"}, +${stats.insertions} -${stats.deletions}`
    : `${stats.branch} — no commits yet`;

  return (
    <span className="relative inline-flex shrink-0">
      <span
        className="inline-flex items-center gap-1 h-6 px-2 rounded-md border border-[var(--color-border-label)] text-xs text-[var(--color-text-secondary)] tabular-nums"
        title={title}
      >
        <GitBranch size={12} strokeWidth={1.5} />
        {stats.commits}
      </span>
      {hasUncommitted && (
        <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-blue-500" title="uncommitted changes" />
      )}
    </span>
  );
}
