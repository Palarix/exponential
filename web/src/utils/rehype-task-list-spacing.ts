import type { Root } from "hast";
import { visit } from "unist-util-visit";

/**
 * Drops the space text node that GFM puts after a task-list checkbox. The
 * stylesheet hangs the checkbox in the item's left gutter, and that leftover
 * space would push the first line right of the wrapped lines.
 */
export function rehypeTaskListSpacing() {
  return (tree: Root) => {
    visit(tree, "element", (node, index, parent) => {
      if (node.tagName !== "input" || node.properties.type !== "checkbox") return;
      if (!parent || index === undefined) return;
      const next = parent.children[index + 1];
      if (next?.type === "text" && next.value.trim() === "") parent.children.splice(index + 1, 1);
    });
  };
}
