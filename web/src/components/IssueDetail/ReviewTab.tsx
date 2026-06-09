import { useState, useEffect, useCallback } from "react";
import { fetchIssueCommits, fetchIssueFiles, fetchIssueDiff, mergeIssue, ApiError } from "../../api/client";
import type { Issue, CommitInfo, FileInfo } from "../../api/client";
import { GitCommitVertical, ChevronDown, ChevronRight, FileDiff, GitMerge } from "lucide-react";

interface ReviewTabProps {
  issue: Issue;
  onRefresh: () => void;
}

export default function ReviewTab({ issue, onRefresh }: ReviewTabProps) {
  const [commits, setCommits] = useState<CommitInfo[]>([]);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [diff, setDiff] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());
  const [showMerge, setShowMerge] = useState(false);
  const [mergeStrategy, setMergeStrategy] = useState("squash");
  const [merging, setMerging] = useState(false);
  const [mergeError, setMergeError] = useState<string | null>(null);

  const bs = issue.branch_stats;

  useEffect(() => {
    if (!bs || bs.commits === 0) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    Promise.all([
      fetchIssueCommits(issue.id),
      fetchIssueFiles(issue.id),
      fetchIssueDiff(issue.id),
    ])
      .then(([c, f, d]) => {
        setCommits(c || []);
        setFiles(f || []);
        setDiff(d || "");
      })
      .catch((err) => {
        setError(err instanceof ApiError ? err.message : "Failed to load review data");
      })
      .finally(() => setLoading(false));
  }, [issue.id, bs?.commits]);

  const toggleFile = useCallback((path: string) => {
    setExpandedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const expandAll = useCallback(() => {
    setExpandedFiles(new Set(files.map((f) => f.path)));
  }, [files]);

  const collapseAll = useCallback(() => {
    setExpandedFiles(new Set());
  }, []);

  const fileDiffs = parseDiffByFile(diff);

  const handleMerge = useCallback(async () => {
    setMerging(true);
    setMergeError(null);
    try {
      await mergeIssue(issue.id, { strategy: mergeStrategy });
      setShowMerge(false);
      onRefresh();
    } catch (err) {
      setMergeError(err instanceof ApiError ? err.message : "Merge failed");
    } finally {
      setMerging(false);
    }
  }, [issue.id, mergeStrategy, onRefresh]);

  if (!bs || bs.commits === 0) {
    return (
      <div className="text-center py-12 text-sm text-[var(--color-text-muted)]">
        No branch detected for this issue.
      </div>
    );
  }

  if (loading) {
    return (
      <div className="text-center py-12 text-sm text-[var(--color-text-muted)]">
        Loading review data...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12 text-sm text-[var(--color-error)]">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] p-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm font-mono text-[var(--color-text-primary)]">{bs.branch}</div>
            <div className="flex items-center gap-3 mt-1 text-sm text-[var(--color-text-muted)]">
              <span className="inline-flex items-center gap-1">
                <GitCommitVertical size={14} />
                <span className="font-mono">{bs.head_sha}</span>
              </span>
              <span>{bs.commits} {bs.commits === 1 ? "commit" : "commits"}</span>
              <span>{bs.files_changed} {bs.files_changed === 1 ? "file" : "files"}</span>
              <span className="text-green-500">+{bs.insertions}</span>
              <span className="text-red-500">-{bs.deletions}</span>
            </div>
          </div>
          <button
            onClick={() => setShowMerge(!showMerge)}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity"
          >
            <GitMerge size={14} />
            Merge
          </button>
        </div>

        {/* Merge panel */}
        {showMerge && (
          <div className="mt-4 pt-4 border-t border-[var(--color-border-subtle)]">
            <div className="text-xs font-medium text-[var(--color-text-muted)] mb-2">Strategy</div>
            <div className="flex items-center gap-2 mb-3">
              {(["squash", "merge", "ff"] as const).map((s) => (
                <button
                  key={s}
                  onClick={() => setMergeStrategy(s)}
                  className={`px-3 py-1.5 text-sm rounded-[var(--radius-md)] border transition-colors ${
                    mergeStrategy === s
                      ? "border-[var(--color-accent-primary)] text-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/10"
                      : "border-[var(--color-border-default)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-focus)]"
                  }`}
                >
                  {s === "squash" ? "Squash" : s === "merge" ? "Merge commit" : "Fast-forward"}
                </button>
              ))}
            </div>
            {mergeError && (
              <div className="text-sm text-[var(--color-error)] mb-2">{mergeError}</div>
            )}
            <button
              onClick={handleMerge}
              disabled={merging}
              className="px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {merging ? "Merging..." : "Confirm merge"}
            </button>
          </div>
        )}
      </div>

      {/* Commits */}
      <div>
        <h3 className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-2">
          Commits ({commits.length})
        </h3>
        <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] divide-y divide-[var(--color-border-subtle)]">
          {commits.map((c) => (
            <div key={c.sha} className="flex items-center gap-3 px-4 py-2.5 text-sm">
              <GitCommitVertical size={14} className="text-[var(--color-text-muted)] shrink-0" />
              <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">{c.sha}</span>
              <span className="text-[var(--color-text-primary)] truncate">{c.message}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Files */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
            Files changed ({files.length})
          </h3>
          <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
            <button onClick={expandAll} className="hover:text-[var(--color-text-primary)] transition-colors">Expand all</button>
            <span>·</span>
            <button onClick={collapseAll} className="hover:text-[var(--color-text-primary)] transition-colors">Collapse all</button>
          </div>
        </div>
        <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] divide-y divide-[var(--color-border-subtle)]">
          {files.map((f) => {
            const isExpanded = expandedFiles.has(f.path);
            const fileDiff = fileDiffs.get(f.path);
            return (
              <div key={f.path}>
                <button
                  onClick={() => toggleFile(f.path)}
                  className="flex items-center gap-3 w-full px-4 py-2 text-sm text-left hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  {isExpanded ? <ChevronDown size={14} className="shrink-0 text-[var(--color-text-muted)]" /> : <ChevronRight size={14} className="shrink-0 text-[var(--color-text-muted)]" />}
                  <FileDiff size={14} className={`shrink-0 ${f.status === "A" ? "text-green-500" : f.status === "D" ? "text-red-500" : "text-[var(--color-text-muted)]"}`} />
                  <span className="font-mono text-[var(--color-text-primary)] truncate">{f.path}</span>
                  <span className="ml-auto shrink-0 text-xs tabular-nums">
                    <span className="text-green-500">+{f.insertions}</span>{" "}
                    <span className="text-red-500">-{f.deletions}</span>
                  </span>
                </button>
                {isExpanded && fileDiff && (
                  <div className="border-t border-[var(--color-border-subtle)] bg-[var(--color-bg-secondary)] overflow-x-auto">
                    <pre className="text-xs leading-5 font-mono p-0 m-0">
                      {fileDiff.map((line, i) => (
                        <div
                          key={i}
                          className={`px-4 ${
                            line.startsWith("+") && !line.startsWith("+++")
                              ? "bg-green-500/10 text-green-400"
                              : line.startsWith("-") && !line.startsWith("---")
                              ? "bg-red-500/10 text-red-400"
                              : line.startsWith("@@")
                              ? "text-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/5"
                              : "text-[var(--color-text-secondary)]"
                          }`}
                        >
                          {line}
                        </div>
                      ))}
                    </pre>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function parseDiffByFile(diff: string): Map<string, string[]> {
  const result = new Map<string, string[]>();
  if (!diff) return result;

  const lines = diff.split("\n");
  let currentFile = "";
  let currentLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith("diff --git")) {
      if (currentFile) {
        result.set(currentFile, currentLines);
      }
      const match = line.match(/b\/(.+)$/);
      currentFile = match ? match[1] : "";
      currentLines = [];
    } else if (currentFile) {
      currentLines.push(line);
    }
  }
  if (currentFile) {
    result.set(currentFile, currentLines);
  }

  return result;
}
