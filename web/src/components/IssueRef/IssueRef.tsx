import type { ComponentProps, ReactNode } from "react";
import { useConfig } from "../../api/queries";
import { issueIdFromHref, splitIssueIds } from "../../utils/issue-refs";
import StatusIcon from "../ui/StatusIcon";
import { useIssueLink } from "./useIssueLink";

/**
 * An issue ID as a chip that links to the issue and previews it on hover.
 * Once the issue list has loaded, an ID that is not in it renders as muted
 * text instead of a link to a missing page. With a custom `label` (a Markdown
 * link such as `[the epic](#/issues/…)`) the chip shows that text instead.
 */
export function IssueRef({ id, label }: { id: string; label?: ReactNode }) {
  const { issues, issue, linkProps, preview } = useIssueLink(id);

  if (issues && !issue && label === undefined) {
    return <span className="issue-ref issue-ref-missing" title="Issue not found">{id}</span>;
  }
  return (
    <>
      <a {...linkProps} className={label === undefined ? "issue-ref" : "issue-ref issue-ref-label"} data-issue-ref>
        {issue && <StatusIcon status={issue.status} size={11} />}
        <span className="issue-ref-text">{label ?? id}</span>
      </a>
      {preview}
    </>
  );
}

/** react-markdown `a` component: issue links become chips, other links stay plain. */
export function MarkdownLink({ href, title, children }: ComponentProps<"a">) {
  const id = issueIdFromHref(href);
  if (id) return <IssueRef id={id} label={children === id ? undefined : children} />;
  return <a href={href} title={title}>{children}</a>;
}

/** Plain text (no Markdown) with each issue ID rendered as a chip. */
export function IssueRefText({ text }: { text: string }) {
  const { prefix } = useConfig();
  return splitIssueIds(text, prefix).map((part, i) =>
    typeof part === "string" ? part : <IssueRef key={i} id={part.id} />,
  );
}
