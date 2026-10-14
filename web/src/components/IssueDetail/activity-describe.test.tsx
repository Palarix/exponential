import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryClient, queryKeys } from "../../api/queries";
import type { Cycle } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { describeEvent } from "./activity-describe";

const parent = makeIssue({ id: "xpo-a1b2c3", title: "Parent epic" });
const cycle: Cycle = {
  id: "2026-10-12", number: 12, start: "2026-10-12", end: "2026-10-25",
  status: "current", done: 0, total: 0,
};

function render(payload: Record<string, unknown>, type = "UPDATE"): { html: string; text: string } {
  const qc = createQueryClient();
  qc.setQueryData(queryKeys.issues, [parent]);
  qc.setQueryData(queryKeys.config, { prefix: "xpo" });
  qc.setQueryData(queryKeys.cycles, { enabled: true, cycles: [cycle] });
  const node = describeEvent({
    type,
    payload,
    created_at: "2026-10-14T00:00:00Z",
    created_by: "Alice <alice@example.com>",
  });
  const html = renderToStaticMarkup(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
  const text = html.replace(/<[^>]+>/g, "").replace(/\s+/g, " ").trim();
  return { html, text };
}

// Bug xpo-ca763f: cleared values and priority/parent/cycle changes were
// missing or rendered as empty/zero values.
describe("describeEvent UPDATE", () => {
  it("describes a cleared estimate", () => {
    expect(render({ estimate: 0 }).text).toBe("cleared the estimate");
  });

  it("keeps a set estimate", () => {
    expect(render({ estimate: 3 }).text).toBe("set estimate to 3 Points");
  });

  it("describes an unassign on its own", () => {
    expect(render({ assignee: "" }).text).toBe("removed the assignee");
  });

  it("keeps an assignment", () => {
    expect(render({ assignee: "Bob <bob@example.com>" }).text).toBe("assigned to Bob");
  });

  it("describes removing all labels", () => {
    expect(render({ labels: [] }).text).toBe("removed all labels");
  });

  it("keeps updated labels", () => {
    expect(render({ labels: ["bug"] }).text).toBe("updated labels to Bug");
  });

  it("describes a cleared priority", () => {
    expect(render({ priority: 0 }).text).toBe("cleared the priority");
  });

  it("describes a set priority with its icon and label", () => {
    const { html, text } = render({ priority: 2 });
    expect(text).toBe("set priority to High");
    expect(html).toContain("<svg");
  });

  it("describes a removed parent", () => {
    expect(render({ parent_id: "" }).text).toBe("removed the parent issue");
  });

  it("describes a set parent as an issue chip", () => {
    const { html, text } = render({ parent_id: "xpo-a1b2c3" });
    expect(text).toBe("set parent to xpo-a1b2c3");
    expect(html).toContain('href="#/issues/xpo-a1b2c3"');
  });

  it("describes removal from a cycle", () => {
    expect(render({ cycle_id: "" }).text).toBe("removed this issue from its cycle");
  });

  it("names a known cycle", () => {
    expect(render({ cycle_id: "2026-10-12" }).text).toBe("added this issue to Cycle 12");
  });

  it("falls back to the cycle ID for an unknown cycle", () => {
    expect(render({ cycle_id: "2027-01-04" }).text).toBe("added this issue to 2027-01-04");
  });

  it("keeps status, title and description texts", () => {
    expect(render({ status: "DOING" }).text).toBe("changed status to In Progress");
    expect(render({ title: "New" }).text).toBe("updated the title");
    expect(render({ description: "x" }).text).toBe("updated the description");
  });

  it("joins several fields with and", () => {
    expect(render({ assignee: "", priority: 0 }).text).toBe("removed the assignee and cleared the priority");
  });
});

describe("describeEvent ARTIFACT", () => {
  // Bug xpo-ca763f: system rows stay muted; values highlight on hover.
  it("shows the filename muted with a hover highlight", () => {
    const { html, text } = render({ action: "updated", filename: "spec.md" }, "ARTIFACT");
    expect(text).toBe("updated spec.md");
    expect(html.replace(/hover:text-\[var\(--color-text-primary\)\]/g, "")).not.toContain("text-[var(--color-text-primary)]");
    expect(html).toContain("hover:text-[var(--color-text-primary)]");
  });
});
