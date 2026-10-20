import { useRef, useState } from "react";
import { Link } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { useIssueList } from "../../api/queries";
import { useAppNav } from "../../app/hooks";
import { useIssueLink } from "../IssueRef/useIssueLink";
import { Menu, MenuItem, Popover, StatusIcon } from "../ui";

const iconButton = "shrink-0 p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]";

/**
 * A link icon to the issues a chat has worked on in Work Mode, now or
 * earlier. One issue: a link that previews it on hover. Several: a menu,
 * newest first, with the current Work Mode issue marked. None: nothing.
 */
export default function WorkedIssuesLink({ chat }: { chat: ChatMeta }) {
  const issues = [...(chat.worked_issues ?? [])].reverse();
  if (issues.length === 0) return null;
  if (issues.length === 1) return <IssueLink id={issues[0]} />;
  return <IssuesMenu ids={issues} working={chat.mode === "work" ? chat.work?.issue_id : undefined} />;
}

function IssueLink({ id }: { id: string }) {
  const { linkProps, preview } = useIssueLink(id);
  return (
    <>
      <a {...linkProps} aria-label={`Go to ${id}`} className={iconButton}>
        <Link className="w-4 h-4" />
      </a>
      {preview}
    </>
  );
}

function IssuesMenu({ ids, working }: { ids: string[]; working?: string }) {
  const all = useIssueList();
  const { openIssue } = useAppNav();
  const btnRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  return (
    <>
      <button ref={btnRef} type="button" aria-label="Worked-on issues" onClick={() => setOpen((o) => !o)} className={iconButton}>
        <Link className="w-4 h-4" />
      </button>
      {open && (
        <Popover anchorRef={btnRef} onClose={() => setOpen(false)}>
          <Menu onClose={() => setOpen(false)} aria-label="Worked-on issues" className="surface-ai w-80">
            {ids.map((id) => {
              const issue = all.find((i) => i.id === id);
              return (
                <MenuItem
                  key={id}
                  label={issue ? `${id} ${issue.title}` : id}
                  icon={issue && <StatusIcon status={issue.status} size={14} />}
                  content={
                    <span className="flex items-center gap-2 min-w-0">
                      <span className="shrink-0 font-mono text-xs text-[var(--color-text-muted)]">{id}</span>
                      <span className="truncate">{issue?.title}</span>
                    </span>
                  }
                  suffix={id === working ? <span className="text-xs text-[var(--color-text-muted)]">Working</span> : undefined}
                  onClick={() => {
                    setOpen(false);
                    openIssue(id);
                  }}
                />
              );
            })}
          </Menu>
        </Popover>
      )}
    </>
  );
}
