import { useCallback, useMemo } from "react";
import { keepPreviousData, QueryClient, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchActivity, fetchArtifactContent, fetchCommitDetail, fetchCommitDiff, fetchConfig, fetchCycleProgress, fetchCycles,
  fetchInbox, fetchInboxStatus, fetchInstances, fetchIssueCommits, fetchIssueDiff, fetchIssueHistory, fetchIssues,
  fetchLocalWorktree, fetchMergeability, fetchMetrics, fetchTimeline, fetchUser, markInboxRead,
} from "./client";
import type { Issue, InboxStatus } from "./types";
import { cyclesFromResponse, deriveAppConfig, patchIssueList, withConfigLabels, type RawConfig } from "./query-utils";
import { invalidateOnServerEvent, queryKeys } from "./query-keys";
import { useToast } from "../components/ui/ToastContext";

export { queryKeys };

/** SSE is the freshness signal, so queries never go stale or retry on their own. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false, staleTime: Infinity },
    },
  });
}

export function useIssues() {
  return useQuery({ queryKey: queryKeys.issues, queryFn: fetchIssues });
}

/** The loaded issue list, or `[]` while loading. */
export function useIssueList(): Issue[] {
  const { data } = useIssues();
  return data ?? EMPTY_ISSUES;
}
const EMPTY_ISSUES: Issue[] = [];

/** Refetches issues; resolves once the new list is in the cache. */
export function useRefreshIssues(): () => Promise<void> {
  const qc = useQueryClient();
  return useCallback(() => qc.invalidateQueries({ queryKey: queryKeys.issues }), [qc]);
}

/** Optimistically merges a patch into the cached issue list. */
export function usePatchIssue(): (issueId: string, patch: Partial<Issue>) => void {
  const qc = useQueryClient();
  return useCallback(
    (issueId, patch) => qc.setQueryData<Issue[]>(queryKeys.issues, (prev) => patchIssueList(prev, issueId, patch)),
    [qc],
  );
}

export function useConfig() {
  const { data } = useQuery({ queryKey: queryKeys.config, queryFn: fetchConfig });
  return useMemo(() => deriveAppConfig(data), [data]);
}

/** Replaces the cached config labels after a label edit. */
export function useSetConfigLabels(): (labels: Record<string, string>) => void {
  const qc = useQueryClient();
  return useCallback(
    (labels) => qc.setQueryData<RawConfig>(queryKeys.config, (prev) => withConfigLabels(prev, labels)),
    [qc],
  );
}

export function useInbox() {
  const items = useQuery({ queryKey: queryKeys.inbox, queryFn: fetchInbox });
  const status = useQuery({ queryKey: queryKeys.inboxStatus, queryFn: fetchInboxStatus });
  return {
    items: items.data ?? [],
    lastRead: status.data?.last_read ?? "",
    unread: status.data?.unread ?? 0,
  };
}

export function useMarkInboxRead(): () => Promise<void> {
  const qc = useQueryClient();
  const showToast = useToast();
  return useCallback(async () => {
    try {
      const res = await markInboxRead();
      qc.setQueryData<InboxStatus>(queryKeys.inboxStatus, (prev) => ({ ...prev, last_read: res.last_read, unread: 0 }));
      showToast("Notifications marked as read");
    } catch (err) {
      showToast(err instanceof Error ? err.message : "Failed to mark as read", { variant: "error" });
    }
  }, [qc, showToast]);
}

export function useUser() {
  return useQuery({ queryKey: queryKeys.user, queryFn: fetchUser }).data ?? null;
}

/** Invalidates everything SSE events can change. */
export function useInvalidateOnServerEvent(): () => void {
  const qc = useQueryClient();
  return useCallback(() => { invalidateOnServerEvent(qc); }, [qc]);
}

// --- Per-view data ---

export function useCycles() {
  const q = useQuery({ queryKey: queryKeys.cycles, queryFn: fetchCycles });
  return { cycles: cyclesFromResponse(q.data), isPending: q.isPending };
}

