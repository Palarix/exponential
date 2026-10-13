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

describe("ApiError message", () => {
  const res = (status: number, statusText = "") => new Response(null, { status, statusText });

  it("uses the error field of a JSON error body, keeping the raw body", () => {
    const body = '{"error":"no branch found for this issue"}';
    const err = new ApiError(res(404), body);
    expect(err.message).toBe("no branch found for this issue");
    expect(err.body).toBe(body);
  });

  it("keeps the body for non-JSON bodies and JSON without a string error", () => {
    expect(new ApiError(res(500), "nope").message).toBe("nope");
    expect(new ApiError(res(500), '{"error":42}').message).toBe('{"error":42}');
    expect(new ApiError(res(500), '["x"]').message).toBe('["x"]');
  });

  it("falls back to the status for an empty body", () => {
    expect(new ApiError(res(502, "Bad Gateway"), "").message).toBe("502 Bad Gateway");
  });
});
