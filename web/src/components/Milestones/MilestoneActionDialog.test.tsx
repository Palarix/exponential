import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import type { Milestone } from "../../api/types";
import { MovePicker } from "./MilestoneActionDialog";
import { moveHelp, milestoneDialogCopy, milestoneIntro, milestoneMenuItems } from "./milestone-action-utils";

const milestone: Milestone = { id: "ms-2", number: 2, name: "1.5", label: "1.5", state: "OPEN", slot: "current", counts: {}, total: 0, done: 0 };

describe("milestoneDialogCopy", () => {
  it("asks to complete a milestone, with a picker for the open issues", () => {
    expect(milestoneDialogCopy("complete", 3)).toEqual({ title: "Complete milestone?", pickerLabel: "Move open issues to", confirm: "Complete milestone" });
  });

  it("asks to delete a milestone, with a picker for all issues", () => {
    expect(milestoneDialogCopy("delete", 4)).toEqual({ title: "Delete milestone?", pickerLabel: "Move issues to", confirm: "Delete milestone" });
  });

  it("has no picker when nothing would move", () => {
    expect(milestoneDialogCopy("complete", 0).pickerLabel).toBeUndefined();
    expect(milestoneDialogCopy("delete", 0).pickerLabel).toBeUndefined();
  });
});

describe("milestoneIntro", () => {
  const withGoal = { ...milestone, name: "1.1", label: "1.1", goal: "Refinement" };
  const text = (...args: Parameters<typeof milestoneIntro>) =>
    milestoneIntro(...args).map((p) => (typeof p === "string" ? p : `**${p.strong}**`)).join("");

  it("names the milestone with its goal and asks to decide about its issues", () => {
    expect(text("delete", withGoal, 3)).toBe(
      "Do you really want to delete **1.1 – Refinement**? Decide what happens to the 3 issues currently assigned to this milestone.",
    );
    expect(text("complete", withGoal, 2)).toBe(
      "Do you really want to complete **1.1 – Refinement**? Decide what happens to the 2 open issues currently assigned to this milestone.",
    );
  });

  it("speaks of a single issue in the singular", () => {
    expect(text("delete", milestone, 1)).toBe("Do you really want to delete **1.5**? Decide what happens to the issue currently assigned to this milestone.");
    expect(text("complete", milestone, 1)).toBe("Do you really want to complete **1.5**? Decide what happens to the open issue currently assigned to this milestone.");
  });

  it("only asks when nothing would move", () => {
    expect(text("delete", milestone, 0)).toBe("Do you really want to delete **1.5**?");
  });
});

describe("moveHelp", () => {
  const current = { value: "ms-1", label: "1.0 (Current)" };
  const none = { value: "none", label: "No milestone" };
  const fresh = { value: "new", label: "A new milestone" };
  /** The help as text, with the bold target in **. */
  const text = (...args: Parameters<typeof moveHelp>) =>
    moveHelp(...args).map((p) => (typeof p === "string" ? p : `**${p.strong}**`)).join("");

  it("says a delete moves every issue, done ones too, to the picked milestone, for good", () => {
    expect(text("delete", milestone, 3, current)).toBe("All 3 issues, including done ones, will move to **1.0 (Current)**. This can't be undone.");
    expect(text("delete", milestone, 1, current)).toBe("The issue will move to **1.0 (Current)**, even if it's done. This can't be undone.");
  });

  it("says a delete to No milestone takes the issues out of the milestone", () => {
    expect(text("delete", milestone, 3, none)).toBe("All 3 issues, including done ones, will be taken out of the milestone. This can't be undone.");
  });

  it("names a new milestone as the target", () => {
    expect(text("delete", milestone, 2, fresh)).toBe("All 2 issues, including done ones, will move to **a new milestone**. This can't be undone.");
  });

  it("says only that a delete of an empty milestone can't be undone", () => {
    expect(text("delete", milestone, 0)).toBe("This can't be undone.");
  });

  it("says a complete moves the open issues, keeps the done ones, and is final", () => {
    expect(text("complete", milestone, 3, current)).toBe(
      "3 open issues will move to **1.0 (Current)**. Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.",
    );
    expect(text("complete", milestone, 1, none)).toBe(
      "1 open issue will be taken out of the milestone. Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.",
    );
  });

  it("says only that completing is final when no open issues move", () => {
    expect(text("complete", milestone, 0)).toBe("Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.");
  });
});

describe("MovePicker", () => {
  const html = renderMarkup(
    <MovePicker label="Move issues to" options={[{ value: "ms-3", label: "1.6 (Next)" }, { value: "none", label: "No milestone" }]} value="ms-3" onChange={() => {}} />,
  );

  it("labels the picker and shows the chosen milestone", () => {
    expect(html).toContain("Move issues to");
    expect(html).toContain("1.6 (Next)");
    expect(html).not.toContain("No milestone");
  });

  it("spans the dialog's width, under its label", () => {
    expect(html).toMatch(/<button[^>]*class="[^"]*w-full[^"]*"[^>]*>/);
  });
});

describe("milestoneMenuItems", () => {
  it("puts Complete milestone first on Current", () => {
    expect(milestoneMenuItems(milestone)).toEqual(["complete", "edit", "delete"]);
  });

  it("puts Edit first on the other open milestones", () => {
    expect(milestoneMenuItems({ ...milestone, slot: "next" })).toEqual(["edit", "complete", "delete"]);
    expect(milestoneMenuItems({ ...milestone, slot: "upcoming" })).toEqual(["edit", "complete", "delete"]);
  });
});
