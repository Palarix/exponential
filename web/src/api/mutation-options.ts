import type { MutationOptions, QueryClient } from "@tanstack/react-query";
import {
  addConfigLabel, addDraft, createIssue, createIssueDraft, deleteConfigLabel, deleteIssueDraft, mergeIssue, startWork,
  updateConfigLabel, updateIssueDraft,
} from "./client";
import { queryKeys } from "./query-keys";
import { applyIssuePatches, removeLabel, renameLabel, withConfigLabels, type IssuePatch, type RawConfig } from "./query-utils";
import type { Issue, IssueDraft, IssueDraftPayload } from "./types";

export type { IssuePatch };

/** The client calls mutations make; injectable for tests. */
export interface MutationApi {
  addDraft: typeof addDraft;
  createIssue: typeof createIssue;
  startWork: typeof startWork;
  mergeIssue: typeof mergeIssue;
  addConfigLabel: typeof addConfigLabel;
  updateConfigLabel: typeof updateConfigLabel;
  deleteConfigLabel: typeof deleteConfigLabel;
  createIssueDraft: typeof createIssueDraft;
  updateIssueDraft: typeof updateIssueDraft;
  deleteIssueDraft: typeof deleteIssueDraft;
}

const clientApi: MutationApi = {
  addDraft, createIssue, startWork, mergeIssue, addConfigLabel, updateConfigLabel, deleteConfigLabel,
  createIssueDraft, updateIssueDraft, deleteIssueDraft,
};

/** Shared by every mutation that changes issues, so a burst of them refetches once. */
export const ISSUES_MUTATION_KEY = ["issues"] as const;
const CONFIG_MUTATION_KEY = ["config"] as const;

/**
 * Refetches issues once the last in-flight issue mutation settles. TanStack
 * runs `onSettled` while the mutation still counts as pending, so the last one sees 1.
 */
function invalidateIssuesWhenIdle(qc: QueryClient): Promise<void> | undefined {
  if (qc.isMutating({ mutationKey: ISSUES_MUTATION_KEY }) > 1) return;
  return qc.invalidateQueries({ queryKey: queryKeys.issues });
}

function issueMutation<TData, TVars>(
  qc: QueryClient,
  mutationFn: (vars: TVars) => Promise<TData>,
): MutationOptions<TData, Error, TVars> {
  return { mutationKey: ISSUES_MUTATION_KEY, mutationFn, onSettled: () => invalidateIssuesWhenIdle(qc) };
}

export interface UpdateIssuesContext {
  undo: IssuePatch[];
}

/** Optimistically patches the cached issues; on failure reverts only the patched fields. */
export function updateIssuesOptions(
  qc: QueryClient,
  api: MutationApi = clientApi,
): MutationOptions<void, Error, IssuePatch[], UpdateIssuesContext> {
  return {
    mutationKey: ISSUES_MUTATION_KEY,
    mutationFn: async (patches) => {
      for (const { issueId, patch } of patches) await api.addDraft(issueId, "UPDATE", patch);
    },
    onMutate: async (patches) => {
      await qc.cancelQueries({ queryKey: queryKeys.issues });
      const { next, undo } = applyIssuePatches(qc.getQueryData<Issue[]>(queryKeys.issues), patches);
      qc.setQueryData(queryKeys.issues, next);
      return { undo };
    },
    onError: (_err, _patches, context) => {
      if (!context) return;
      qc.setQueryData<Issue[]>(queryKeys.issues, (prev) => applyIssuePatches(prev, context.undo).next);
    },
    onSettled: () => invalidateIssuesWhenIdle(qc),
  };
}

export type CreateIssuePayload = Parameters<typeof createIssue>[0];

interface CreateIssueVars {
  issue: CreateIssuePayload;
  update?: Partial<Issue>;
}

/** Creates an issue, then applies fields the CREATE draft can't carry (status, estimate). */
async function createWithUpdate(api: MutationApi, { issue, update }: CreateIssueVars): Promise<string> {
  const issueId = await api.createIssue(issue);
  if (update && Object.keys(update).length > 0) await api.addDraft(issueId, "UPDATE", update);
  return issueId;
}

