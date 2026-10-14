import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { useConfig } from "../../api/queries";
import { rehypeTaskListSpacing } from "../../utils/rehype-task-list-spacing";
import { remarkIssueRefs } from "../../utils/remark-issue-refs";
import { MarkdownLink } from "./IssueRef";

const COMPONENTS = { a: MarkdownLink };
const REHYPE_PLUGINS = [rehypeTaskListSpacing];

/** Markdown with the project's issue IDs rendered as IssueRef chips. */
export function IssueMarkdown({ children }: { children: string }) {
  const { prefix } = useConfig();
  return (
    <Markdown
      remarkPlugins={[remarkGfm, remarkBreaks, [remarkIssueRefs, { prefix }]]}
      rehypePlugins={REHYPE_PLUGINS}
      components={COMPONENTS}
    >
      {children}
    </Markdown>
  );
}
