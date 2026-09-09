import { describe, it, expect } from "vitest";
import {
  parseDiffByFile,
  addLineNumbers,
  buildFileTree,
  flattenSingleChildDirs,
  buildSplitLines,
  hasDirtyDescendant,
} from "./diff-utils";

/* ─── parseDiffByFile ─── */

describe("parseDiffByFile", () => {
  it("returns empty map for empty string", () => {
    expect(parseDiffByFile("")).toEqual(new Map());
  });

  it("parses a single-file diff", () => {
    const diff = [
      "diff --git a/src/app.ts b/src/app.ts",
      "index abc1234..def5678 100644",
      "--- a/src/app.ts",
      "+++ b/src/app.ts",
      "@@ -1,3 +1,4 @@",
      " import foo from 'foo';",
      "+import bar from 'bar';",
      " ",
      " export default foo;",
    ].join("\n");

    const result = parseDiffByFile(diff);
    expect(result.size).toBe(1);
    expect(result.has("src/app.ts")).toBe(true);
    const lines = result.get("src/app.ts")!;
    expect(lines[0]).toBe("diff --git a/src/app.ts b/src/app.ts");
    expect(lines).toContain("+import bar from 'bar';");
  });

  it("parses a multi-file diff", () => {
    const diff = [
      "diff --git a/a.ts b/a.ts",
      "@@ -1,2 +1,2 @@",
      "-old",
      "+new",
      "diff --git a/b.ts b/b.ts",
      "@@ -1,1 +1,1 @@",
      "-x",
      "+y",
    ].join("\n");

    const result = parseDiffByFile(diff);
    expect(result.size).toBe(2);
    expect(result.has("a.ts")).toBe(true);
    expect(result.has("b.ts")).toBe(true);
  });

  it("filters out .xpo/ paths", () => {
    const diff = [
      "diff --git a/.xpo/issues.db b/.xpo/issues.db",
      "@@ -1 +1 @@",
      "-old",
      "+new",
      "diff --git a/src/main.ts b/src/main.ts",
      "@@ -1 +1 @@",
      "-a",
      "+b",
    ].join("\n");

    const result = parseDiffByFile(diff);
    expect(result.size).toBe(1);
    expect(result.has("src/main.ts")).toBe(true);
    expect(result.has(".xpo/issues.db")).toBe(false);
  });
});

/* ─── addLineNumbers ─── */

describe("addLineNumbers", () => {
  it("classifies meta lines before any hunk", () => {
    const lines = [
      "diff --git a/f.ts b/f.ts",
      "index abc..def 100644",
      "--- a/f.ts",
      "+++ b/f.ts",
    ];
    const result = addLineNumbers(lines);
    expect(result).toHaveLength(4);
    for (const ln of result) {
      expect(ln.type).toBe("meta");
      expect(ln.oldNum).toBe("");
      expect(ln.newNum).toBe("");
    }
  });

  it("parses hunk headers and tracks line numbers", () => {
    const lines = [
      "@@ -10,3 +20,4 @@",
      " context line",
      "-deleted line",
      "+added line",
      "+another add",
    ];
    const result = addLineNumbers(lines);
    expect(result[0]).toMatchObject({ type: "hunk", oldNum: "", newNum: "" });
    expect(result[1]).toMatchObject({
      type: "ctx",
      oldNum: "10",
      newNum: "20",
    });
    expect(result[2]).toMatchObject({
      type: "del",
      oldNum: "11",
      newNum: "",
    });
    expect(result[3]).toMatchObject({
      type: "add",
      oldNum: "",
      newNum: "21",
    });
    expect(result[4]).toMatchObject({
      type: "add",
      oldNum: "",
      newNum: "22",
    });
  });

  it("handles multiple hunks resetting line counters", () => {
    const lines = [
      "@@ -1,2 +1,2 @@",
      "-a",
      "+b",
      "@@ -50,1 +50,1 @@",
      " unchanged",
    ];
    const result = addLineNumbers(lines);
    const lastLine = result[result.length - 1];
    expect(lastLine).toMatchObject({ type: "ctx", oldNum: "50", newNum: "50" });
  });

  it("classifies lines before a hunk as meta", () => {
    const lines = ["some preamble text", "@@ -1,1 +1,1 @@", " ctx"];
    const result = addLineNumbers(lines);
    expect(result[0].type).toBe("meta");
    expect(result[1].type).toBe("hunk");
    expect(result[2].type).toBe("ctx");
  });

  it("classifies special prefixes as meta", () => {
    const prefixes = [
      "similarity index 100%",
      "rename from old.ts",
      "new file mode 100644",
      "deleted file mode 100644",
      "old mode 100755",
      "new mode 100644",
      "copy from a.ts",
      "\\ No newline at end of file",
    ];
    for (const line of prefixes) {
      const result = addLineNumbers(["@@ -1,1 +1,1 @@", line]);
      expect(result[1].type).toBe("meta");
    }
  });
});

