import { useState } from "react";
import { Trash2 } from "lucide-react";
import type { IssueDraft } from "../../api/types";
import { useIssueDrafts } from "../../api/queries";
import { useDeleteIssueDraft } from "../../api/mutations";
import { useAppNav } from "../../app/hooks";
import { formatRelativeTime } from "../../utils/format";
import { draftDisplayTitle } from "../NewIssueModal/issue-form";
import { Button, ConfirmDialog, EmptyState, Heading, IconButton, LabelBadge, TopBar, ViewContainer } from "../ui";

/** Lists the user's issue drafts, newest first. A click reopens one in the new-issue modal. */
export default function Drafts() {
  const drafts = useIssueDrafts();
  const { openDraft, newIssue } = useAppNav();
  const { mutate: deleteDraft } = useDeleteIssueDraft();
  const [pendingDelete, setPendingDelete] = useState<IssueDraft | null>(null);

  return (
    <ViewContainer
      topBar={<TopBar className="pl-4" left={<Heading title="Drafts" />} />}
      scroll={drafts.length > 0}
    >
      {drafts.length === 0 ? (
        <EmptyState
          title="No drafts"
          description={'Unfinished issues you keep with "Save draft" in the new issue dialog appear here.'}
          actionLabel="New issue"
          onAction={newIssue}
          icon={
            <svg width="160" height="120" viewBox="0 0 160 120" fill="none">
              <path d="M50 15h44l16 16v74a4 4 0 0 1-4 4H50a4 4 0 0 1-4-4V19a4 4 0 0 1 4-4z" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <path d="M94 15v16h16" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <rect x="58" y="46" width="36" height="4" rx="2" fill="var(--color-text-muted)" />
              <rect x="58" y="58" width="44" height="4" rx="2" fill="var(--color-text-muted)" />
              <rect x="58" y="70" width="24" height="4" rx="2" fill="var(--color-text-muted)" />
            </svg>
          }
        />
      ) : (
        drafts.map((d) => (
          <div
            key={d.id}
            role="button"
            tabIndex={0}
            onClick={() => openDraft(d)}
            onKeyDown={(e) => { if (e.key === "Enter") openDraft(d); }}
            className="group flex items-center gap-3 px-4 py-2.5 border-b border-[var(--color-border-subtle)] cursor-pointer transition-colors duration-[var(--duration-fast)] hover:bg-[var(--color-hover-surface)]"
          >
            {d.payload.labels?.[0] && <LabelBadge label={d.payload.labels[0]} />}
            <span className={`flex-1 min-w-0 truncate text-sm ${d.payload.title.trim() ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] italic"}`}>
              {draftDisplayTitle(d.payload)}
            </span>
            <span className="text-xs tabular-nums text-[var(--color-text-muted)] shrink-0">
              {formatRelativeTime(d.updated_at)}
            </span>
            <IconButton
              icon={<Trash2 size={14} />}
              tooltip="Delete draft"
              aria-label="Delete draft"
              className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
              onClick={(e) => { e.stopPropagation(); setPendingDelete(d); }}
            />
          </div>
        ))
      )}
      {pendingDelete && (
        <ConfirmDialog
          scope="drafts.confirm-delete"
          title="Delete this draft?"
          description={`"${draftDisplayTitle(pendingDelete.payload)}" will be deleted. This cannot be undone.`}
          onCancel={() => setPendingDelete(null)}
        >
          <Button variant="secondary" size="sm" onClick={() => setPendingDelete(null)}>Cancel</Button>
          <Button variant="danger" size="sm" onClick={() => { deleteDraft(pendingDelete.id); setPendingDelete(null); }}>Delete</Button>
        </ConfirmDialog>
      )}
    </ViewContainer>
  );
}
