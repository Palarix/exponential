import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { rehypeTaskListSpacing } from "./rehype-task-list-spacing";

function render(md: string): string {
  return renderToStaticMarkup(
    <Markdown remarkPlugins={[remarkGfm, remarkBreaks]} rehypePlugins={[rehypeTaskListSpacing]}>
      {md}
    </Markdown>,
  );
}

describe("rehypeTaskListSpacing", () => {
  it("drops the space after the checkbox in a tight task list", () => {
    expect(render("- [ ] first\n- [x] second")).toContain(
      `<li class="task-list-item"><input type="checkbox" disabled=""/>first</li>`,
    );
    expect(render("- [ ] first\n- [x] second")).toContain(`<input type="checkbox" disabled="" checked=""/>second`);
  });

  it("drops the space after the checkbox in a loose task list", () => {
    expect(render("- [ ] first\n\n- [ ] second")).toContain(`<p><input type="checkbox" disabled=""/>first</p>`);
  });

  it("keeps spaces inside the item text", () => {
    expect(render("- [ ] two words")).toContain(`<input type="checkbox" disabled=""/>two words`);
  });

  it("leaves non-task lists alone", () => {
    expect(render("- plain item")).toBe("<ul>\n<li>plain item</li>\n</ul>");
  });
});
