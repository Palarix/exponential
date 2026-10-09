import type { QueryClient } from "@tanstack/react-query";
import type { Issue } from "./types";

type IssueRef = Pick<Issue, "id" | "updated_at" | "status" | "branch_stats">;

const headSha = (issue: IssueRef) => issue.branch_stats?.head_sha ?? null;

/**
 * Per-issue keys carry the issue field their data depends on (`updated_at`,
 * `status`, `head_sha`), so they refetch exactly when the issues query brings
 * a change — not on every SSE event. Superseded versions age out via gcTime.
 */
export const queryKeys = {
  issues: ["issues"],
  config: ["config"],
  inbox: ["inbox"],
  inboxStatus: ["inbox", "status"],
  user: ["user"],
  cycles: ["cycles"],
  cycleProgress: (cycleId: string) => ["cycles", cycleId, "progress"],
  metrics: ["metrics"],
  activity: ["activity"],
  timelineAll: ["timeline"],
  timeline: (limit: number) => ["timeline", limit],
  instances: ["instances"],
  commitDetail: (sha: string) => ["commit", sha],
  issueHistory: (issue: IssueRef) => ["issue", issue.id, "history", issue.updated_at],
  artifact: (issue: IssueRef, filename: string) => ["issue", issue.id, "artifact", filename, issue.updated_at],
  worktree: (issue: IssueRef) => ["issue", issue.id, "worktree", issue.status],
  issueCommits: (issue: IssueRef) => ["issue", issue.id, "commits", headSha(issue)],
  issueDiff: (issue: IssueRef, scope?: "uncommitted") => ["issue", issue.id, "diff", scope ?? "branch", headSha(issue)],
  mergeability: (issue: IssueRef) => ["issue", issue.id, "mergeability", headSha(issue)],
  commitDiff: (issueId: string, sha: string) => ["issue", issueId, "commit", sha],
} as const;

/** Prefixes of everything an SSE event can change. */
export const SSE_INVALIDATED_KEYS = [
  queryKeys.issues,
  queryKeys.inbox,
  queryKeys.cycles,
  queryKeys.metrics,
  queryKeys.activity,
  queryKeys.timelineAll,
] as const;

export function invalidateOnServerEvent(qc: QueryClient): Promise<void> {
  return Promise.all(SSE_INVALIDATED_KEYS.map((queryKey) => qc.invalidateQueries({ queryKey }))).then(() => {});
}