/* ─── buildFileTree ─── */

describe("buildFileTree", () => {
  it("builds a flat file at root", () => {
    const tree = buildFileTree(["README.md"]);
    expect(tree).toHaveLength(1);
    expect(tree[0]).toMatchObject({
      name: "README.md",
      path: "README.md",
      isFile: true,
      children: [],
    });
  });

  it("builds a nested path", () => {
    const tree = buildFileTree(["src/utils/format.ts"]);
    expect(tree).toHaveLength(1);
    expect(tree[0].name).toBe("src");
    expect(tree[0].isFile).toBe(false);
    expect(tree[0].children[0].name).toBe("utils");
    expect(tree[0].children[0].children[0]).toMatchObject({
      name: "format.ts",
      path: "src/utils/format.ts",
      isFile: true,
    });
  });

  it("shares directory nodes for sibling files", () => {
    const tree = buildFileTree(["src/a.ts", "src/b.ts"]);
    expect(tree).toHaveLength(1);
    expect(tree[0].name).toBe("src");
    expect(tree[0].children).toHaveLength(2);
    expect(tree[0].children.map((c) => c.name).sort()).toEqual(["a.ts", "b.ts"]);
  });

  it("handles empty input", () => {
    expect(buildFileTree([])).toEqual([]);
  });

  it("handles multiple top-level files", () => {
    const tree = buildFileTree(["a.ts", "b.ts", "c.ts"]);
    expect(tree).toHaveLength(3);
    expect(tree.every((n) => n.isFile)).toBe(true);
  });
});

/* ─── flattenSingleChildDirs ─── */

describe("flattenSingleChildDirs", () => {
  it("collapses a single-child directory chain", () => {
    const tree = buildFileTree(["src/components/Button.tsx"]);
    const flat = flattenSingleChildDirs(tree);
    expect(flat).toHaveLength(1);
    expect(flat[0].name).toBe("src/components");
    expect(flat[0].isFile).toBe(false);
    expect(flat[0].children).toHaveLength(1);
    expect(flat[0].children[0].name).toBe("Button.tsx");
  });

  it("does not collapse when a dir has multiple children", () => {
    const tree = buildFileTree(["src/a.ts", "src/b.ts"]);
    const flat = flattenSingleChildDirs(tree);
    expect(flat).toHaveLength(1);
    expect(flat[0].name).toBe("src");
    expect(flat[0].children).toHaveLength(2);
  });

  it("does not collapse a single file child", () => {
    const tree = buildFileTree(["src/index.ts"]);
    const flat = flattenSingleChildDirs(tree);
    // src has one child that IS a file, so no collapse
    expect(flat[0].name).toBe("src");
    expect(flat[0].children[0].name).toBe("index.ts");
  });

  it("collapses deeply nested single-child chain", () => {
    const tree = buildFileTree(["a/b/c/d/file.ts"]);
    const flat = flattenSingleChildDirs(tree);
    expect(flat).toHaveLength(1);
    // Recursion collapses a->b into "a/b", then c->d into "c/d"
    expect(flat[0].name).toBe("a/b");
    expect(flat[0].children[0].name).toBe("c/d");
    expect(flat[0].children[0].children[0].name).toBe("file.ts");
  });

  it("handles empty input", () => {
    expect(flattenSingleChildDirs([])).toEqual([]);
  });
});

/* ─── buildSplitLines ─── */

