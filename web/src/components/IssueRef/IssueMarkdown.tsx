import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { useConfig } from "../../api/queries";
import { remarkIssueRefs } from "../../utils/remark-issue-refs";
import { MarkdownLink } from "./IssueRef";

const COMPONENTS = { a: MarkdownLink };

/** Markdown with the project's issue IDs rendered as IssueRef chips. */
export function IssueMarkdown({ children }: { children: string }) {
  const { prefix } = useConfig();
  return (
    <Markdown remarkPlugins={[remarkGfm, remarkBreaks, [remarkIssueRefs, { prefix }]]} components={COMPONENTS}>
      {children}
    </Markdown>
  );
}
