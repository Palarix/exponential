import { beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import {
  addCommentOptions, addLabelOptions, createIssueOptions, deleteIssueOptions, deleteLabelOptions, mergeIssueOptions,
  startWorkOptions, updateIssuesOptions, updateLabelOptions, type MutationApi,
} from "./mutation-options";
import { queryKeys } from "./query-keys";
import type { RawConfig } from "./query-utils";
import type { Issue } from "./types";
import { makeIssue } from "../test-utils";

function deferred() {
  let resolve!: () => void;
  let reject!: (err: Error) => void;
  const promise = new Promise<void>((res, rej) => { resolve = res; reject = rej; });
  return { promise, resolve, reject };
}

function fakeApi(): MutationApi {
  return {
    addDraft: vi.fn().mockResolvedValue(undefined),
    createIssue: vi.fn().mockResolvedValue("xpo-new111"),
    startWork: vi.fn().mockResolvedValue({ status: "ok", issue_id: "a", branch: "b", messages: [] }),
    mergeIssue: vi.fn().mockResolvedValue({ status: "ok", merge_sha: "abc", messages: [] }),
    addConfigLabel: vi.fn().mockResolvedValue(undefined),
    updateConfigLabel: vi.fn().mockResolvedValue(undefined),
    deleteConfigLabel: vi.fn().mockResolvedValue(undefined),
  };
}

const rawConfig = { labels: { ui: "#f00" } } as unknown as RawConfig;

let qc: QueryClient;
let api: MutationApi;
let invalidated: unknown[][];

// Runs the same lifecycle as useMutation().mutateAsync, through the mutation cache.
function run<TData, TVars, TContext>(opts: MutationOptions<TData, Error, TVars, TContext>, vars: TVars): Promise<TData> {
  return qc.getMutationCache().build(qc, opts).execute(vars);
}

const issues = () => qc.getQueryData<Issue[]>(queryKeys.issues);

beforeEach(() => {
  qc = new QueryClient();
  api = fakeApi();
  invalidated = [];
  vi.spyOn(qc, "invalidateQueries").mockImplementation(async (filters) => {
    invalidated.push([...(filters?.queryKey ?? [])]);
  });
  qc.setQueryData<Issue[]>(queryKeys.issues, [
    makeIssue({ id: "a", status: "PLANNED", sort_order: "a0", estimate: 0 }),
    makeIssue({ id: "b", status: "PLANNED", sort_order: "a1" }),
  ]);
  qc.setQueryData(queryKeys.config, rawConfig);
});

describe("updateIssuesOptions", () => {
  it("patches the cache before the server answers", async () => {
    const gate = deferred();
    vi.mocked(api.addDraft).mockReturnValue(gate.promise);
    const pending = run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { status: "DOING", sort_order: "a9" } }]);
    await vi.waitFor(() => expect(api.addDraft).toHaveBeenCalled());
    expect(issues()?.[0]).toMatchObject({ status: "DOING", sort_order: "a9" });
    gate.resolve();
    await pending;
    expect(issues()?.[0]).toMatchObject({ status: "DOING", sort_order: "a9" });
  });

  it("sends one UPDATE draft per patch, in order", async () => {
    await run(updateIssuesOptions(qc, api), [
      { issueId: "a", patch: { status: "DONE" } },
      { issueId: "b", patch: { status: "DONE" } },
    ]);
    expect(vi.mocked(api.addDraft).mock.calls).toEqual([
      ["a", "UPDATE", { status: "DONE" }],
      ["b", "UPDATE", { status: "DONE" }],
    ]);
  });

  it("rolls back the patched fields when the write fails", async () => {
    vi.mocked(api.addDraft).mockRejectedValue(new Error("nope"));
    await expect(run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { status: "DOING", sort_order: "a9" } }])).rejects.toThrow("nope");
    expect(issues()?.[0]).toMatchObject({ status: "PLANNED", sort_order: "a0" });
  });

  it("keeps a concurrent edit to another field when rolling back", async () => {
    const gate = deferred();
    vi.mocked(api.addDraft).mockReturnValueOnce(gate.promise);
    const failing = run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { status: "DOING" } }]);
    await vi.waitFor(() => expect(api.addDraft).toHaveBeenCalledTimes(1));
    await run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { estimate: 5 } }]);
    gate.reject(new Error("nope"));
    await expect(failing).rejects.toThrow();
    expect(issues()?.[0]).toMatchObject({ status: "PLANNED", estimate: 5 });
  });

  it("invalidates issues once, after the last concurrent issue mutation settles", async () => {
    const gate = deferred();
    vi.mocked(api.addDraft).mockReturnValueOnce(gate.promise);
    const slow = run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { status: "DOING" } }]);
    await vi.waitFor(() => expect(api.addDraft).toHaveBeenCalledTimes(1));
    await run(updateIssuesOptions(qc, api), [{ issueId: "b", patch: { estimate: 2 } }]);
    expect(invalidated).toEqual([]);
    gate.resolve();
    await slow;
    expect(invalidated).toEqual([[...queryKeys.issues]]);
  });

  it("invalidates after a failure too", async () => {
    vi.mocked(api.addDraft).mockRejectedValue(new Error("nope"));
    await run(updateIssuesOptions(qc, api), [{ issueId: "a", patch: { status: "DOING" } }]).catch(() => {});
    expect(invalidated).toEqual([[...queryKeys.issues]]);
  });
});

