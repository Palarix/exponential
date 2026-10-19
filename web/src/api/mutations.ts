import { useCallback } from "react";
import { useMutation, useQueryClient, type MutationOptions } from "@tanstack/react-query";
import {
  addCommentOptions, addLabelOptions, createIssueFromDraftOptions, createIssueOptions, deleteIssueDraftOptions,
  deleteIssueOptions, deleteIssueToast, deleteLabelOptions, mergeIssueOptions, saveIssueDraftOptions, startWorkOptions, updateIssuesOptions,
  updateLabelOptions, type IssuePatch,
} from "./mutation-options";
import { ApiError, createRelease, reorderRelease } from "./client";
import { queryKeys } from "./query-keys";
import type { Issue } from "./types";
import { useToast } from "../components/ui/ToastContext";

export type { IssuePatch };

/** Chains an error toast after the factory's own `onError` (rollback). */
function useWithErrorToast<TData, TVars, TContext>(
  options: MutationOptions<TData, Error, TVars, TContext>,
): MutationOptions<TData, Error, TVars, TContext> {
  const showToast = useToast();
  return {
    ...options,
    onError: (err, vars, context, mutationContext) => {
      options.onError?.(err, vars, context, mutationContext);
      const reason = err instanceof ApiError || err instanceof Error ? err.message : "unknown error";
      showToast(`Couldn't save: ${reason}`, { variant: "error" });
    },
  };
}

/** Optimistic issue updates, applied in order and rolled back on failure. */
export function useUpdateIssues() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(updateIssuesOptions(qc)));
}

/** `useUpdateIssues` for a single issue: `updateIssue(id, patch, { onSuccess })`. */
export function useUpdateIssue() {
  const { mutate } = useUpdateIssues();
  return useCallback(
    (issueId: string, patch: Partial<Issue>, callbacks?: Parameters<typeof mutate>[1]) =>
      mutate([{ issueId, patch }], callbacks),
    [mutate],
  );
}

export function useCreateIssue() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(createIssueOptions(qc)));
}

export function useCreateIssueFromDraft() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(createIssueFromDraftOptions(qc)));
}

/** A 404 means another tab deleted the draft; the update never recreates it. */
export function useSaveIssueDraft() {
  const qc = useQueryClient();
  const showToast = useToast();
  return useMutation({
    ...saveIssueDraftOptions(qc),
    onError: (err) => {
      const message = err instanceof ApiError && err.status === 404
        ? "This draft no longer exists"
        : `Couldn't save draft: ${err.message}`;
      showToast(message, { variant: "error" });
    },
  });
}

export function useDeleteIssueDraft() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(deleteIssueDraftOptions(qc)));
}

export function useDeleteIssue() {
  const qc = useQueryClient();
  const showToast = useToast();
  return useMutation(useWithErrorToast({
    ...deleteIssueOptions(qc),
    onSuccess: (_data, vars) => showToast(deleteIssueToast(vars)),
  }));
}

export function useAddComment() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(addCommentOptions(qc)));
}

export function useStartWork() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(startWorkOptions(qc)));
}

/** No toast: MergeView shows the error inline in its confirm dialog. */
export function useMergeIssue() {
  const qc = useQueryClient();
  return useMutation(mergeIssueOptions(qc));
}

export function useAddLabel() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(addLabelOptions(qc)));
}

export function useUpdateLabel() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(updateLabelOptions(qc)));
}

export function useDeleteLabel() {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast(deleteLabelOptions(qc)));
}

// --- Releases ---

/** Release writes change the board and can save placeholders, so both refetch. */
function useReleaseMutation<TData, TVars>(mutationFn: (vars: TVars) => Promise<TData>) {
  const qc = useQueryClient();
  return useMutation(useWithErrorToast<TData, TVars, unknown>({
    mutationFn,
    onSettled: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.releases });
      await qc.invalidateQueries({ queryKey: queryKeys.issues });
    },
  }));
}

export function useCreateRelease() {
  return useReleaseMutation(createRelease);
}

export function useReorderRelease() {
  return useReleaseMutation(reorderRelease);
}
