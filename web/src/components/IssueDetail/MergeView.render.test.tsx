import { describe, expect, it, onTestFinished, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { ApiError, type MergedChanges } from "../../api/client";
import { queryKeys } from "../../api/query-keys";
import MergeView from "./MergeView";

const stats = { branch: "xpo-aaa111-x", head_sha: "abc", commits: 1, files_changed: 1, insertions: 1, deletions: 0, has_uncommitted: false };

const changes: MergedChanges = {
  merge_sha: "0123456789abcdef0123456789abcdef01234567",
  base_sha: "fedcba9",
  branch: "xpo-aaa111-x",
  strategy: "squash",
  merged_at: "2026-01-02T00:00:00Z",
  merged_by: "Alice <alice@x.io>",
  commits: [{ sha: "0123456789ab", message: "xpo-aaa111: Test issue", author: "Alice", date: "2026-01-02T00:00:00Z" }],
  diff: ["diff --git a/src/app.ts b/src/app.ts", "--- a/src/app.ts", "+++ b/src/app.ts", "@@ -1 +1 @@", "-a", "+b"].join("\n"),
};

function render(issue = makeIssue(), client = new QueryClient()): string {
  return renderMarkup(<MergeView issue={issue} onClose={() => {}} onMerged={() => {}} />, client);
}

describe("MergeView modes", () => {
  it("shows the no-branch screen, not an error, for an unmerged issue without a branch", () => {
    const html = render(makeIssue({ status: "DOING" }));
    expect(html).toContain("No branch found for this issue");
    expect(html).toContain("Back to issue");
    expect(html).not.toContain("Loading...");
  });

  it("shows the branch view's loading state while the branch is being read", () => {
    expect(render(makeIssue({ status: "DOING", branch_stats: stats }))).toContain("Loading...");
  });

  it("shows the Merged view from the merge commit for a merged DONE issue", () => {
    // The diff viewer reads its unified/split preference.
    vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {} });
    onTestFinished(() => {
      vi.unstubAllGlobals();
    });
    const issue = makeIssue({ status: "DONE", merged: true });
    const client = new QueryClient();
    client.setQueryData(queryKeys.mergedChanges(issue), changes);
    const html = render(issue, client);
    expect(html).toContain("Merged");
    expect(html).toContain("0123456");
    expect(html).toContain("app.ts");
    expect(html).toContain("xpo-aaa111-x");
    // Read-only: no merge controls, no uncommitted scope.
    expect(html).not.toContain("Squash and merge");
    expect(html).not.toContain("Uncommitted");
  });

  it("falls back to a merged screen with the reason when the merge commit can't be found", () => {
    const issue = makeIssue({ status: "DONE", merged: true });
    // Otherwise a cached error renders as a retry in progress.
    const client = new QueryClient({ defaultOptions: { queries: { retryOnMount: false } } });
    const query = client.getQueryCache().build(client, { queryKey: queryKeys.mergedChanges(issue) });
    const err = new ApiError(new Response(null, { status: 404 }), '{"error":"merge commit for xpo-aaa111 not found on main"}');
    query.setState({ status: "error", error: err, errorUpdatedAt: 1, fetchStatus: "idle" });
    const html = render(issue, client);
    expect(html).toContain("This issue has been merged");
    expect(html).toContain("merge commit for xpo-aaa111 not found on main");
    expect(html).toContain("Back to issue");
    expect(html).not.toContain("{&quot;error&quot;");
  });
});
