import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import type { Release } from "../../api/types";
import { MovePicker } from "./ReleaseActionDialog";
import { moveHelp, releaseDialogCopy, releaseIntro, releaseMenuItems } from "./release-action-utils";

const release: Release = { id: "rel-2", number: 2, name: "1.5", label: "1.5", state: "OPEN", slot: "current", counts: {}, total: 0, done: 0 };

describe("releaseDialogCopy", () => {
  it("asks to complete a release, with a picker for the open issues", () => {
    expect(releaseDialogCopy("complete", 3)).toEqual({ title: "Complete release?", pickerLabel: "Move open issues to", confirm: "Complete release" });
  });

  it("asks to delete a release, with a picker for all issues", () => {
    expect(releaseDialogCopy("delete", 4)).toEqual({ title: "Delete release?", pickerLabel: "Move issues to", confirm: "Delete release" });
  });

  it("has no picker when nothing would move", () => {
    expect(releaseDialogCopy("complete", 0).pickerLabel).toBeUndefined();
    expect(releaseDialogCopy("delete", 0).pickerLabel).toBeUndefined();
  });
});

describe("releaseIntro", () => {
  const withGoal = { ...release, name: "1.1", label: "1.1", goal: "Refinement" };
  const text = (...args: Parameters<typeof releaseIntro>) =>
    releaseIntro(...args).map((p) => (typeof p === "string" ? p : `**${p.strong}**`)).join("");

  it("names the release with its goal and asks to decide about its issues", () => {
    expect(text("delete", withGoal, 3)).toBe(
      "Do you really want to delete **1.1 – Refinement**? Decide what happens to the 3 issues currently assigned to this release.",
    );
    expect(text("complete", withGoal, 2)).toBe(
      "Do you really want to complete **1.1 – Refinement**? Decide what happens to the 2 open issues currently assigned to this release.",
    );
  });

  it("speaks of a single issue in the singular", () => {
    expect(text("delete", release, 1)).toBe("Do you really want to delete **1.5**? Decide what happens to the issue currently assigned to this release.");
    expect(text("complete", release, 1)).toBe("Do you really want to complete **1.5**? Decide what happens to the open issue currently assigned to this release.");
  });

  it("only asks when nothing would move", () => {
    expect(text("delete", release, 0)).toBe("Do you really want to delete **1.5**?");
  });
});

describe("moveHelp", () => {
  const current = { value: "rel-1", label: "1.0 (Current)" };
  const none = { value: "none", label: "No release" };
  const fresh = { value: "new", label: "A new release" };
  /** The help as text, with the bold target in **. */
  const text = (...args: Parameters<typeof moveHelp>) =>
    moveHelp(...args).map((p) => (typeof p === "string" ? p : `**${p.strong}**`)).join("");

  it("says a delete moves every issue, done ones too, to the picked release, for good", () => {
    expect(text("delete", release, 3, current)).toBe("All 3 issues, including done ones, will move to **1.0 (Current)**. This can't be undone.");
    expect(text("delete", release, 1, current)).toBe("The issue will move to **1.0 (Current)**, even if it's done. This can't be undone.");
  });

  it("says a delete to No release takes the issues out of the release", () => {
    expect(text("delete", release, 3, none)).toBe("All 3 issues, including done ones, will be taken out of the release. This can't be undone.");
  });

  it("names a new release as the target", () => {
    expect(text("delete", release, 2, fresh)).toBe("All 2 issues, including done ones, will move to **a new release**. This can't be undone.");
  });

  it("says only that a delete of an empty release can't be undone", () => {
    expect(text("delete", release, 0)).toBe("This can't be undone.");
  });

  it("says a complete moves the open issues, keeps the done ones, and is final", () => {
    expect(text("complete", release, 3, current)).toBe(
      "3 open issues will move to **1.0 (Current)**. Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.",
    );
    expect(text("complete", release, 1, none)).toBe(
      "1 open issue will be taken out of the release. Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.",
    );
  });

  it("says only that completing is final when no open issues move", () => {
    expect(text("complete", release, 0)).toBe("Done issues stay with 1.5. Completing is final: 1.5 moves to Past and can't be reopened.");
  });
});

describe("MovePicker", () => {
  const html = renderMarkup(
    <MovePicker label="Move issues to" options={[{ value: "rel-3", label: "1.6 (Next)" }, { value: "none", label: "No release" }]} value="rel-3" onChange={() => {}} />,
  );

  it("labels the picker and shows the chosen release", () => {
    expect(html).toContain("Move issues to");
    expect(html).toContain("1.6 (Next)");
    expect(html).not.toContain("No release");
  });

  it("spans the dialog's width, under its label", () => {
    expect(html).toMatch(/<button[^>]*class="[^"]*w-full[^"]*"[^>]*>/);
  });
});

describe("releaseMenuItems", () => {
  it("puts Complete release first on Current", () => {
    expect(releaseMenuItems(release)).toEqual(["complete", "edit", "delete"]);
  });

  it("puts Edit first on the other open releases", () => {
    expect(releaseMenuItems({ ...release, slot: "next" })).toEqual(["edit", "complete", "delete"]);
    expect(releaseMenuItems({ ...release, slot: "upcoming" })).toEqual(["edit", "complete", "delete"]);
  });
});
