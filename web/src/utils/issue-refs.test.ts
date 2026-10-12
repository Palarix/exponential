import { describe, expect, it } from "vitest";
import { issueHref, issueIdFromHref, splitIssueIds } from "./issue-refs";

describe("splitIssueIds", () => {
  it("splits plain text around IDs", () => {
    expect(splitIssueIds("see xpo-a1b2c3 and xpo-111111.", "xpo-")).toEqual([
      "see ",
      { id: "xpo-a1b2c3" },
      " and ",
      { id: "xpo-111111" },
      ".",
    ]);
  });

  it("returns the text unchanged when there is no ID", () => {
    expect(splitIssueIds("nothing here", "xpo-")).toEqual(["nothing here"]);
  });

  it("matches IDs wrapped in punctuation", () => {
    expect(splitIssueIds("(xpo-a1b2c3), 'xpo-b2c3d4'", "xpo-")).toEqual([
      "(",
      { id: "xpo-a1b2c3" },
      "), '",
      { id: "xpo-b2c3d4" },
      "'",
    ]);
  });

  it("ignores short, non-hex and longer tokens", () => {
    for (const text of ["xpo-abc", "xpo-ghijkl", "xpo-a1b2c3d", "xpo-a1b2c3_x"]) {
      expect(splitIssueIds(text, "xpo-")).toEqual([text]);
    }
  });

  it("only matches the given prefix", () => {
    expect(splitIssueIds("abc-a1b2c3 xpo-a1b2c3", "abc-")).toEqual([{ id: "abc-a1b2c3" }, " xpo-a1b2c3"]);
  });

  it("escapes regex characters in the prefix", () => {
    expect(splitIssueIds("x.o-aabbcc xyo-aabbcc", "x.o-")).toEqual([{ id: "x.o-aabbcc" }, " xyo-aabbcc"]);
  });

  it("accepts the prefix as the API sends it, without the hyphen", () => {
    expect(splitIssueIds("see xpo-a1b2c3", "xpo")).toEqual(["see ", { id: "xpo-a1b2c3" }]);
    expect(splitIssueIds("xpoa1b2c3", "xpo")).toEqual(["xpoa1b2c3"]);
  });

  it("returns the text unchanged for an empty prefix", () => {
    expect(splitIssueIds("a1b2c3", "")).toEqual(["a1b2c3"]);
  });
});

describe("issueHref / issueIdFromHref", () => {
  it("round-trips an ID", () => {
    expect(issueHref("xpo-a1b2c3")).toBe("#/issues/xpo-a1b2c3");
    expect(issueIdFromHref("#/issues/xpo-a1b2c3")).toBe("xpo-a1b2c3");
  });

  it("returns null for other hrefs", () => {
    for (const href of [undefined, "", "https://example.com", "#/issues", "#/issues/xpo-a1b2c3/spec", "#/board"]) {
      expect(issueIdFromHref(href)).toBeNull();
    }
  });
});
