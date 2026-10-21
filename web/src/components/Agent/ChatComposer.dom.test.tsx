// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { act, type ComponentProps } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { editorIn, mountDom, pressKey, setupDom, typeInto } from "../../test-dom";
import ChatComposer from "./ChatComposer";
import { createComposerDrafts } from "./composer-drafts";
import type { ComposerQueue } from "./useChatSession";

type Props = ComponentProps<typeof ChatComposer>;

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function mount(extra: Partial<Props> = {}) {
  const props: Props = {
    running: false,
    placeholder: "Ask",
    onDismissContext: () => {},
    status: undefined,
    prefs: { provider: "claude", model: "", effort: "" },
    onPrefsChange: () => {},
    onSend: vi.fn(async () => true),
    onStop: () => {},
    drafts: createComposerDrafts(),
    draftKey: "c1",
    ...extra,
  };
  const client = new QueryClient();
  const el = (p: Props) => (
    <QueryClientProvider client={client}>
      <TooltipPrimitive.Provider>
        <ChatComposer {...p} />
      </TooltipPrimitive.Provider>
    </QueryClientProvider>
  );
  const m = mountDom(el(props));
  cleanup = m.cleanup;
  return { props, host: m.host, ...editorIn(m.host), rerender: (next: Partial<Props>) => m.rerender(el({ ...props, ...next })) };
}

/** Lets the send promise settle. */
const settle = () => act(async () => {});

describe("ChatComposer Markdown input", () => {
  it("shows the open chat's own draft, formatted", () => {
    const drafts = createComposerDrafts();
    drafts.prefill("c1", "Let's implement **xpo-1**.");
    drafts.setText("c2", "Something else");
    const { dom, rerender } = mount({ drafts });
    expect(dom.querySelector("strong")?.textContent).toBe("xpo-1");
    expect(dom.textContent).not.toContain("Something else");
    rerender({ draftKey: "c2" });
    expect(dom.textContent).toBe("Something else");
    rerender({ draftKey: "c3" });
    expect(dom.textContent).toBe("");
  });

  it("keeps what the user types in the chat's draft, as Markdown", () => {
    const drafts = createComposerDrafts();
    const { editor } = mount({ drafts });
    typeInto(editor, "- fix **it**");
    expect(drafts.get("c1").text).toBe("- fix **it**");
  });

  it("sends the Markdown on Enter and clears the input", async () => {
    const drafts = createComposerDrafts();
    const { editor, dom, props } = mount({ drafts });
    typeInto(editor, "run `make test`");
    pressKey(dom, "Enter");
    await settle();
    expect(props.onSend).toHaveBeenCalledWith("run `make test`", [], undefined);
    expect(drafts.get("c1").text).toBe("");
    expect(dom.textContent).toBe("");
  });

  it("adds a line on Shift+Enter without sending", () => {
    const drafts = createComposerDrafts();
    const { editor, dom, props } = mount({ drafts });
    typeInto(editor, "one");
    pressKey(dom, "Enter", { shiftKey: true });
    typeInto(editor, "two");
    expect(props.onSend).not.toHaveBeenCalled();
    expect(drafts.get("c1").text).toBe("one\n\ntwo");
  });

  it("sends nothing on Enter while composing with an IME", () => {
    const { editor, dom, props } = mount();
    typeInto(editor, "日本");
    pressKey(dom, "Enter", { isComposing: true });
    expect(props.onSend).not.toHaveBeenCalled();
  });

  it("edits the last message on ↑ in an empty input", () => {
    const onEditLast = vi.fn();
    const { dom } = mount({ onEditLast });
    pressKey(dom, "ArrowUp");
    expect(onEditLast).toHaveBeenCalledOnce();
  });

  it("pulls the last queued message on ↑ in an empty input", () => {
    const edit = vi.fn();
    const queue: ComposerQueue = { items: [{ id: "q1", text: "a", time: "" }, { id: "q2", text: "b", time: "" }], canSteer: true, steer: () => {}, remove: () => {}, edit };
    const { dom } = mount({ queue, running: true });
    pressKey(dom, "ArrowUp");
    expect(edit).toHaveBeenCalledWith("q2");
  });

  it("puts the last message's Markdown in the input to edit it, and Esc cancels", () => {
    const drafts = createComposerDrafts();
    drafts.setText("c1", "my draft");
    const { dom } = mount({ drafts });
    act(() => drafts.startEdit("c1", "the **last** message", []));
    expect(dom.querySelector("strong")?.textContent).toBe("last");
    expect(drafts.get("c1").text).toBe("the **last** message");
    pressKey(dom, "Escape");
    expect(drafts.get("c1").editing).toBeUndefined();
    expect(dom.textContent).toBe("my draft");
  });

  it("sends an edit as an edit", async () => {
    const drafts = createComposerDrafts();
    const { dom, props } = mount({ drafts });
    act(() => drafts.startEdit("c1", "the last message", []));
    pressKey(dom, "Enter");
    await settle();
    expect(props.onSend).toHaveBeenCalledWith("the last message", [], { edit: true });
  });

  it("gives up focus on Esc when asked to", () => {
    const { dom } = mount({ blurOnEscape: true });
    const blur = vi.spyOn(dom, "blur");
    pressKey(dom, "Escape");
    expect(blur).toHaveBeenCalled();
  });
});
