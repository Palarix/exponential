import type { BranchStats } from "../../api/types";
import { GitBranch } from "lucide-react";
import { indicatorPill } from "./indicator-pill";

interface BranchBadgeProps {
  stats: BranchStats;
}

/**
 * A row indicator that the issue has a branch: an icon-only pill. The
 * commits, files and line changes are in its tooltip; a dot marks
 * uncommitted changes.
 */
export default function BranchBadge({ stats }: BranchBadgeProps) {
  const hasCommits = stats.commits > 0;
  const hasUncommitted = !hasCommits && stats.has_uncommitted;
  const title = hasCommits
    ? `${stats.branch} — ${stats.commits} commit${stats.commits === 1 ? "" : "s"}, ${stats.files_changed} file${stats.files_changed === 1 ? "" : "s"}, +${stats.insertions} -${stats.deletions}`
    : hasUncommitted
    ? `${stats.branch} — uncommitted changes: ${stats.files_changed} file${stats.files_changed === 1 ? "" : "s"}, +${stats.insertions} -${stats.deletions}`
    : `${stats.branch} — no commits yet`;

  return (
    <span className={`relative ${indicatorPill}`} title={title}>
      <GitBranch size={12} strokeWidth={1.75} />
      {stats.commits}
      {hasUncommitted && (
        <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-blue-500" title="uncommitted changes" />
      )}
    </span>
  );
}
