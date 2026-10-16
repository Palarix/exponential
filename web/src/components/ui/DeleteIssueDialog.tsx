import type { Issue } from "../../api/client";
import Button from "./Button";
import ConfirmDialog from "./ConfirmDialog";
import StatusIcon from "./StatusIcon";

interface DeleteIssueDialogProps {
  issue: Pick<Issue, "id" | "title" | "status">;
  /** With sub-issues, offer to keep them as top-level issues or delete them too. */
  subIssueCount: number;
  /** `cascade` deletes the sub-issues too. */
  onDelete: (cascade: boolean) => void;
  onCancel: () => void;
}

/** Confirms deleting an issue, named in a card. Escape and a backdrop click cancel it. */
export default function DeleteIssueDialog({ issue, subIssueCount, onDelete, onCancel }: DeleteIssueDialogProps) {
  const hasChildren = subIssueCount > 0;
  const one = subIssueCount === 1;
  return (
    <ConfirmDialog
      scope="issue.confirm-delete"
      title="Delete issue?"
      description={hasChildren
        ? `It has ${subIssueCount} ${one ? "sub-issue" : "sub-issues"}. Keep ${one ? "it as a top-level issue" : "them as top-level issues"}, or delete ${one ? "it" : "them"} too?`
        : "This can't be undone."}
      body={
        <div className="flex items-center gap-2.5 min-w-0 px-3 py-2 text-sm rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-2)]">
          <StatusIcon status={issue.status} size={14} />
          <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">{issue.id}</span>
          <span className="text-[var(--color-text-primary)] truncate">{issue.title}</span>
        </div>
      }
      onCancel={onCancel}
    >
      <Button variant="secondary" size="sm" onClick={onCancel}>Cancel</Button>
      {hasChildren && <Button variant="secondary" tone="danger" size="sm" onClick={() => onDelete(false)}>Keep sub-issues</Button>}
      <Button variant="danger" size="sm" onClick={() => onDelete(hasChildren)}>{hasChildren ? "Delete all" : "Delete"}</Button>
    </ConfirmDialog>
  );
}
