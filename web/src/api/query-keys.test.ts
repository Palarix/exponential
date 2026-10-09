import { describe, expect, it } from "vitest";
import { QueryClient, type QueryKey } from "@tanstack/react-query";
import { invalidateOnServerEvent, queryKeys, SSE_INVALIDATED_KEYS } from "./query-keys";
import { makeIssue } from "../test-utils";

const startsWith = (key: QueryKey, prefix: QueryKey) =>
  prefix.every((part, i) => key[i] === part);

const stats = (head_sha: string) => ({
  branch: "b", head_sha, commits: 1, files_changed: 1, insertions: 1, deletions: 0, has_uncommitted: false,
});

describe("queryKeys", () => {
  it("nests parameterised keys under the prefixes SSE invalidates", () => {
    expect(startsWith(queryKeys.cycleProgress("c1"), queryKeys.cycles)).toBe(true);
    expect(startsWith(queryKeys.timeline(200), queryKeys.timelineAll)).toBe(true);
  });

  it("keys history and artifacts by updated_at", () => {
    const a = makeIssue({ id: "x", updated_at: "t1" });
    const b = { ...a, updated_at: "t2" };
    const c = { ...a, title: "unrelated change" };
    expect(queryKeys.issueHistory(a)).not.toEqual(queryKeys.issueHistory(b));
    expect(queryKeys.issueHistory(a)).toEqual(queryKeys.issueHistory(c));
    expect(queryKeys.artifact(a, "spec.md")).not.toEqual(queryKeys.artifact(b, "spec.md"));
    expect(queryKeys.artifact(a, "spec.md")).not.toEqual(queryKeys.artifact(a, "walkthrough.md"));
  });

  it("keys the worktree by status", () => {
    const a = makeIssue({ id: "x", status: "PLANNED" });
    expect(queryKeys.worktree(a)).not.toEqual(queryKeys.worktree({ ...a, status: "DOING" }));
    expect(queryKeys.worktree(a)).toEqual(queryKeys.worktree({ ...a, updated_at: "later" }));
  });

  it("keys git data by head_sha", () => {
    const a = makeIssue({ id: "x", branch_stats: stats("aaa") });
    const b = { ...a, branch_stats: stats("bbb") };
    const c = { ...a, updated_at: "later" };
    for (const key of [queryKeys.issueCommits, queryKeys.mergeability, (i: typeof a) => queryKeys.issueDiff(i)]) {
      expect(key(a)).not.toEqual(key(b));
      expect(key(a)).toEqual(key(c));
    }
    expect(queryKeys.issueDiff(a)).not.toEqual(queryKeys.issueDiff(a, "uncommitted"));
  });

  it("tolerates issues without branch stats", () => {
    const a = makeIssue({ id: "x" });
    expect(queryKeys.issueCommits(a)).toEqual(["issue", "x", "commits", null]);
  });
});

describe("invalidateOnServerEvent", () => {
  const seeded = () => {
    const qc = new QueryClient();
    const keys: QueryKey[] = [
      queryKeys.issues, queryKeys.inbox, queryKeys.inboxStatus, queryKeys.cycles, queryKeys.cycleProgress("c1"),
      queryKeys.metrics, queryKeys.activity, queryKeys.timeline(100),
      queryKeys.user, queryKeys.config, queryKeys.instances,
      queryKeys.issueHistory(makeIssue({ id: "x" })),
    ];
    for (const k of keys) qc.setQueryData(k, "data");
    return qc;
  };
  const invalidated = (qc: QueryClient, key: QueryKey) => qc.getQueryState(key)?.isInvalidated;

  it("invalidates everything server events can change", async () => {
    const qc = seeded();
    await invalidateOnServerEvent(qc);
    for (const k of [queryKeys.issues, queryKeys.inboxStatus, queryKeys.cycleProgress("c1"), queryKeys.metrics, queryKeys.activity, queryKeys.timeline(100)]) {
      expect(invalidated(qc, k), JSON.stringify(k)).toBe(true);
    }
  });

  it("leaves user, config, polled instances and version-keyed issue data alone", async () => {
    const qc = seeded();
    await invalidateOnServerEvent(qc);
    for (const k of [queryKeys.user, queryKeys.config, queryKeys.instances, queryKeys.issueHistory(makeIssue({ id: "x" }))]) {
      expect(invalidated(qc, k), JSON.stringify(k)).toBe(false);
    }
  });

  it("lists each prefix once", () => {
    expect(new Set(SSE_INVALIDATED_KEYS.map((k) => JSON.stringify(k))).size).toBe(SSE_INVALIDATED_KEYS.length);
  });
});
