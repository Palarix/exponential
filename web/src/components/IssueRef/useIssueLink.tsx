import { useEffect, useRef, useState } from "react";
import { useIssues } from "../../api/queries";
import { issueHref } from "../../utils/issue-refs";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { IssuePreview } from "./IssuePreview";

const PREVIEW_DELAY = 250;

/**
 * A link to an issue that previews it on hover or focus. Spread `linkProps`
 * on the <a> and render `preview` next to it. `issue` is undefined until the
 * issue list loads, or when the ID is not in it.
 */
export function useIssueLink(id: string) {
  const { data: issues } = useIssues();
  const issue = issues?.find((i) => i.id === id);
  const ref = useRef<HTMLAnchorElement>(null);
  const [preview, setPreview] = useState(false);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);

  const linkProps = {
    ref,
    href: issueHref(id),
    onMouseEnter: () => issue && intent.enter(),
    onMouseLeave: () => intent.leave(),
    onFocus: () => issue && intent.enter(),
    onBlur: () => intent.leave(),
    onClick: () => intent.leave(),
  };
  return {
    issues,
    issue,
    linkProps,
    preview: preview && issue && <IssuePreview anchorRef={ref} issue={issue} placement="bottom-start" />,
  };
}
