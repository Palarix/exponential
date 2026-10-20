import { describe, expect, it } from "vitest";
import {
  EMPTY_ISSUE_FORM, draftDisplayTitle, formFromDraftPayload, formHasContent, formsEqual, draftPayloadFromForm, shownRelease,
  type IssueForm,
} from "./issue-form";
import type { Release, ReleasesResponse } from "../../api/types";
import { makeIssue } from "../../test-utils";

const filled: IssueForm = {
  title: "Title",
  description: "Body",
  labels: ["bug"],
  status: "PLANNED",
  estimate: "3",
  parentId: "xpo-abc123",
  assignee: "Bob <bob@example.com>",
  additionalLabels: ["backend"],
  release: "none",
};

describe("issue form <-> issue draft payload", () => {
  it("round-trips every field", () => {
    expect(formFromDraftPayload(draftPayloadFromForm(filled))).toEqual(filled);
  });

  it("maps form fields to the payload names the server expects", () => {
    expect(draftPayloadFromForm(filled)).toEqual({
      title: "Title",
      description: "Body",
      labels: ["bug"],
      additional_labels: ["backend"],
      status: "PLANNED",
      estimate: 3,
      parent_id: "xpo-abc123",
      assignee: "Bob <bob@example.com>",
      release: "none",
    });
  });

  it("omits empty optional fields", () => {
    expect(draftPayloadFromForm(EMPTY_ISSUE_FORM)).toEqual({ title: "", labels: ["feature"], status: "BACKLOG" });
  });

  it("fills defaults for a sparse payload", () => {
    expect(formFromDraftPayload({ title: "Only a title" })).toEqual({ ...EMPTY_ISSUE_FORM, title: "Only a title" });
  });
});

describe("formHasContent", () => {
  it("is true only when title or description has text", () => {
    expect(formHasContent(EMPTY_ISSUE_FORM)).toBe(false);
    expect(formHasContent({ ...EMPTY_ISSUE_FORM, title: "   " })).toBe(false);
    expect(formHasContent({ ...EMPTY_ISSUE_FORM, status: "DOING" })).toBe(false);
    expect(formHasContent({ ...EMPTY_ISSUE_FORM, title: "x" })).toBe(true);
    expect(formHasContent({ ...EMPTY_ISSUE_FORM, description: "x" })).toBe(true);
  });
});

describe("formsEqual", () => {
  it("compares every field, including label arrays", () => {
    expect(formsEqual(filled, { ...filled })).toBe(true);
    expect(formsEqual(filled, { ...filled, additionalLabels: ["backend", "ui"] })).toBe(false);
    expect(formsEqual(filled, { ...filled, estimate: "5" })).toBe(false);
  });
});

describe("draftDisplayTitle", () => {
  it("falls back to 'Untitled draft'", () => {
    expect(draftDisplayTitle({ title: "  " })).toBe("Untitled draft");
    expect(draftDisplayTitle({ title: " Fix it " })).toBe("Fix it");
  });
});

describe("shownRelease", () => {
  const rel = (number: number, o: Partial<Release> = {}): Release => ({
    id: `rel-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
  });
  const board: ReleasesResponse = {
    current: rel(1, { label: "1.1", slot: "current" }),
    next: rel(2, { id: undefined, placeholder: true, slot: "next" }),
    upcoming: [],
    past: [rel(9, { label: "1.0", state: "RELEASED" })],
  };
  const parentIn = (release_id?: string) => [makeIssue({ id: "xpo-par", release_id })];
  const form = (o: Partial<IssueForm>): IssueForm => ({ ...EMPTY_ISSUE_FORM, parentId: "xpo-par", ...o });

  it("shows the parent's open release until the user picks one: the value the server copies", () => {
    expect(shownRelease(form({}), parentIn("rel-1"), board)?.name).toBe("1.1");
  });

  it("copies nothing from a parent in a released release", () => {
    expect(shownRelease(form({}), parentIn("rel-9"), board)).toBeNull();
  });

  it("shows the picked release, a placeholder by its slot ref", () => {
    expect(shownRelease(form({ release: "rel-1" }), parentIn(), board)?.name).toBe("1.1");
    expect(shownRelease(form({ release: "@next" }), parentIn(), board)?.name).toBe("#2");
  });

  it("shows nothing for an explicit No release, even under a parent in a release", () => {
    expect(shownRelease(form({ release: "none" }), parentIn("rel-1"), board)).toBeNull();
  });

  it("leaves release out of the draft until one is picked", () => {
    expect(draftPayloadFromForm(EMPTY_ISSUE_FORM)).not.toHaveProperty("release");
  });
});
