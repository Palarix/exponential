import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, fetchCommitDiff, fetchIssueDiff, fetchIssues } from "./client";

describe("API client on 401", () => {
  const assign = vi.fn();

  beforeEach(() => {
    vi.stubGlobal("window", { location: { assign } });
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response('{"error":"unauthorized"}', { status: 401 })),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    assign.mockReset();
  });

  it.each([
    ["request()", () => fetchIssues()],
    ["fetchIssueDiff", () => fetchIssueDiff("xpo-1")],
    ["fetchCommitDiff", () => fetchCommitDiff("xpo-1", "abc")],
  ])("%s redirects to /auth and rejects with ApiError", async (_name, call) => {
    await expect(call()).rejects.toBeInstanceOf(ApiError);
    expect(assign).toHaveBeenCalledWith("/auth");
  });

  it("does not redirect on other errors", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("nope", { status: 500 })));
    await expect(fetchIssues()).rejects.toBeInstanceOf(ApiError);
    expect(assign).not.toHaveBeenCalled();
  });
});