describe("createIssueOptions", () => {
  it("creates the issue, applies the follow-up update and resolves to the new id", async () => {
    const id = await run(createIssueOptions(qc, api), { issue: { title: "T", labels: ["feature"] }, update: { status: "PLANNED" } });
    expect(id).toBe("xpo-new111");
    expect(api.createIssue).toHaveBeenCalledWith({ title: "T", labels: ["feature"] });
    expect(api.addDraft).toHaveBeenCalledWith("xpo-new111", "UPDATE", { status: "PLANNED" });
    expect(invalidated).toEqual([[...queryKeys.issues]]);
  });

  it("skips the follow-up update when there is nothing to set", async () => {
    await run(createIssueOptions(qc, api), { issue: { title: "T" }, update: {} });
    await run(createIssueOptions(qc, api), { issue: { title: "T" } });
    expect(api.addDraft).not.toHaveBeenCalled();
  });
});

describe("simple issue mutations", () => {
  it("deleteIssue sends a DELETE draft and invalidates without touching the cache first", async () => {
    const before = issues();
    const gate = deferred();
    vi.mocked(api.addDraft).mockReturnValue(gate.promise);
    const pending = run(deleteIssueOptions(qc, api), { issueId: "a", cascade: true });
    await vi.waitFor(() => expect(api.addDraft).toHaveBeenCalledWith("a", "DELETE", { cascade: true }));
    expect(issues()).toBe(before);
    gate.resolve();
    await pending;
    expect(invalidated).toEqual([[...queryKeys.issues]]);
  });

  it("addComment sends a COMMENT draft", async () => {
    await run(addCommentOptions(qc, api), { issueId: "a", comment: { id: "cmt-1", text: "hi" } });
    expect(api.addDraft).toHaveBeenCalledWith("a", "COMMENT", { id: "cmt-1", text: "hi" });
    expect(invalidated).toEqual([[...queryKeys.issues]]);
  });

  it("startWork and mergeIssue invalidate issues", async () => {
    await run(startWorkOptions(qc, api), "a");
    await run(mergeIssueOptions(qc, api), { issueId: "a", options: { strategy: "squash" } });
    expect(api.startWork).toHaveBeenCalledWith("a");
    expect(api.mergeIssue).toHaveBeenCalledWith("a", { strategy: "squash" });
    expect(invalidated).toEqual([[...queryKeys.issues], [...queryKeys.issues]]);
  });
});

describe("label mutations", () => {
  const labels = () => qc.getQueryData<RawConfig>(queryKeys.config)?.labels;

  it("addLabel stores the new label in the config cache", async () => {
    await run(addLabelOptions(qc, api), { name: "bug", color: "#0f0" });
    expect(api.addConfigLabel).toHaveBeenCalledWith("bug", "#0f0");
    expect(labels()).toEqual({ ui: "#f00", bug: "#0f0" });
    expect(invalidated).toEqual([[...queryKeys.config]]);
  });

  it("updateLabel renames in the cache and refetches issues, which carry the label", async () => {
    await run(updateLabelOptions(qc, api), { oldName: "ui", newName: "frontend", color: "#00f" });
    expect(api.updateConfigLabel).toHaveBeenCalledWith("ui", "frontend", "#00f");
    expect(labels()).toEqual({ frontend: "#00f" });
    expect(invalidated).toEqual([[...queryKeys.config], [...queryKeys.issues]]);
  });

  it("deleteLabel removes it from the cache and refetches issues", async () => {
    await run(deleteLabelOptions(qc, api), { name: "ui" });
    expect(labels()).toEqual({});
    expect(invalidated).toEqual([[...queryKeys.config], [...queryKeys.issues]]);
  });

  it("leaves the cache alone when the server rejects the label", async () => {
    vi.mocked(api.addConfigLabel).mockRejectedValue(new Error("dup"));
    await run(addLabelOptions(qc, api), { name: "bug", color: "#0f0" }).catch(() => {});
    expect(labels()).toEqual({ ui: "#f00" });
  });
});
