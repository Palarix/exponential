import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import ApprovalCard from "./ApprovalCard";

describe("ApprovalCard", () => {
  const merge = renderMarkup(
    <ApprovalCard chatId="c1" step={{ kind: "tool", id: "t1", name: "mcp__xpo__merge", arg: "xpo-1", done: false, approval: { id: "a1", input: '{"id":"xpo-1"}' } }} />,
  );

  it("says what will happen and names the tool on one line", () => {
    expect(merge).toMatch(/Merge.*xpo-1.*via.*xpo merge/s);
  });

  it("shows nothing the headline already says", () => {
    expect(merge).not.toContain("Details");
    expect(merge).not.toContain("&quot;id&quot;");
  });

  it("offers the three decisions, the main one last", () => {
    const deny = merge.indexOf(">Deny<");
    const session = merge.indexOf(">Allow for this chat<");
    const allow = merge.indexOf(">Allow<");
    expect(deny).toBeGreaterThan(-1);
    expect(session).toBeGreaterThan(deny);
    expect(allow).toBeGreaterThan(session);
  });

  it("shows the command a shell call would run", () => {
    const bash = renderMarkup(
      <ApprovalCard
        chatId="c1"
        step={{ kind: "tool", id: "t2", name: "Bash", arg: "Clean the build", done: false, approval: { id: "a2", input: '{"command":"rm -rf build","description":"Clean the build"}' } }}
      />,
    );
    expect(bash).toContain("Run a shell command");
    expect(bash).toContain("rm -rf build");
  });

  it("lists the other arguments", () => {
    const update = renderMarkup(
      <ApprovalCard chatId="c1" step={{ kind: "tool", id: "t3", name: "mcp__xpo__update", arg: "xpo-1", done: false, approval: { id: "a3", input: '{"id":"xpo-1","status":"DONE"}' } }} />,
    );
    expect(update).toMatch(/status.*DONE/s);
  });
});
