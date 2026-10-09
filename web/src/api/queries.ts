import { useCallback, useMemo } from "react";
import { QueryClient, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchConfig, fetchInbox, fetchInboxStatus, fetchIssues, fetchUser, markInboxRead } from "./client";
import type { Issue, InboxStatus } from "./types";
import { deriveAppConfig, patchIssueList, withConfigLabels, type RawConfig } from "./query-utils";
import { useToast } from "../components/ui";

export const queryKeys = {
  issues: ["issues"],
  config: ["config"],
  inbox: ["inbox"],
  inboxStatus: ["inbox", "status"],
  user: ["user"],
} as const;

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
  return useCallback(() => {
    qc.invalidateQueries({ queryKey: queryKeys.issues });
    qc.invalidateQueries({ queryKey: queryKeys.inbox });
  }, [qc]);
}