describe("buildSplitLines", () => {
  it("returns empty array for empty input", () => {
    expect(buildSplitLines([])).toEqual([]);
  });

  it("pairs matched deletions and additions", () => {
    const lines = ["@@ -1,1 +1,1 @@", "-old line", "+new line"];
    const chunks = buildSplitLines(lines);
    // First chunk is the hunk header
    expect(chunks[0].left[0].text).toBe("@@ -1,1 +1,1 @@");
    // Second chunk has the del/add pair
    const data = chunks[1];
    expect(data.left).toHaveLength(1);
    expect(data.right).toHaveLength(1);
    expect(data.left[0]).toMatchObject({ type: "del", text: "-old line" });
    expect(data.right[0]).toMatchObject({ type: "add", text: "+new line" });
  });

  it("pads shorter side with empty lines", () => {
    const lines = ["@@ -1,2 +1,1 @@", "-line1", "-line2", "+replacement"];
    const chunks = buildSplitLines(lines);
    const data = chunks[1];
    expect(data.left).toHaveLength(2);
    expect(data.right).toHaveLength(2);
    expect(data.left[0].type).toBe("del");
    expect(data.left[1].type).toBe("del");
    expect(data.right[0].type).toBe("add");
    expect(data.right[1].type).toBe("empty");
  });

  it("places context lines on both sides", () => {
    const lines = ["@@ -5,3 +5,3 @@", " context", "-old", "+new"];
    const chunks = buildSplitLines(lines);
    // chunk[1] starts with context
    const data = chunks[1];
    expect(data.left[0]).toMatchObject({ type: "ctx", num: "5" });
    expect(data.right[0]).toMatchObject({ type: "ctx", num: "5" });
  });

  it("skips meta lines", () => {
    const lines = [
      "diff --git a/f.ts b/f.ts",
      "index abc..def 100644",
      "--- a/f.ts",
      "+++ b/f.ts",
      "@@ -1,1 +1,1 @@",
      "-a",
      "+b",
    ];
    const chunks = buildSplitLines(lines);
    // Only hunk header chunk and data chunk
    expect(chunks).toHaveLength(2);
  });

  it("handles additions with no deletions", () => {
    const lines = ["@@ -1,1 +1,3 @@", " existing", "+added1", "+added2"];
    const chunks = buildSplitLines(lines);
    const data = chunks[1];
    // Context on both sides, then adds on right with empty on left
    expect(data.left[0].type).toBe("ctx");
    expect(data.left[1].type).toBe("empty");
    expect(data.left[2].type).toBe("empty");
    expect(data.right[1].type).toBe("add");
    expect(data.right[2].type).toBe("add");
  });

  it("tracks line numbers correctly", () => {
    const lines = ["@@ -10,2 +20,2 @@", " ctx", "-del"];
    const chunks = buildSplitLines(lines);
    const data = chunks[1];
    expect(data.left[0].num).toBe("10");
    expect(data.right[0].num).toBe("20");
    expect(data.left[1].num).toBe("11");
  });

  it("starts a new chunk on a second hunk header", () => {
    const lines = [
      "@@ -1,1 +1,1 @@",
      " first",
      "@@ -50,1 +50,1 @@",
      " second",
    ];
    const chunks = buildSplitLines(lines);
    expect(chunks.length).toBeGreaterThanOrEqual(3);
  });
});

/* ─── hasDirtyDescendant ─── */

describe("hasDirtyDescendant", () => {
  it("returns true when a dirty file is a direct child of the directory", () => {
    const dirty = new Set(["src/app.ts"]);
    expect(hasDirtyDescendant("src", dirty)).toBe(true);
  });

  it("returns true when a dirty file is a nested descendant", () => {
    const dirty = new Set(["src/components/Button.tsx"]);
    expect(hasDirtyDescendant("src", dirty)).toBe(true);
  });

  it("returns false when no dirty files are under the directory", () => {
    const dirty = new Set(["lib/utils.ts"]);
    expect(hasDirtyDescendant("src", dirty)).toBe(false);
  });

  it("returns false for empty dirty set", () => {
    expect(hasDirtyDescendant("src", new Set())).toBe(false);
  });

  it("does not match a directory that is a prefix of another directory name", () => {
    const dirty = new Set(["src-old/file.ts"]);
    expect(hasDirtyDescendant("src", dirty)).toBe(false);
  });
});
