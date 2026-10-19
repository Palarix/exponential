import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { renderMarkup } from "../../render-markup";
import { queryKeys } from "../../api/query-keys";
import type { Release } from "../../api/types";
import { ReleaseRowBody, ReleaseSummaryCard } from "./ReleaseRow";

const noop = () => {};
const current: Release = {
  id: "rel-2",
  number: 2,
  name: "1.1",
  label: "1.1",
  goal: "Live courier tracking",
  state: "OPEN",
  slot: "current",
  started_at: "2026-10-03T09:00:00Z",
  target_date: "2026-11-15",
  counts: { DOING: 2, PLANNED: 1, DONE: 3 },
  total: 6,
  done: 3,
};

const render = (el: ReactElement, client?: QueryClient) =>
  renderMarkup(<TooltipPrimitive.Provider>{el}</TooltipPrimitive.Provider>, client);

function row(release: Release, extra: Partial<Parameters<typeof ReleaseRowBody>[0]> = {}) {
  return render(
    <ReleaseRowBody
      release={release}
      today="2026-10-20"
      handle={<span aria-label="Drag to reorder" />}
      expanded={false}
      onToggle={noop}
      onOpen={noop}
      {...extra}
    />,
  );
}

describe("ReleaseRowBody", () => {
  it("reads as one title: the label, the goal, then the slot badge", () => {
    const html = row(current);
    expect(html).toMatch(/>1\.1<\/span><span[^>]*> – Live courier tracking<\/span><\/span><span[^>]*>Current</);
  });

  it("shows the number as the name until the release has a label", () => {
    const html = row({ ...current, label: undefined, name: "#2" });
    expect(html).toContain(">#2<");
    expect(html).not.toContain("Unnamed");
  });

  it("puts the target date and the done/total progress on the title line", () => {
    const html = row(current);
    expect(html).toContain("Nov 15");
    expect(html).toMatch(/aria-label="3 of 6 done\. Show breakdown"/);
    expect(html).toMatch(/aria-expanded="false"[^>]*aria-label="Show issues"/);
  });

  it("marks a target date that passed with open scope", () => {
    expect(row({ ...current, target_date: "2026-10-10" })).toMatch(/text-\[var\(--color-warning\)\][^>]*>Oct 10</);
  });

  it("shows the order warning with the conflicting releases", () => {
    expect(row(current, { warning: ["1.2"] })).toContain('aria-label="Target date conflicts with 1.2"');
  });

  it("shows a past release faded, with its release date, and lets it expand", () => {
    const html = row({ ...current, state: "RELEASED", slot: "past", released_at: "2026-10-02T10:00:00Z" }, { handle: null });
    expect(html).toContain("opacity-60");
    expect(html).toContain("Oct 2");
    expect(html).toContain("Show issues");
    expect(html).not.toMatch(/>Current</);
  });
});

describe("ReleaseSummaryCard", () => {
  it("breaks the scope down by status, with the target date and the burndown", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.releaseProgress("#2"), {
      days: [
        { date: "2026-10-03", scope: 5, open: 5, completed: 0 },
        { date: "2026-10-04", scope: 6, open: 3, completed: 3 },
      ],
    });
    const html = render(<ReleaseSummaryCard release={current} today="2026-10-20" />, client);
    expect(html).toContain("In progress");
    expect(html).toContain("3/6 done");
    expect(html).toMatch(/aria-label="Time to target date"/);
    expect(html).toMatch(/aria-label="Burndown"/);
  });

  it("has no burndown for a release that was never Current", () => {
    const html = render(<ReleaseSummaryCard release={{ ...current, slot: "next", started_at: undefined }} today="2026-10-20" />);
    expect(html).not.toMatch(/aria-label="Burndown"/);
    expect(html).toContain("Starts when this release becomes Current.");
  });
});
