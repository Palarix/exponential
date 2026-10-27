import { describe, expect, it } from "vitest";
import type { Milestone, MilestonesResponse } from "../../api/types";
import { makeIssue } from "../../test-utils";
import {
  childrenPrompt, filterPickerOptions, isInMilestone, openDescendants, pickerOptions, planMilestoneMoves, milestoneChipTooltip, reopenHint,
} from "./milestone-assign-utils";

const rel = (number: number, overrides: Partial<Milestone> = {}): Milestone => ({
  id: `ms-${number}`, number, name: overrides.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...overrides,
});
const placeholder = (number: number, slot: "current" | "next", locked = false): Milestone =>
  rel(number, { id: undefined, placeholder: true, slot, locked });
const board = (overrides: Partial<MilestonesResponse> = {}): MilestonesResponse => ({
  current: rel(1, { slot: "current", label: "1.1" }),
  next: rel(2, { slot: "next", label: "1.2" }),
  upcoming: [rel(3, { slot: "upcoming", label: "Team Mode" })],
  past: [rel(9, { state: "COMPLETED", slot: "past", label: "1.0" })],
  ...overrides,
});

const epic = makeIssue({ id: "xpo-ep1", title: "Epic" });
const kid = makeIssue({ id: "xpo-k1", parent_id: "xpo-ep1" });
const grandkid = makeIssue({ id: "xpo-g1", parent_id: "xpo-k1" });
const doneKid = makeIssue({ id: "xpo-k2", parent_id: "xpo-ep1", status: "DONE" });
const kidInTwo = makeIssue({ id: "xpo-k3", parent_id: "xpo-ep1", milestone_id: "ms-2" });
const all = [epic, kid, grandkid, doneKid, kidInTwo];

describe("pickerOptions", () => {
  it("lists Current and Next, then Upcoming, never past milestones", () => {
    const opts = pickerOptions(board());
    expect(opts.map((o) => o.milestone.number)).toEqual([1, 2, 3]);
    expect(opts.map((o) => o.slotLabel)).toEqual(["Current", "Next", undefined]);
  });

  it("lists the issue's completed milestone first, disabled, so the picker shows where it is", () => {
    const opts = pickerOptions(board(), "ms-9");
    expect(opts.map((o) => o.milestone.number)).toEqual([9, 1, 2, 3]);
    expect(opts[0]).toMatchObject({ disabled: true, completed: true });
  });

  it("adds no completed milestone for an issue in an open one", () => {
    expect(pickerOptions(board(), "ms-1").map((o) => o.milestone.number)).toEqual([1, 2, 3]);
  });

  it("keeps placeholders and disables a locked Next", () => {
    const opts = pickerOptions(board({ current: placeholder(1, "current"), next: placeholder(2, "next", true), upcoming: [] }));
    expect(opts.map((o) => [o.milestone.number, o.disabled])).toEqual([[1, false], [2, true]]);
  });
});

describe("filterPickerOptions", () => {
  const opts = pickerOptions(board());
  it("matches the name, ignoring case", () => {
    expect(filterPickerOptions(opts, "team").map((o) => o.milestone.number)).toEqual([3]);
  });
  it("matches the number as #N", () => {
    expect(filterPickerOptions(opts, "#2").map((o) => o.milestone.number)).toEqual([2]);
  });
  it("matches the goal", () => {
    const withGoal = pickerOptions(board({ upcoming: [rel(3, { label: "Team Mode", goal: "Shared boards" })] }));
    expect(filterPickerOptions(withGoal, "shared").map((o) => o.milestone.number)).toEqual([3]);
  });
  it("matches the slot", () => {
    expect(filterPickerOptions(opts, "next").map((o) => o.milestone.number)).toEqual([2]);
  });
  it("returns all for an empty query", () => {
    expect(filterPickerOptions(opts, "  ")).toHaveLength(3);
  });
});

describe("openDescendants", () => {
  it("returns open descendants at any depth, never terminal ones", () => {
    expect(openDescendants(all, "xpo-ep1").map((i) => i.id).sort()).toEqual(["xpo-g1", "xpo-k1", "xpo-k3"]);
  });
  it("survives a parent cycle", () => {
    const a = makeIssue({ id: "a", parent_id: "b" });
    const b = makeIssue({ id: "b", parent_id: "a" });
    expect(openDescendants([a, b], "a").map((i) => i.id)).toEqual(["b"]);
  });
});

describe("isInMilestone", () => {
  it("compares milestone IDs; a placeholder holds nothing; null is no milestone", () => {
    expect(isInMilestone(kidInTwo, rel(2))).toBe(true);
    expect(isInMilestone(kid, rel(2))).toBe(false);
    expect(isInMilestone(kid, placeholder(2, "next"))).toBe(false);
    expect(isInMilestone(kid, null)).toBe(true);
    expect(isInMilestone(kidInTwo, null)).toBe(false);
  });
});

