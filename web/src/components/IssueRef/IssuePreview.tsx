import { useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { BookOpen, FileText, GitCommitHorizontal, ListTree, MessageSquare, OctagonMinus, Paperclip } from "lucide-react";
import type { Issue } from "../../api/client";
import { useIssueList } from "../../api/queries";
import { cn } from "../../utils/cn";
import { Avatar, Popover, StatusIcon } from "../ui";
import type { Placement } from "../ui/popover-utils";
import { IssueMarkdown } from "./IssueMarkdown";
import { previewCounts, type PreviewCounts } from "./issue-preview-utils";

/** Lines of the description the card shows. */
const EXCERPT_LINES = 6;

/** Hover card for an issue, beside its anchor. */
export function IssuePreview({
  anchorRef,
  issue,
  placement = "right-start",
}: {
  anchorRef: RefObject<HTMLElement | null>;
  issue: Issue;
  placement?: Placement;
}) {
  return (
    <Popover anchorRef={anchorRef} onClose={() => {}} placement={placement} offset={8} className="pointer-events-none">
      <IssuePreviewCard issue={issue} />
    </Popover>
  );
}

const card = "w-80 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] p-3";

/**
 * The app's one issue hover card, wherever an issue is hovered: status icon,
 * ID and assignee avatar, the title (up to two lines), the description
 * excerpt, and a footer of icons with counts, each only when there is
 * something to count. Everything comes from the issue list: the card fetches
 * nothing.
 */
export function IssuePreviewCard({ issue }: { issue: Issue }) {
  const items = footerItems(previewCounts(issue, useIssueList()));
  return (
    <div role="tooltip" className={card}>
      <div className="flex items-center gap-1.5 text-xs">
        <StatusIcon status={issue.status} size={12} isInferred={issue.is_inferred} />
        <span className="font-mono text-[var(--color-text-muted)]">{issue.id}</span>
        {issue.assignee && <span className="ml-auto"><Avatar name={issue.assignee} size="sm" /></span>}
      </div>
      <div className="mt-1.5 text-sm text-[var(--color-text-primary)] line-clamp-2">{issue.title}</div>
      {issue.description.trim() && <div className="mt-2"><Excerpt markdown={issue.description} /></div>}
      {items.length > 0 && (
        <div data-preview="footer" className="mt-2 flex items-center justify-between gap-3 text-xs text-[var(--color-text-muted)]">
          {items}
        </div>
      )}
    </div>
  );
}

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;

/** One footer item per thing the issue has, in a fixed order. */
function footerItems(c: PreviewCounts): ReactNode[] {
  const items: ReactNode[] = [];
  if (c.comments) items.push(<Count key="comments" icon={MessageSquare} label={plural(c.comments, "comment")}>{c.comments}</Count>);
  if (c.attachments) items.push(<Count key="attachments" icon={Paperclip} label={plural(c.attachments, "attachment")}>{c.attachments}</Count>);
  if (c.spec) items.push(<Count key="spec" icon={FileText} label="Spec" />);
  if (c.walkthrough) items.push(<Count key="walkthrough" icon={BookOpen} label="Walkthrough" />);
  if (c.children) {
    items.push(
      <Count key="children" icon={ListTree} label={`${c.children.done} of ${c.children.total} sub-issues done`}>
        {c.children.done}/{c.children.total}
      </Count>,
    );
  }
  if (c.commits) items.push(<Count key="commits" icon={GitCommitHorizontal} label={plural(c.commits, "commit")}>{c.commits}</Count>);
  if (c.blockers) {
    items.push(
      <Count key="blockers" icon={OctagonMinus} iconClassName="text-[var(--color-error)]" label={`Blocked by ${plural(c.blockers, "issue")}`}>
        {c.blockers}
      </Count>,
    );
  }
  return items;
}

function Count({ icon: Icon, iconClassName, label, children }: {
  icon: typeof MessageSquare;
  iconClassName?: string;
  label: string;
  children?: ReactNode;
}) {
  return (
    <span title={label} aria-label={label} className="flex items-center gap-1 tabular-nums">
      <Icon size={12} className={iconClassName} aria-hidden />
      {children}
    </span>
  );
}

/**
 * The description as Markdown, cut at EXCERPT_LINES lines. It fades out at
 * the bottom only when it is cut off. The card is not interactive, so it
 * doesn't scroll: a click on the row opens the full issue.
 */
function Excerpt({ markdown }: { markdown: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const [cut, setCut] = useState(false);
  useLayoutEffect(() => {
    const el = ref.current;
    if (el) setCut(el.scrollHeight > el.clientHeight);
  }, [markdown]);
  return (
    <div
      ref={ref}
      data-preview="excerpt"
      className={cn("prose-exponential prose-preview overflow-hidden", cut && "fade-bottom")}
      style={{ maxHeight: `calc(${EXCERPT_LINES} * var(--text-xs--line-height))` }}
    >
      <IssueMarkdown>{markdown}</IssueMarkdown>
    </div>
  );
}
