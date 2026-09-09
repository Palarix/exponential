import type { Issue } from "./api/types";

export function makeIssue(overrides: Partial<Issue> = {}): Issue {
  return {
    id: "xpo-aaa111",
    title: "Test issue",
    description: "",
    status: "PLANNED",
    estimate: 0,
    priority: 0,
    sort_order: "a0",
    created_at: "2026-01-01T00:00:00Z",
    created_by: "test <test@test.local>",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}
