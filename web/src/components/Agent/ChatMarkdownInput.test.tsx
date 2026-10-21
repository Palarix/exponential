// @vitest-environment jsdom
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { act, createRef } from "react";
import { editorIn, mountDom, pressKey as key, setupDom, typeInto as type } from "../../test-dom";
import ChatMarkdownInput, { type ChatMarkdownInputHandle } from "./ChatMarkdownInput";
import { typing } from "./chat-thread-utils";

type Props = Parameters<typeof ChatMarkdownInput>[0];

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function mount(props: Partial<Props> = {}) {
  const onChange = vi.fn();
  const ref = createRef<ChatMarkdownInputHandle>();
  const all: Props = { value: "", onChange, placeholder: "Ask", ref, ...props };
  const m = mountDom(<ChatMarkdownInput {...all} />);
  cleanup = m.cleanup;
  return {
    onChange,
    ref,
    ...editorIn(m.host),
    rerender: (next: Partial<Props>) => m.rerender(<ChatMarkdownInput {...all} {...next} />),
  };
}

const last = (fn: ReturnType<typeof vi.fn>) => fn.mock.calls[fn.mock.calls.length - 1]?.[0];

describe("ChatMarkdownInput", () => {
  it("shows the value's Markdown formatted", () => {
    const { dom } = mount({ value: "Fix **this** and `that`\n\n- one\n- two" });
    expect(dom.querySelector("strong")?.textContent).toBe("this");
    expect(dom.querySelector("code")?.textContent).toBe("that");
    expect(dom.querySelectorAll("li")).toHaveLength(2);
  });

  it("formats Markdown as the user types and reports plain Markdown", () => {
    const { dom, editor, onChange } = mount();
    type(editor, "make it **bold** now");
    expect(dom.querySelector("strong")?.textContent).toBe("bold");
    expect(last(onChange)).toBe("make it **bold** now");
  });

  it("starts a list on '- '", () => {
    const { dom, editor, onChange } = mount();
    type(editor, "- first");
    expect(dom.querySelector("ul li")?.textContent).toBe("first");
    expect(last(onChange)).toBe("- first");
  });

  it("gives Enter to onKeyDown and adds no line when it handles it", () => {
    const onKeyDown = vi.fn(() => true);
    const { editor, dom, onChange } = mount({ value: "hello", onKeyDown });
    key(dom, "Enter");
    expect(onKeyDown).toHaveBeenCalledOnce();
    expect(editor.getText()).toBe("hello");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("starts a new paragraph on Shift+Enter, so a list can start on the next line", () => {
    const onKeyDown = vi.fn(() => false);
    const { editor, dom, onChange } = mount({ onKeyDown });
    type(editor, "Please:");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "- fix it");
    expect(last(onChange)).toBe("Please:\n\n- fix it");
  });

  it("adds a list item on Shift+Enter inside a list", () => {
    const { editor, dom, onChange } = mount();
    type(editor, "- one");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "two");
    expect(dom.querySelectorAll("li")).toHaveLength(2);
    expect(last(onChange)).toBe("- one\n- two");
  });

  it("adds a newline on Shift+Enter inside a code block", () => {
    const { editor, dom, onChange } = mount({ value: "```\na\n```" });
    act(() => {
      editor.commands.setTextSelection(editor.state.doc.content.size - 1);
    });
    key(dom, "Enter", { shiftKey: true });
    type(editor, "b");
    expect(last(onChange)).toBe("```\na\nb\n```");
  });

  it("opens a code block on Shift+Enter after a ``` line", () => {
    const { editor, dom, onChange } = mount();
    type(editor, "Look:");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "```");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "a = 1");
    expect(dom.querySelector("pre code")?.textContent).toBe("a = 1");
    expect(last(onChange)).toBe("Look:\n\n```\na = 1\n```");
  });

  it("opens a code block on Enter after a ```lang line, instead of handing Enter on", () => {
    const onKeyDown = vi.fn(() => true);
    const { editor, dom, onChange } = mount({ onKeyDown });
    type(editor, "```go");
    key(dom, "Enter");
    type(editor, "x := 1");
    expect(onKeyDown).not.toHaveBeenCalled();
    expect(last(onChange)).toBe("```go\nx := 1\n```");
  });

  it("leaves a code block on Shift+Enter after a closing ``` line", () => {
    const { editor, dom, onChange } = mount();
    type(editor, "```");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "a = 1");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "```");
    key(dom, "Enter", { shiftKey: true });
    type(editor, "done");
    expect(dom.querySelector("pre code")?.textContent).toBe("a = 1");
    expect(last(onChange)).toBe("```\na = 1\n```\n\ndone");
  });

  it("shows a new value from outside without reporting it back", () => {
    const { dom, onChange, rerender } = mount({ value: "first" });
    rerender({ value: "the **last** message" });
    expect(dom.querySelector("strong")?.textContent).toBe("last");
    expect(onChange).not.toHaveBeenCalled();
    rerender({ value: "" });
    expect(dom.textContent).toBe("");
  });

  it("keeps its content when the value is what it just reported", () => {
    const { editor, onChange, rerender, dom } = mount();
    type(editor, "* item");
    rerender({ value: last(onChange) });
    expect(dom.querySelectorAll("li")).toHaveLength(1);
    expect(dom.textContent).toBe("item");
  });

  it("hands pasted files to onPasteFiles", () => {
    const onPasteFiles = vi.fn();
    const { editor } = mount({ onPasteFiles });
    const file = new File(["x"], "shot.png", { type: "image/png" });
    const event = { clipboardData: { files: [file], getData: () => "" }, preventDefault: () => {} } as unknown as ClipboardEvent;
    const view = editor.view;
    const handled = view.someProp("handlePaste", (f) => f(view, event, view.state.doc.slice(0, 0)));
    expect(handled).toBe(true);
    expect(onPasteFiles).toHaveBeenCalledWith([file]);
  });

  it("counts as a field the user is typing in while it has text", () => {
    const { dom, rerender } = mount({ value: "draft" });
    expect(typing(dom)).toBe(true);
    rerender({ value: "  " });
    expect(typing(dom)).toBe(false);
    const area = document.createElement("textarea");
    area.value = "x";
    expect(typing(area)).toBe(true);
  });

  it("focuses and blurs through its handle", () => {
    // jsdom can't focus a contenteditable, so watch the calls.
    const { ref, dom } = mount({ value: "hi" });
    const focus = vi.spyOn(dom, "focus");
    const blur = vi.spyOn(dom, "blur");
    act(() => ref.current?.focus());
    expect(focus).toHaveBeenCalled();
    act(() => ref.current?.blur());
    expect(blur).toHaveBeenCalled();
  });
});
