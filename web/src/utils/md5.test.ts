import { describe, it, expect } from "vitest";
import { md5 } from "./md5";

describe("md5", () => {
  it("hashes empty string (RFC 1321 test vector)", () => {
    expect(md5("")).toBe("d41d8cd98f00b204e9800998ecf8427e");
  });

  it("hashes 'a' (RFC 1321 test vector)", () => {
    expect(md5("a")).toBe("0cc175b9c0f1b6a831c399e269772661");
  });

  it("hashes 'abc' (RFC 1321 test vector)", () => {
    expect(md5("abc")).toBe("900150983cd24fb0d6963f7d28e17f72");
  });

  it("hashes 'message digest' (RFC 1321 test vector)", () => {
    expect(md5("message digest")).toBe("f96b697d7cb7938d525a2f31aaf161d0");
  });

  it("hashes lowercase alphabet (RFC 1321 test vector)", () => {
    expect(md5("abcdefghijklmnopqrstuvwxyz")).toBe("c3fcd3d76192e4007dfb496cca67e13b");
  });

  it("produces consistent results for email-like strings", () => {
    const h1 = md5("user@example.com");
    const h2 = md5("user@example.com");
    expect(h1).toBe(h2);
    expect(h1).toHaveLength(32);
  });
});
