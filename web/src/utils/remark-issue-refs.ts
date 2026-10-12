import type { Element as HastElement, Text as HastText } from "hast";
import type { InlineCode, Link, PhrasingContent, Root, Text } from "mdast";
import { SKIP, visit } from "unist-util-visit";
import { issueHref, splitIssueIds, type IssueIdPart } from "./issue-refs";

/**
 * Links issue IDs while Markdown is parsed, so IDs in code blocks and in
 * existing links are left alone. IDs inside inline code become links inside
 * the one code span; code that is only an ID becomes a plain ID link.
 */
export function remarkIssueRefs({ prefix }: { prefix: string }) {
  return (tree: Root) => {
    visit(tree, (node, index, parent) => {
      if (node.type === "link" || node.type === "linkReference" || node.type === "definition") return SKIP;
      if (!parent || index === undefined) return;
      if (node.type === "text") {
        const parts = splitIssueIds(node.value, prefix);
        if (parts.length === 1 && typeof parts[0] === "string") return;
        const nodes: PhrasingContent[] = parts.map((p) => (typeof p === "string" ? text(p) : idLink(p.id)));
        parent.children.splice(index, 1, ...nodes);
        return [SKIP, index + nodes.length];
      }
      if (node.type === "inlineCode") {
        const parts = splitIssueIds(node.value, prefix);
        // Code that is only an ID renders as the chip alone, not a chip in a code box.
        if (parts.length === 1 && typeof parts[0] !== "string") {
          parent.children.splice(index, 1, idLink(parts[0].id));
          return SKIP;
        }
        linkInlineCode(node, parts);
      }
    });
  };
}

function text(value: string): Text {
  return { type: "text", value };
}

function idLink(id: string): Link {
  return { type: "link", url: issueHref(id), children: [text(id)] };
}

function linkInlineCode(node: InlineCode, parts: IssueIdPart[]) {
  if (parts.length === 1 && typeof parts[0] === "string") return;
  const children = parts.map((p): HastText | HastElement =>
    typeof p === "string"
      ? { type: "text", value: p }
      : { type: "element", tagName: "a", properties: { href: issueHref(p.id) }, children: [{ type: "text", value: p.id }] },
  );
  node.data = { ...node.data, hName: "code", hChildren: children };
}
