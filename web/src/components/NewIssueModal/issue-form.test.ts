import { describe, expect, it } from "vitest";
import {
  EMPTY_ISSUE_FORM, draftDisplayTitle, formFromDraftPayload, formHasContent, formsEqual, draftPayloadFromForm,
  type IssueForm,
} from "./issue-form";

const filled: IssueForm = {
  title: "Title",
  description: "Body",
  labels: ["bug"],
  status: "PLANNED",
  estimate: "3",
  parentId: "xpo-abc123",
  assignee: "Bob <bob@example.com>",
  additionalLabels: ["backend"],
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
