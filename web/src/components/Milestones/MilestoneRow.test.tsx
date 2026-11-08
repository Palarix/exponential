import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { renderMarkup } from "../../render-markup";
import { queryKeys } from "../../api/query-keys";
import type { Milestone } from "../../api/types";
import { MilestoneRowBody, MilestoneSummaryCard } from "./MilestoneRow";

const noop = () => {};
const current: Milestone = {
  id: "ms-2",
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
  points: { total: 13, done: 5 },
};

const render = (el: ReactElement, client?: QueryClient) =>
  renderMarkup(<TooltipPrimitive.Provider>{el}</TooltipPrimitive.Provider>, client);

function row(milestone: Milestone, extra: Partial<Parameters<typeof MilestoneRowBody>[0]> = {}) {
  return render(
    <MilestoneRowBody
      milestone={milestone}
      today="2026-10-20"
      handle={<span aria-label="Drag to reorder" />}
      expanded={false}
      onToggle={noop}
      onOpen={noop}
      {...extra}
    />,
  );
}

describe("MilestoneRowBody", () => {
  it("shows only the name and the progress while collapsed, with the goal as the name's tooltip", () => {
    const html = row(current);
    expect(html).not.toContain("Live courier tracking");
    expect(html).toMatch(/<button[^>]*data-state="closed"[^>]*>1\.1<\/button>/);
    const at = (s: string) => html.indexOf(s);
    expect(at(">1.1<")).toBeLessThan(at("issues done"));
    expect(at("issues done")).toBeLessThan(at("Nov 15"));
    expect(at("Nov 15")).toBeLessThan(at(">Current<"));
  });

  it("shows the goal on a second line under the name while expanded", () => {
    const html = row(current, { expanded: true });
    expect(html).toMatch(/<p[^>]*text-xs[^>]*>Live courier tracking<\/p>/);
    expect(html.indexOf("issues done")).toBeLessThan(html.indexOf("Live courier tracking"));
    expect(html.indexOf("Live courier tracking")).toBeLessThan(html.indexOf("Nov 15"));
    expect(html).not.toMatch(/<button[^>]*data-state="closed"[^>]*>1\.1<\/button>/);
  });

  it("gives a milestone without a goal no tooltip and no second line", () => {
    const noGoal = { ...current, goal: undefined };
    expect(row(noGoal)).not.toMatch(/<button[^>]*data-state="closed"[^>]*>1\.1<\/button>/);
    expect(row(noGoal, { expanded: true })).not.toMatch(/<p[^>]*text-xs/);
  });

  it("truncates a long name with an ellipsis", () => {
    expect(row(current)).toMatch(/<button[^>]*class="[^"]*\btruncate\b[^"]*"[^>]*>1\.1<\/button>/);
  });

  it("shows the number as the name until the milestone has a label", () => {
    const html = row({ ...current, label: undefined, name: "#2" });
    expect(html).toContain(">#2<");
    expect(html).not.toContain("Unnamed");
  });

  it("puts the target date and the done/total progress on the title line", () => {
    const html = row(current);
    expect(html).toContain("Nov 15");
    expect(html).toMatch(/aria-label="3 of 6 issues done\. Click for story points"/);
    expect(html).toMatch(/aria-expanded="false"[^>]*aria-label="Show issues"/);
  });

  it("shows story points done of total in points mode", () => {
    const html = row(current, { progressMode: "points" });
    expect(html).toMatch(/aria-label="5 of 13 story points done\. Click for issue count"/);
    expect(html).toContain("5/13");
    expect(html).not.toContain("3/6");
  });

  it("shows the ring in issues mode and the triangle instead in points mode", () => {
    const pill = (html: string) => {
      const start = html.indexOf(" done. Click for");
      return html.slice(start, html.indexOf("</button>", start));
    };
    expect(pill(row(current))).toContain("<circle");
    const points = pill(row(current, { progressMode: "points" }));
    expect(points).toContain("lucide-triangle");
    expect(points).not.toContain("<circle");
    expect(pill(row({ ...current, points: { total: 13, done: 13 } }, { progressMode: "points" }))).not.toContain("circle-check");
  });

  it("raises the progress pill, a button, off the row", () => {
    const html = row(current);
    expect(html).toMatch(/<button[^>]*aria-label="3 of 6 issues done[^"]*" class="[^"]*bg-\[var\(--color-surface-2\)\]/);
  });

  it("hides the pill in points mode for a past milestone without stored points", () => {
    const old = { ...current, state: "COMPLETED" as const, slot: "past" as const, completed_at: "2026-10-02T10:00:00Z", points: undefined };
    expect(row(old, { handle: null, progressMode: "points" })).not.toMatch(/story points done/);
    expect(row(old, { handle: null })).toMatch(/aria-label="3 of 6 issues done/);
  });

  it("marks a target date that passed with open scope", () => {
    expect(row({ ...current, target_date: "2026-10-10" })).toMatch(/text-\[var\(--color-warning\)\][^>]*>Oct 10</);
  });

  it("shows the order warning with the conflicting milestones", () => {
    expect(row(current, { warning: ["1.2"] })).toContain('aria-label="Target date conflicts with 1.2"');
  });

  it("shows a past milestone faded, with its completion date, and lets it expand", () => {
    const html = row({ ...current, state: "COMPLETED", slot: "past", completed_at: "2026-10-02T10:00:00Z" }, { handle: null });
    expect(html).toContain("opacity-60");
    expect(html).toContain("Oct 2");
    expect(html).toContain("Show issues");
    expect(html).not.toMatch(/>Current</);
  });

  it("shows an expanded past milestone at full opacity", () => {
    const done = { ...current, state: "COMPLETED" as const, slot: "past" as const, completed_at: "2026-10-02T10:00:00Z" };
    expect(row(done, { handle: null, expanded: true })).not.toContain("opacity-60");
  });

  it("makes a past milestone's name a toggle for its issues, not a link", () => {
    const html = row({ ...current, state: "COMPLETED", slot: "past", completed_at: "2026-10-02T10:00:00Z" }, { handle: null, expanded: true });
    expect(html).toMatch(/<button[^>]*aria-expanded="true"[^>]*>1\.1<\/button>/);
  });

  it("makes only an open milestone's name a link, not the goal or the badge", () => {
    const html = row(current, { expanded: true });
    expect(html).toMatch(/<button[^>]*title="Show in Issues"[^>]*>1\.1<\/button>/);
    expect(html).not.toMatch(/<button[^>]*>[^<]*Live courier tracking/);
    // Collapsed, the goal tooltip takes the place of the title.
    expect(row(current)).not.toContain('title="Show in Issues"');
    expect(row({ ...current, goal: undefined })).toContain('title="Show in Issues"');
  });

  it("puts the progress right after the name", () => {
    const html = row(current);
    const progress = html.indexOf('aria-label="3 of 6 issues done. Click for story points"');
    expect(html.indexOf(">1.1<")).toBeLessThan(progress);
    expect(progress).toBeLessThan(html.indexOf("Nov 15"));
  });

  it("puts the slot badge at the far right, just before the actions", () => {
    const html = row(current, { actions: <span aria-label="Milestone actions" /> });
    expect(html.indexOf("Nov 15")).toBeLessThan(html.indexOf(">Current<"));
    expect(html.indexOf(">Current<")).toBeLessThan(html.indexOf('aria-label="Milestone actions"'));
  });

  it("keeps the chevron rightmost, with the actions just left of it", () => {
    const html = row(current, { actions: <span aria-label="Milestone actions" /> });
    expect(html.indexOf("Nov 15")).toBeLessThan(html.indexOf('aria-label="Milestone actions"'));
    expect(html.indexOf('aria-label="Milestone actions"')).toBeLessThan(html.indexOf('aria-label="Show issues"'));
    const afterChevron = html.slice(html.indexOf('aria-label="Show issues"'));
    expect(afterChevron.slice(afterChevron.indexOf("</button>"))).toMatch(/^<\/button><\/div>/);
  });
});

describe("MilestoneSummaryCard", () => {
  it("breaks the scope down by status, with the target date and the burndown", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.milestoneProgress("#2"), {
      days: [
        { date: "2026-10-03", scope: 5, open: 5, completed: 0 },
        { date: "2026-10-04", scope: 6, open: 3, completed: 3 },
      ],
    });
    const html = render(<MilestoneSummaryCard milestone={current} today="2026-10-20" />, client);
    expect(html).toContain("In progress");
    expect(html).toContain(">3/6 done<");
    expect(html).toMatch(/aria-label="Time to target date"/);
    expect(html).toMatch(/aria-label="Burndown"/);
  });

  it("draws the burndown on the first day, as flat lines for open and scope", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.milestoneProgress("#2"), {
      days: [{ date: "2026-10-03", scope: 6, open: 4, completed: 2 }],
    });
    const html = render(<MilestoneSummaryCard milestone={current} today="2026-10-03" />, client);
    expect(html).toMatch(/aria-label="Burndown"/);
    expect(html).not.toContain("Not enough days");
    const lines = [...html.matchAll(/<path d="M0,([\d.]+) L200,([\d.]+)"/g)];
    expect(lines).toHaveLength(2);
    for (const [, from, to] of lines) expect(from).toBe(to);
    expect(lines[0][1]).not.toBe(lines[1][1]);
  });

  it("has no burndown for a milestone that was never Current", () => {
    const html = render(<MilestoneSummaryCard milestone={{ ...current, slot: "next", started_at: undefined }} today="2026-10-20" />);
    expect(html).not.toMatch(/aria-label="Burndown"/);
    expect(html).toContain("Starts when this milestone becomes Current.");
  });

  it("shows a past milestone's final stats; its burndown is in the expanded row", () => {
    const completed = { ...current, state: "COMPLETED" as const, slot: "past" as const, completed_at: "2026-10-09T10:00:00Z" };
    const html = render(<MilestoneSummaryCard milestone={completed} today="2026-10-20" />);
    expect(html).toContain("In progress");
    expect(html).not.toContain("Burndown");
  });

  it("shows only the issue count in issues mode", () => {
    const html = render(<MilestoneSummaryCard milestone={current} today="2026-10-20" mode="issues" />);
    expect(html).toContain(">3/6 done<");
    expect(html).not.toContain("pts");
  });

  it("shows only the story points in points mode", () => {
    const html = render(<MilestoneSummaryCard milestone={current} today="2026-10-20" mode="points" />);
    expect(html).toContain(">5/13 pts<");
    expect(html).not.toContain("3/6 done");
  });
});