export function useCycleProgress(cycleId: string) {
  return useQuery({ queryKey: queryKeys.cycleProgress(cycleId), queryFn: () => fetchCycleProgress(cycleId) }).data?.days ?? NO_DAYS;
}
const NO_DAYS: never[] = [];

// Time-based metrics (aging, cycle time) change without events, so these keep polling.
const DASHBOARD_POLL = { refetchInterval: 30_000, refetchOnWindowFocus: true } as const;

export function useMetrics() {
  return useQuery({ queryKey: queryKeys.metrics, queryFn: fetchMetrics, ...DASHBOARD_POLL }).data ?? null;
}

export function useActivity() {
  return useQuery({ queryKey: queryKeys.activity, queryFn: fetchActivity, ...DASHBOARD_POLL }).data ?? NO_ACTIVITY;
}
const NO_ACTIVITY: never[] = [];

/** Keeps the previous page visible while a larger `limit` loads. */
export function useTimeline(limit: number) {
  return useQuery({
    queryKey: queryKeys.timeline(limit),
    queryFn: () => fetchTimeline(limit),
    placeholderData: keepPreviousData,
  });
}

export function useCommitDetail(sha: string, enabled: boolean) {
  return useQuery({ queryKey: queryKeys.commitDetail(sha), queryFn: () => fetchCommitDetail(sha), enabled });
}

export function useInstances() {
  return useQuery({
    queryKey: queryKeys.instances,
    queryFn: fetchInstances,
    refetchInterval: 10_000,
    refetchOnWindowFocus: true,
  }).data ?? NO_INSTANCES;
}
const NO_INSTANCES: never[] = [];

// --- Issue detail data (version-keyed, see queryKeys) ---

export function useIssueHistory(issue: Issue) {
  return useQuery({
    queryKey: queryKeys.issueHistory(issue),
    queryFn: () => fetchIssueHistory(issue.id),
    placeholderData: keepPreviousData,
  }).data ?? NO_HISTORY;
}
const NO_HISTORY: never[] = [];

export function artifactQuery(issue: Issue, filename: string) {
  return { queryKey: queryKeys.artifact(issue, filename), queryFn: () => fetchArtifactContent(issue.id, filename) };
}

export function useArtifact(issue: Issue, filename: string, enabled = true) {
  return useQuery({ ...artifactQuery(issue, filename), enabled });
}

/** Fetches an artifact through the cache, e.g. for a download. */
export function useFetchArtifact(): (issue: Issue, filename: string) => Promise<string> {
  const qc = useQueryClient();
  return useCallback((issue, filename) => qc.fetchQuery(artifactQuery(issue, filename)), [qc]);
}

export function useLocalWorktree(issue: Issue) {
  return useQuery({
    queryKey: queryKeys.worktree(issue),
    queryFn: () => fetchLocalWorktree(issue.id),
    placeholderData: keepPreviousData,
  }).data ?? null;
}

export function useIssueCommits(issue: Issue) {
  return useQuery({ queryKey: queryKeys.issueCommits(issue), queryFn: () => fetchIssueCommits(issue.id) });
}

export function useIssueDiff(issue: Issue, scope?: "uncommitted", refetchInterval: number | false = false) {
  return useQuery({ queryKey: queryKeys.issueDiff(issue, scope), queryFn: () => fetchIssueDiff(issue.id, scope), refetchInterval });
}

export function useMergeability(issue: Issue) {
  return useQuery({ queryKey: queryKeys.mergeability(issue), queryFn: () => fetchMergeability(issue.id) });
}

export function useCommitDiff(issueId: string, sha: string | null) {
  return useQuery({
    queryKey: queryKeys.commitDiff(issueId, sha ?? ""),
    queryFn: () => fetchCommitDiff(issueId, sha!),
    enabled: !!sha,
  });
}
