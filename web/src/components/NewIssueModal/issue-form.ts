import type { Issue, IssueDraftPayload, Milestone, MilestonesResponse } from "../../api/types";
import { openMilestones } from "../Milestones/milestones-utils";

/** The editable state of the new-issue modal. */
export interface IssueForm {
  title: string;
  description: string;
  /** The single primary label. */
  labels: string[];
  status: string;
  /** Estimate as the dropdown value, "0" for none. */
  estimate: string;
  parentId: string;
  assignee: string;
  additionalLabels: string[];
  /**
   * The picked milestone: a milestone ID, `@current`/`@next` for a placeholder,
   * or "none". Empty until the user picks, so a sub-issue copies its parent's.
   */
  milestone: string;
}

export const EMPTY_ISSUE_FORM: IssueForm = {
  title: "",
  description: "",
  labels: ["feature"],
  status: "BACKLOG",
  estimate: "0",
  parentId: "",
  assignee: "",
  additionalLabels: [],
  milestone: "",
};

/** Title or description has text. Other fields alone do not make a form worth keeping. */
export function formHasContent(form: IssueForm): boolean {
  return form.title.trim() !== "" || form.description.trim() !== "";
}

const sameList = (a: string[], b: string[]) => a.length === b.length && a.every((x, i) => x === b[i]);

export function formsEqual(a: IssueForm, b: IssueForm): boolean {
  return a.title === b.title && a.description === b.description && sameList(a.labels, b.labels) &&
    a.status === b.status && a.estimate === b.estimate && a.parentId === b.parentId &&
    a.assignee === b.assignee && sameList(a.additionalLabels, b.additionalLabels) && a.milestone === b.milestone;
}

export function draftPayloadFromForm(form: IssueForm): IssueDraftPayload {
  const estimate = parseInt(form.estimate, 10);
  return {
    title: form.title,
    ...(form.description && { description: form.description }),
    labels: form.labels,
    ...(form.additionalLabels.length > 0 && { additional_labels: form.additionalLabels }),
    status: form.status,
    ...(estimate > 0 && { estimate }),
    ...(form.parentId && { parent_id: form.parentId }),
    ...(form.assignee && { assignee: form.assignee }),
    ...(form.milestone && { milestone: form.milestone }),
  };
}

export function formFromDraftPayload(p: IssueDraftPayload): IssueForm {
  return {
    title: p.title ?? "",
    description: p.description ?? "",
    labels: p.labels?.length ? p.labels : EMPTY_ISSUE_FORM.labels,
    status: p.status || EMPTY_ISSUE_FORM.status,
    estimate: String(p.estimate ?? 0),
    parentId: p.parent_id ?? "",
    assignee: p.assignee ?? "",
    additionalLabels: p.additional_labels ?? [],
    milestone: p.milestone ?? "",
  };
}

/**
 * The milestone the form will create the issue in: the picked one, or else the
 * parent's milestone if it is OPEN (the server copies it). Null for none.
 */
export function shownMilestone(form: IssueForm, issues: Issue[], board: MilestonesResponse | undefined): Milestone | null {
  if (!board || form.milestone === "none") return null;
  const open = openMilestones(board);
  if (form.milestone === "@current") return board.current;
  if (form.milestone === "@next") return board.next;
  if (form.milestone) return open.find((r) => r.id === form.milestone) ?? null;
  const parent = issues.find((i) => i.id === form.parentId);
  return (parent?.milestone_id && open.find((r) => r.id === parent.milestone_id)) || null;
}

export function draftDisplayTitle(p: Pick<IssueDraftPayload, "title">): string {
  return p.title.trim() || "Untitled draft";
}
