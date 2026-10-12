import { useEffect, useRef, useState, type ComponentProps, type ReactNode } from "react";
import { useConfig, useIssues } from "../../api/queries";
import { issueHref, issueIdFromHref, splitIssueIds } from "../../utils/issue-refs";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { IssuePreview } from "./IssuePreview";

const PREVIEW_DELAY = 250;

/**
 * An issue ID as a chip that links to the issue and previews it on hover.
 * Once the issue list has loaded, an ID that is not in it renders as muted
 * text instead of a link to a missing page. With a custom `label` (a Markdown
 * link such as `[the epic](#/issues/…)`) the chip shows that text instead.
 */
export function IssueRef({ id, label }: { id: string; label?: ReactNode }) {
  const { data: issues } = useIssues();
  const issue = issues?.find((i) => i.id === id);
  const ref = useRef<HTMLAnchorElement>(null);
  const [preview, setPreview] = useState(false);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);

  if (issues && !issue && label === undefined) {
    return <span className="issue-ref issue-ref-missing" title="Issue not found">{id}</span>;
  }
  return (
    <>
      <a
        ref={ref}
        href={issueHref(id)}
        className={label === undefined ? "issue-ref" : "issue-ref issue-ref-label"}
        data-issue-ref
        onMouseEnter={() => issue && intent.enter()}
        onMouseLeave={() => intent.leave()}
        onFocus={() => issue && intent.enter()}
        onBlur={() => intent.leave()}
        onClick={() => intent.leave()}
      >
        {label ?? id}
      </a>
      {preview && issue && <IssuePreview anchorRef={ref} issue={issue} placement="bottom-start" />}
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
