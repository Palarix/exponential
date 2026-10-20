import type { Issue, IssueDraftPayload, Release, ReleasesResponse } from "../../api/types";
import { openReleases } from "../Releases/releases-utils";

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
   * The picked release: a release ID, `@current`/`@next` for a placeholder,
   * or "none". Empty until the user picks, so a sub-issue copies its parent's.
   */
  release: string;
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
  release: "",
};

/** Title or description has text. Other fields alone do not make a form worth keeping. */
export function formHasContent(form: IssueForm): boolean {
  return form.title.trim() !== "" || form.description.trim() !== "";
}

const sameList = (a: string[], b: string[]) => a.length === b.length && a.every((x, i) => x === b[i]);

export function formsEqual(a: IssueForm, b: IssueForm): boolean {
  return a.title === b.title && a.description === b.description && sameList(a.labels, b.labels) &&
    a.status === b.status && a.estimate === b.estimate && a.parentId === b.parentId &&
    a.assignee === b.assignee && sameList(a.additionalLabels, b.additionalLabels) && a.release === b.release;
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
    ...(form.release && { release: form.release }),
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
    release: p.release ?? "",
  };
}

/**
 * The release the form will create the issue in: the picked one, or else the
 * parent's release if it is OPEN (the server copies it). Null for none.
 */
export function shownRelease(form: IssueForm, issues: Issue[], board: ReleasesResponse | undefined): Release | null {
  if (!board || form.release === "none") return null;
  const open = openReleases(board);
  if (form.release === "@current") return board.current;
  if (form.release === "@next") return board.next;
  if (form.release) return open.find((r) => r.id === form.release) ?? null;
  const parent = issues.find((i) => i.id === form.parentId);
  return (parent?.release_id && open.find((r) => r.id === parent.release_id)) || null;
}

export function draftDisplayTitle(p: Pick<IssueDraftPayload, "title">): string {
  return p.title.trim() || "Untitled draft";
}
