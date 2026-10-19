import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import ApprovalCard, { ButtonLabel } from "./ApprovalCard";

describe("ApprovalCard", () => {
  const mergeStep = { kind: "tool", id: "t1", name: "mcp__xpo__merge", arg: "xpo-1", done: false, approval: { id: "a1", input: '{"id":"xpo-1"}' } } as const;
  const merge = renderMarkup(<ApprovalCard chatId="c1" step={mergeStep} defaultOpen />);
  const collapsed = renderMarkup(<ApprovalCard chatId="c1" step={mergeStep} />);

  it("opens collapsed: no details, the headline and the buttons on one row", () => {
    expect(collapsed).not.toContain("<dl");
    expect(collapsed).toMatch(/flex-wrap[^>]*>.*Merge.*>Deny</s);
  });

  it("puts a chevron in the headline, which toggles the details", () => {
    expect(collapsed).toMatch(/<button[^>]*aria-expanded="false"[^>]*>.*Merge.*lucide-chevron-right/s);
    expect(collapsed).not.toMatch(/lucide-chevron-right[^"]*rotate-90/);
    expect(merge).toMatch(/<button[^>]*aria-expanded="true"[^>]*>.*Merge.*lucide-chevron-right[^"]*rotate-90/s);
  });

  it("says what will happen and names the tool on one line", () => {
    expect(merge).toMatch(/Merge.*xpo-1.*via.*xpo merge/s);
  });

  it("lists the call's arguments in a monospace panel", () => {
    expect(merge).toMatch(/font-mono[^>]*>.*<dt[^>]*>id<\/dt><dd[^>]*>xpo-1<\/dd>/s);
    expect(merge).not.toContain("&quot;id&quot;");
  });

  it("shows no panel for a call without arguments", () => {
    const bare = renderMarkup(<ApprovalCard chatId="c1" step={{ kind: "tool", id: "t4", name: "mcp__xpo__list", arg: "", done: false, approval: { id: "a4", input: "{}" } }} />);
    expect(bare).not.toContain("<dl");
    expect(bare).not.toContain("<pre");
    expect(bare).not.toContain("lucide-chevron-right");
  });

  it("offers Deny and a split Allow button, with the chat-wide grant in its menu", () => {
    const deny = merge.indexOf(">Deny<");
    const allow = merge.indexOf(">Allow<");
    expect(deny).toBeGreaterThan(-1);
    expect(allow).toBeGreaterThan(deny);
    expect(merge).not.toContain("Allow once");
    expect(merge).toMatch(/aria-label="More ways to allow"[^>]*aria-haspopup="menu"|aria-haspopup="menu"[^>]*aria-label="More ways to allow"/);
    expect(merge).not.toContain(">Allow for this chat<");
  });

  it("puts the headline and the buttons on one row when the call has no arguments", () => {
    const bare = renderMarkup(<ApprovalCard chatId="c1" step={{ kind: "tool", id: "t4", name: "mcp__xpo__list", arg: "", done: false, approval: { id: "a4", input: "{}" } }} />);
    expect(bare).toMatch(/flex-wrap[^>]*>.*List issues.*>Deny</s);
    expect(merge).not.toMatch(/flex-wrap[^>]*>.*Merge.*>Deny</s);
    // A zero flex-basis (flex-1) never wraps: a long headline would squeeze beside the buttons.
    expect(bare).toContain("flex-[1_1_auto]");
    expect(bare).not.toMatch(/class="[^"]*\bflex-1\b[^"]*"[^>]*>List issues/);
  });

  it("is focusable, shows the composer's focus border, and has no key hint", () => {
    expect(merge).toMatch(/aria-label="Approval needed"[^>]*tabindex="0"|tabindex="0"[^>]*aria-label="Approval needed"/);
    expect(merge).toContain("focus-within:border-[var(--color-border-focus)]");
    expect(merge).not.toContain("<kbd");
    expect(merge).not.toContain("↵");
  });

  it("names the tool in a code chip at the headline's size", () => {
    expect(merge).toMatch(/via <span class="[^"]*font-mono[^"]*bg-\[var\(--color-tint-rest\)\][^"]*">xpo merge<\/span>/);
    expect(merge).not.toMatch(/text-xs[^>]*>via/);
  });

  it("shows the command a shell call would run", () => {
    const bash = renderMarkup(
      <ApprovalCard
        chatId="c1"
        step={{ kind: "tool", id: "t2", name: "Bash", arg: "Clean the build", done: false, approval: { id: "a2", input: '{"command":"rm -rf build","description":"Clean the build"}' } }}
        defaultOpen
      />,
    );
    expect(bash).toContain("Run a shell command");
    expect(bash).toContain("rm -rf build");
  });

  it("puts the buttons in a footer, below the details", () => {
    const bash = renderMarkup(
      <ApprovalCard chatId="c1" step={{ kind: "tool", id: "t2", name: "Bash", arg: "Build", done: false, approval: { id: "a2", input: '{"command":"make build"}' } }} defaultOpen />,
    );
    expect(bash.indexOf(">Deny<")).toBeGreaterThan(bash.indexOf("make build"));
  });

  it("keeps the tool label on one line", () => {
    expect(merge).toMatch(/whitespace-nowrap[^>]*>via/);
  });

  it("lists the other arguments", () => {
    const update = renderMarkup(
      <ApprovalCard chatId="c1" step={{ kind: "tool", id: "t3", name: "mcp__xpo__update", arg: "xpo-1", done: false, approval: { id: "a3", input: '{"id":"xpo-1","status":"DONE"}' } }} defaultOpen />,
    );
    expect(update).toMatch(/status.*DONE/s);
  });
});

describe("ButtonLabel", () => {
  it("shows the label and no spinner when idle", () => {
    const idle = renderMarkup(<ButtonLabel text="Allow" sending={false} />);
    expect(idle).toContain(">Allow<");
    expect(idle).not.toContain("invisible");
    expect(idle).not.toContain("animate-spin");
  });

  // The hidden label keeps the button's size, so sending does not shift the row.
  it("lays the spinner over the hidden label while sending", () => {
    const sending = renderMarkup(<ButtonLabel text="Allow" sending />);
    expect(sending).toMatch(/class="invisible"[^>]*>Allow</);
    expect(sending).toMatch(/class="[^"]*\babsolute\b[^"]*\binset-0\b[^"]*"[^>]*>.*animate-spin/s);
  });
});
