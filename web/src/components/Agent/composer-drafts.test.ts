import { describe, expect, it, vi } from "vitest";
import { createComposerDrafts, EMPTY_DRAFT, NEW_CHAT_POPUP, NEW_CHAT_VIEW } from "./composer-drafts";

const png = (name = "shot.png") => new File(["x"], name, { type: "image/png" });
const txt = (name = "notes.txt") => new File(["x"], name, { type: "text/plain" });

describe("composer drafts", () => {
  it("keeps a draft per chat", () => {
    const drafts = createComposerDrafts();
    drafts.setText("a", "hello A");
    drafts.setText("b", "hello B");
    expect(drafts.get("a").text).toBe("hello A");
    expect(drafts.get("b").text).toBe("hello B");
    expect(drafts.get("c")).toBe(EMPTY_DRAFT);
  });

  it("keeps the popup's and the full view's new chat drafts apart", () => {
    const drafts = createComposerDrafts();
    drafts.setText(NEW_CHAT_VIEW, "main idea");
    drafts.setText(NEW_CHAT_POPUP, "side question");
    expect(drafts.get(NEW_CHAT_VIEW).text).toBe("main idea");
    expect(drafts.get(NEW_CHAT_POPUP).text).toBe("side question");
  });

  it("moves a draft, replacing the target's and freeing its previews", () => {
    const revoke = vi.spyOn(URL, "revokeObjectURL");
    const drafts = createComposerDrafts();
    drafts.setText(NEW_CHAT_VIEW, "old");
    drafts.addFiles(NEW_CHAT_VIEW, [png("old.png")]);
    const replaced = drafts.get(NEW_CHAT_VIEW).attachments[0];
    drafts.setText(NEW_CHAT_POPUP, "moved");
    drafts.addFiles(NEW_CHAT_POPUP, [png("moved.png")]);
    const moved = drafts.get(NEW_CHAT_POPUP).attachments[0];
    drafts.move(NEW_CHAT_POPUP, NEW_CHAT_VIEW);
    expect(drafts.get(NEW_CHAT_VIEW)).toEqual({ text: "moved", attachments: [moved] });
    expect(drafts.get(NEW_CHAT_POPUP)).toBe(EMPTY_DRAFT);
    expect(revoke).toHaveBeenCalledWith(replaced.preview);
    expect(revoke).not.toHaveBeenCalledWith(moved.preview);
    revoke.mockRestore();
  });

  it("moves nothing from an empty draft", () => {
    const drafts = createComposerDrafts();
    drafts.setText(NEW_CHAT_VIEW, "keep");
    drafts.move(NEW_CHAT_POPUP, NEW_CHAT_VIEW);
    expect(drafts.get(NEW_CHAT_VIEW).text).toBe("keep");
  });

  it("returns the same object until the draft changes", () => {
    const drafts = createComposerDrafts();
    drafts.setText("a", "x");
    const first = drafts.get("a");
    expect(drafts.get("a")).toBe(first);
    drafts.setText("a", "xy");
    expect(drafts.get("a")).not.toBe(first);
  });

  it("scopes attachments per chat, with a preview for images only", () => {
    const drafts = createComposerDrafts();
    expect(drafts.addFiles("a", [png(), txt()])).toBeUndefined();
    const [image, file] = drafts.get("a").attachments;
    expect(image.file.name).toBe("shot.png");
    expect(image.preview).toMatch(/^blob:/);
    expect(file.preview).toBeUndefined();
    expect(drafts.get("b").attachments).toEqual([]);
  });

  it("reports files that do not fit", () => {
    const drafts = createComposerDrafts();
    expect(drafts.addFiles("a", [new File([], "empty.txt", { type: "text/plain" })])).toBe("empty.txt is empty");
    expect(drafts.get("a").attachments).toEqual([]);
  });

  it("frees an image's preview when it is removed", () => {
    const revoke = vi.spyOn(URL, "revokeObjectURL");
    const drafts = createComposerDrafts();
    drafts.addFiles("a", [png(), txt()]);
    const [image, file] = drafts.get("a").attachments;
    drafts.removeAttachments("a", new Set([image.key]));
    expect(revoke).toHaveBeenCalledWith(image.preview);
    expect(drafts.get("a").attachments).toEqual([file]);
    revoke.mockRestore();
  });

  it("clears only the sent text and attachments of one chat", () => {
    const drafts = createComposerDrafts();
    drafts.setText("a", "send me");
    drafts.addFiles("a", [txt("one.txt")]);
    const sent = drafts.get("a").attachments.map((a) => a.key);
    drafts.addFiles("a", [txt("later.txt")]);
    drafts.setText("b", "keep me");
    drafts.sent("a", new Set(sent));
    expect(drafts.get("a").text).toBe("");
    expect(drafts.get("a").attachments.map((a) => a.file.name)).toEqual(["later.txt"]);
    expect(drafts.get("b").text).toBe("keep me");
  });

  it("drops an empty draft entirely", () => {
    const drafts = createComposerDrafts();
    drafts.setText("a", "x");
    drafts.sent("a", new Set());
    expect(drafts.get("a")).toBe(EMPTY_DRAFT);
  });

  it("prefills text, flagged until the composer takes it", () => {
    const drafts = createComposerDrafts();
    drafts.addFiles("a", [txt()]);
    drafts.prefill("a", "Let's implement xpo-1.");
    expect(drafts.get("a")).toMatchObject({ text: "Let's implement xpo-1.", prefilled: true });
    expect(drafts.get("a").attachments).toHaveLength(1);
    drafts.prefillUsed("a");
    expect(drafts.get("a").prefilled).toBeUndefined();
    expect(drafts.get("a").text).toBe("Let's implement xpo-1.");
  });

  it("tells subscribers about changes, until they unsubscribe", () => {
    const drafts = createComposerDrafts();
    const listener = vi.fn();
    const unsubscribe = drafts.subscribe(listener);
    drafts.setText("a", "x");
    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
    drafts.setText("a", "xy");
    expect(listener).toHaveBeenCalledTimes(1);
  });
});