export function createIssueOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return issueMutation(qc, (vars: CreateIssueVars) => createWithUpdate(api, vars));
}

/**
 * Creates an issue from an issue draft, then deletes the draft. A failed
 * delete does not fail the mutation: the issue exists, and retrying would
 * duplicate it. The caller reports `draftDeleted: false` instead.
 */
export function createIssueFromDraftOptions(
  qc: QueryClient,
  api: MutationApi = clientApi,
): MutationOptions<{ issueId: string; draftDeleted: boolean }, Error, CreateIssueVars & { draftId: string }> {
  return {
    mutationKey: ISSUES_MUTATION_KEY,
    mutationFn: async ({ draftId, ...vars }) => {
      const issueId = await createWithUpdate(api, vars);
      const draftDeleted = await api.deleteIssueDraft(draftId).then(() => true, () => false);
      return { issueId, draftDeleted };
    },
    onSettled: async () => {
      await invalidateIssuesWhenIdle(qc);
      await qc.invalidateQueries({ queryKey: queryKeys.issueDrafts });
    },
  };
}

/** Creates an issue draft, or replaces the payload of `draftId`. */
export function saveIssueDraftOptions(
  qc: QueryClient,
  api: MutationApi = clientApi,
): MutationOptions<IssueDraft, Error, { draftId?: string; payload: IssueDraftPayload }> {
  return {
    mutationFn: ({ draftId, payload }) => (draftId ? api.updateIssueDraft(draftId, payload) : api.createIssueDraft(payload)),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.issueDrafts }),
  };
}

export function deleteIssueDraftOptions(qc: QueryClient, api: MutationApi = clientApi): MutationOptions<void, Error, string> {
  return {
    mutationFn: (draftId) => api.deleteIssueDraft(draftId),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.issueDrafts }),
  };
}

export function deleteIssueOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return issueMutation(qc, ({ issueId, cascade }: { issueId: string; cascade: boolean }) =>
    api.addDraft(issueId, "DELETE", { cascade }));
}

export function addCommentOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return issueMutation(qc, ({ issueId, comment }: { issueId: string; comment: unknown }) =>
    api.addDraft(issueId, "COMMENT", comment));
}

export function startWorkOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return issueMutation(qc, (issueId: string) => api.startWork(issueId));
}

export function mergeIssueOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return issueMutation(qc, ({ issueId, options }: { issueId: string; options: Parameters<typeof mergeIssue>[1] }) =>
    api.mergeIssue(issueId, options));
}

// --- Config labels ---

function labelMutation<TVars>(
  qc: QueryClient,
  mutationFn: (vars: TVars) => Promise<void>,
  nextLabels: (labels: Record<string, string>, vars: TVars) => Record<string, string>,
  touchesIssues: boolean,
): MutationOptions<void, Error, TVars> {
  return {
    mutationKey: CONFIG_MUTATION_KEY,
    mutationFn,
    onSuccess: (_data, vars) => {
      qc.setQueryData<RawConfig>(queryKeys.config, (prev) => withConfigLabels(prev, nextLabels(prev?.labels ?? {}, vars)));
    },
    onSettled: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.config });
      // Renames and deletes rewrite the labels on issues server-side.
      if (touchesIssues) await qc.invalidateQueries({ queryKey: queryKeys.issues });
    },
  };
}

export function addLabelOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return labelMutation<{ name: string; color: string }>(
    qc,
    ({ name, color }) => api.addConfigLabel(name, color),
    (labels, { name, color }) => ({ ...labels, [name]: color }),
    false,
  );
}

export function updateLabelOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return labelMutation<{ oldName: string; newName: string; color: string }>(
    qc,
    ({ oldName, newName, color }) => api.updateConfigLabel(oldName, newName, color),
    (labels, { oldName, newName, color }) => renameLabel(labels, oldName, newName, color),
    true,
  );
}

export function deleteLabelOptions(qc: QueryClient, api: MutationApi = clientApi) {
  return labelMutation<{ name: string }>(
    qc,
    ({ name }) => api.deleteConfigLabel(name),
    (labels, { name }) => removeLabel(labels, name),
    true,
  );
}
