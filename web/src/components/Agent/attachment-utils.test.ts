import { describe, expect, it } from "vitest";
import { addAttachments, attachmentUrl, formatSize, isAgentImage, MAX_ATTACHMENTS, MAX_FILE_BYTES, MAX_IMAGE_BYTES, toSendAttachment } from "./attachment-utils";

const file = (name: string, type: string, size: number) => ({ name, type, size });

describe("isAgentImage", () => {
  it("accepts the image types the agents take", () => {
    for (const t of ["image/png", "image/jpeg", "image/gif", "image/webp"]) expect(isAgentImage(t)).toBe(true);
    for (const t of ["image/svg+xml", "image/heic", "text/plain", ""]) expect(isAgentImage(t)).toBe(false);
  });
});

describe("addAttachments", () => {
  it("adds files under the limits", () => {
    const a = file("a.png", "image/png", 10);
    const b = file("b.txt", "text/plain", 10);
    expect(addAttachments([a], [b])).toEqual({ added: [b] });
  });

  it("refuses an image over the image limit and names it", () => {
    const big = file("big.png", "image/png", MAX_IMAGE_BYTES + 1);
    const ok = file("ok.txt", "text/plain", 1);
    const res = addAttachments([], [big, ok]);
    expect(res.added).toEqual([ok]);
    expect(res.error).toContain("big.png");
    expect(res.error).toContain("5 MB");
  });

  it("allows a non-image up to the file limit", () => {
    const log = file("log.txt", "text/plain", MAX_IMAGE_BYTES + 1);
    expect(addAttachments([], [log]).added).toEqual([log]);
    expect(addAttachments([], [file("dump.bin", "", MAX_FILE_BYTES + 1)]).error).toContain("10 MB");
  });

  it("refuses empty files", () => {
    expect(addAttachments([], [file("e.txt", "text/plain", 0)])).toEqual({ added: [], error: "e.txt is empty" });
  });

  it("stops at the count limit", () => {
    const have = Array.from({ length: MAX_ATTACHMENTS - 1 }, (_, i) => file(`${i}.txt`, "text/plain", 1));
    const res = addAttachments(have, [file("x.txt", "text/plain", 1), file("y.txt", "text/plain", 1)]);
    expect(res.added.map((f) => f.name)).toEqual(["x.txt"]);
    expect(res.error).toBe(`At most ${MAX_ATTACHMENTS} attachments per message`);
  });
});

describe("toSendAttachment", () => {
  it("encodes the file as base64", async () => {
    const f = new File([new Uint8Array([0, 1, 2, 250, 255])], "b.bin", { type: "application/octet-stream" });
    expect(await toSendAttachment(f)).toEqual({ name: "b.bin", mime: "application/octet-stream", data: "AAEC+v8=" });
  });

  it("names a pasted file without a name", async () => {
    const f = new File(["x"], "", { type: "image/png" });
    expect((await toSendAttachment(f)).name).toBe("pasted.png");
  });
});

describe("helpers", () => {
  it("formats sizes", () => {
    expect(formatSize(900)).toBe("900 B");
    expect(formatSize(2048)).toBe("2 KB");
    expect(formatSize(3.5 * 1024 * 1024)).toBe("3.5 MB");
  });

  it("builds the attachment URL", () => {
    expect(attachmentUrl("c 1", "a1")).toBe("/api/local/chats/c%201/attachments/a1");
  });
});
