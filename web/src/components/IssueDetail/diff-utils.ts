export interface FileTreeNode {
  name: string;
  path: string;
  isFile: boolean;
  children: FileTreeNode[];
}

export function buildFileTree(paths: string[]): FileTreeNode[] {
  const root: FileTreeNode[] = [];
  for (const filePath of paths) {
    const parts = filePath.split("/");
    let nodes = root;
    let currentPath = "";
    for (let i = 0; i < parts.length; i++) {
      currentPath = currentPath ? `${currentPath}/${parts[i]}` : parts[i];
      const isLast = i === parts.length - 1;
      let node = nodes.find((n) => n.name === parts[i]);
      if (!node) {
        node = {
          name: parts[i],
          path: currentPath,
          isFile: isLast,
          children: [],
        };
        nodes.push(node);
      }
      nodes = node.children;
    }
  }
  return root;
}

export function flattenSingleChildDirs(nodes: FileTreeNode[]): FileTreeNode[] {
  return nodes.map((node) => {
    if (
      !node.isFile &&
      node.children.length === 1 &&
      !node.children[0].isFile
    ) {
      const merged: FileTreeNode = {
        name: `${node.name}/${node.children[0].name}`,
        path: node.children[0].path,
        isFile: false,
        children: flattenSingleChildDirs(node.children[0].children),
      };
      return merged;
    }
    return { ...node, children: flattenSingleChildDirs(node.children) };
  });
}

export interface SplitLine {
  num: string;
  text: string;
  type: "ctx" | "add" | "del" | "empty";
}

export function buildSplitLines(
  lines: string[],
): { left: SplitLine[]; right: SplitLine[] }[] {
  const chunks: { left: SplitLine[]; right: SplitLine[] }[] = [];
  let oldLine = 0;
  let newLine = 0;
  let currentLeft: SplitLine[] = [];
  let currentRight: SplitLine[] = [];
  let pendingDels: string[] = [];
  let pendingAdds: string[] = [];
  let delStart = 0;
  let addStart = 0;

  const flushPending = () => {
    const max = Math.max(pendingDels.length, pendingAdds.length);
    for (let i = 0; i < max; i++) {
      if (i < pendingDels.length) {
        currentLeft.push({
          num: String(delStart + i),
          text: pendingDels[i],
          type: "del",
        });
      } else {
        currentLeft.push({ num: "", text: "", type: "empty" });
      }
      if (i < pendingAdds.length) {
        currentRight.push({
          num: String(addStart + i),
          text: pendingAdds[i],
          type: "add",
        });
      } else {
        currentRight.push({ num: "", text: "", type: "empty" });
      }
    }
    pendingDels = [];
    pendingAdds = [];
  };

  for (const line of lines) {
    if (line.startsWith("@@")) {
      flushPending();
      if (currentLeft.length > 0 || currentRight.length > 0) {
        chunks.push({ left: currentLeft, right: currentRight });
        currentLeft = [];
        currentRight = [];
      }
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (match) {
        oldLine = parseInt(match[1], 10);
        newLine = parseInt(match[2], 10);
      }
      chunks.push({
        left: [{ num: "", text: line, type: "ctx" }],
        right: [{ num: "", text: line, type: "ctx" }],
      });
    } else if (
      line.startsWith("diff --git") ||
      line.startsWith("index ") ||
      line.startsWith("---") ||
      line.startsWith("+++") ||
      line.startsWith("\\") ||
      line.startsWith("similarity") ||
      line.startsWith("rename ") ||
      line.startsWith("new file") ||
      line.startsWith("deleted file") ||
      line.startsWith("old mode") ||
      line.startsWith("new mode") ||
      line.startsWith("copy ")
    ) {
      continue;
    } else if (oldLine === 0 && newLine === 0) {
      continue;
    } else if (line.startsWith("-")) {
      if (pendingAdds.length > 0) {
        flushPending();
      }
      if (pendingDels.length === 0) delStart = oldLine;
      pendingDels.push(line);
      oldLine++;
    } else if (line.startsWith("+")) {
      if (pendingAdds.length === 0) addStart = newLine;
      pendingAdds.push(line);
      newLine++;
    } else {
      flushPending();
      currentLeft.push({ num: String(oldLine), text: line, type: "ctx" });
      currentRight.push({ num: String(newLine), text: line, type: "ctx" });
      oldLine++;
      newLine++;
    }
  }

  flushPending();
  if (currentLeft.length > 0 || currentRight.length > 0) {
    chunks.push({ left: currentLeft, right: currentRight });
  }

  return chunks;
}

export interface NumberedLine {
  oldNum: string;
  newNum: string;
  text: string;
  type: "ctx" | "add" | "del" | "hunk" | "meta";
}

export function addLineNumbers(lines: string[]): NumberedLine[] {
  const result: NumberedLine[] = [];
  let oldLine = 0;
  let newLine = 0;

  for (const line of lines) {
    if (line.startsWith("@@")) {
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (match) {
        oldLine = parseInt(match[1], 10);
        newLine = parseInt(match[2], 10);
      }
      result.push({ oldNum: "", newNum: "", text: line, type: "hunk" });
    } else if (
      line.startsWith("diff --git") ||
      line.startsWith("index ") ||
      line.startsWith("---") ||
      line.startsWith("+++") ||
      line.startsWith("\\") ||
      line.startsWith("similarity") ||
      line.startsWith("rename ") ||
      line.startsWith("new file") ||
      line.startsWith("deleted file") ||
      line.startsWith("old mode") ||
      line.startsWith("new mode") ||
      line.startsWith("copy ")
    ) {
      result.push({ oldNum: "", newNum: "", text: line, type: "meta" });
    } else if (oldLine === 0 && newLine === 0) {
      result.push({ oldNum: "", newNum: "", text: line, type: "meta" });
    } else if (line.startsWith("+")) {
      result.push({
        oldNum: "",
        newNum: String(newLine),
        text: line,
        type: "add",
      });
      newLine++;
    } else if (line.startsWith("-")) {
      result.push({
        oldNum: String(oldLine),
        newNum: "",
        text: line,
        type: "del",
      });
      oldLine++;
    } else {
      result.push({
        oldNum: String(oldLine),
        newNum: String(newLine),
        text: line,
        type: "ctx",
      });
      oldLine++;
      newLine++;
    }
  }
  return result;
}

export function hasDirtyDescendant(
  dirPath: string,
  dirtyFiles: Set<string>,
): boolean {
  const prefix = dirPath + "/";
  for (const f of dirtyFiles) {
    if (f.startsWith(prefix)) return true;
  }
  return false;
}

export function parseDiffByFile(diff: string): Map<string, string[]> {
  const result = new Map<string, string[]>();
  if (!diff) return result;
  const lines = diff.split("\n");
  let currentFile = "";
  let currentLines: string[] = [];
  for (const line of lines) {
    if (line.startsWith("diff --git")) {
      if (currentFile) result.set(currentFile, currentLines);
      const match = line.match(/ b\/(.+)$/);
      currentFile = match ? match[1] : "";
      currentLines = [line];
    } else if (currentFile) {
      currentLines.push(line);
    }
  }
  if (currentFile) result.set(currentFile, currentLines);
  for (const key of result.keys()) {
    if (key.startsWith(".xpo/")) result.delete(key);
  }
  return result;
}
