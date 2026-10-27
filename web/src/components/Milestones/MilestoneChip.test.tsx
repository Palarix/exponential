import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import { renderMarkup } from "../../render-markup";
import { queryKeys } from "../../api/query-keys";
import type { Milestone } from "../../api/types";
import { makeIssue } from "../../test-utils";
import MilestoneChip from "./MilestoneChip";

const rel = (number: number, o: Partial<Milestone> = {}): Milestone => ({
  id: `ms-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
});

function chip(issue = makeIssue({ milestone_id: "ms-2", milestone: "1.2" }), client = new QueryClient(), expanded = false) {
  return renderMarkup(<TooltipPrimitive.Provider><MilestoneChip issue={issue} expanded={expanded} onOpen={() => {}} /></TooltipPrimitive.Provider>, client);
}

describe("MilestoneChip", () => {
  it("shows the Milestone icon and the milestone name in a picker trigger", () => {
    const html = chip();
    expect(html).toContain("lucide-milestone");
    expect(html).toContain(">1.2<");
    expect(html).toContain('aria-label="Milestone 1.2, change milestone"');
    expect(html).toMatch(/<button[^>]*data-milestone-trigger[^>]*rounded-full|<button[^>]*rounded-full[^>]*data-milestone-trigger/);
    expect(html).toContain('aria-haspopup="menu"');
    expect(html).toContain('aria-expanded="false"');
  });

  it("marks the trigger expanded while its picker is open", () => {
    expect(chip(undefined, undefined, true)).toContain('aria-expanded="true"');
  });

  it("renders nothing without a milestone", () => {
    expect(chip(makeIssue())).toBe("");
  });

  it("marks a completed milestone in its name", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.milestones, {
      current: rel(1), next: rel(2), upcoming: [], past: [rel(9, { label: "1.0", state: "COMPLETED" })],
    });
    const html = chip(makeIssue({ milestone_id: "ms-9", milestone: "1.0" }), client);
    expect(html).toContain('aria-label="Milestone 1.0 · completed, change milestone"');
  });

  it("truncates a long label", () => {
    expect(chip(makeIssue({ milestone_id: "ms-2", milestone: "A very long milestone label indeed" }))).toMatch(/truncate/);
  });
});