describe("childrenPrompt", () => {
  it("lists the open descendants not already in the target and counts the done ones", () => {
    const p = childrenPrompt(all, ["xpo-ep1"], rel(2));
    expect(p?.children.map((c) => c.id).sort()).toEqual(["xpo-g1", "xpo-k1"]);
    expect(p?.doneCount).toBe(1);
    expect(p?.epics.map((e) => e.id)).toEqual(["xpo-ep1"]);
  });

  it("is null without open children to move", () => {
    expect(childrenPrompt(all, ["xpo-g1"], rel(2))).toBeNull();
    const onlyDone = [epic, doneKid];
    expect(childrenPrompt(onlyDone, ["xpo-ep1"], rel(2))).toBeNull();
  });

  it("asks once for an epic whose children are assigned with it", () => {
    const p = childrenPrompt(all, ["xpo-k1", "xpo-ep1"], rel(2));
    expect(p?.epics.map((e) => e.id)).toEqual(["xpo-ep1"]);
    expect(p?.children.map((c) => c.id).sort()).toEqual(["xpo-g1", "xpo-k1"]);
  });

  it("asks about every child for No milestone when they have a milestone", () => {
    const p = childrenPrompt(all, ["xpo-ep1"], null);
    expect(p?.children.map((c) => c.id)).toEqual(["xpo-k3"]);
  });
});

describe("planMilestoneMoves", () => {
  it("sends the milestone ID, and patches optimistically", () => {
    const { moves, optimistic } = planMilestoneMoves(all, ["xpo-k1"], rel(3, { label: "Team Mode" }), false);
    expect(moves).toEqual([{ issueId: "xpo-k1", ref: "ms-3", withChildren: false }]);
    expect(optimistic).toEqual([{ issueId: "xpo-k1", patch: { milestone_id: "ms-3", milestone: "Team Mode" } }]);
  });

  it("sends a slot ref for a placeholder and patches only the name", () => {
    const { moves, optimistic } = planMilestoneMoves(all, ["xpo-k1"], placeholder(4, "next"), false);
    expect(moves[0].ref).toBe("@next");
    expect(optimistic[0].patch).toEqual({ milestone: "#4" });
  });

  it("sends an empty ref for No milestone", () => {
    const { moves, optimistic } = planMilestoneMoves(all, ["xpo-k3"], null, false);
    expect(moves).toEqual([{ issueId: "xpo-k3", ref: "", withChildren: false }]);
    expect(optimistic[0].patch).toEqual({ milestone_id: "", milestone: "" });
  });

  it("with children: one move for the epic, optimistic patches for its open descendants", () => {
    const { moves, optimistic } = planMilestoneMoves(all, ["xpo-ep1", "xpo-k1"], rel(2, { label: "1.2" }), true);
    expect(moves).toEqual([{ issueId: "xpo-ep1", ref: "ms-2", withChildren: true }]);
    expect(optimistic.map((p) => p.issueId).sort()).toEqual(["xpo-ep1", "xpo-g1", "xpo-k1"]);
  });

  it("with children: a done issue assigned with its epic still moves on its own", () => {
    const { moves } = planMilestoneMoves(all, ["xpo-ep1", "xpo-k2"], rel(2), true);
    expect(moves).toEqual([
      { issueId: "xpo-ep1", ref: "ms-2", withChildren: true },
      { issueId: "xpo-k2", ref: "ms-2", withChildren: false },
    ]);
  });

  it("without children: each assigned issue moves on its own", () => {
    const { moves } = planMilestoneMoves(all, ["xpo-ep1", "xpo-k1"], rel(2), false);
    expect(moves.map((m) => m.issueId)).toEqual(["xpo-ep1", "xpo-k1"]);
  });

  it("skips issues already in the target", () => {
    expect(planMilestoneMoves(all, ["xpo-k3"], rel(2), false).moves).toEqual([]);
  });
});

describe("reopenHint", () => {
  it("returns the COMPLETED milestone of an open issue", () => {
    expect(reopenHint(makeIssue({ milestone_id: "ms-9" }), board())?.name).toBe("1.0");
  });
  it("is null for done issues, open milestones and no milestone", () => {
    expect(reopenHint(makeIssue({ milestone_id: "ms-9", status: "DONE" }), board())).toBeNull();
    expect(reopenHint(makeIssue({ milestone_id: "ms-1" }), board())).toBeNull();
    expect(reopenHint(makeIssue(), board())).toBeNull();
    expect(reopenHint(makeIssue({ milestone_id: "ms-9" }), undefined)).toBeNull();
  });
});

describe("milestoneChipTooltip", () => {
  it("names the milestone and marks a completed one", () => {
    expect(milestoneChipTooltip("1.2")).toBe("Milestone 1.2");
    expect(milestoneChipTooltip("1.0", rel(9, { state: "COMPLETED" }))).toBe("Milestone 1.0 · completed");
  });
});
