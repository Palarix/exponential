import { useState, useEffect, useCallback, useMemo } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkBreaks from "remark-breaks";
import {
  fetchIssueCommits, fetchIssueFiles, fetchIssueDiff, fetchCommitDiff,
  fetchMergeability, mergeIssue, addDraft, ApiError,
} from "../../api/client";
import type { Issue, CommitInfo, FileInfo, Mergeability } from "../../api/client";
import { GitCommitVertical, GitMerge, X, Check, AlertTriangle, FileDiff, ChevronDown, ChevronRight, GitBranch, Search } from "lucide-react";
import { StatusIcon, Avatar } from "../ui";
import { formatRelativeTime } from "../../utils/format";

interface MergeViewProps {
  issue: Issue;
  onClose: () => void;
  onMerged: () => void;
}

type Tab = "files" | "commits" | "conversation";

export default function MergeView({ issue, onClose, onMerged }: MergeViewProps) {
  const [commits, setCommits] = useState<CommitInfo[]>([]);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [diff, setDiff] = useState("");
  const [mergeability, setMergeability] = useState<Mergeability | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("files");

  // Files tab
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileFilter, setFileFilter] = useState("");

  // Commits tab
  const [selectedCommit, setSelectedCommit] = useState<string | null>(null);
  const [commitDiffText, setCommitDiffText] = useState("");
  const [commitDiffLoading, setCommitDiffLoading] = useState(false);

  // Conversation tab
  const [newComment, setNewComment] = useState("");
  const [commentSaving, setCommentSaving] = useState(false);

  // Merge
  const [strategyOpen, setStrategyOpen] = useState(false);
  const [mergeStrategy, setMergeStrategy] = useState("squash");
  const [merging, setMerging] = useState(false);
  const [mergeError, setMergeError] = useState<string | null>(null);

  const bs = issue.branch_stats!;

  useEffect(() => {
    setLoading(true);
    setError(null);
    Promise.all([
      fetchIssueCommits(issue.id),
      fetchIssueFiles(issue.id),
      fetchIssueDiff(issue.id),
      fetchMergeability(issue.id),
    ])
      .then(([c, f, d, m]) => {
        setCommits(c || []);
        setFiles(f || []);
        setDiff(d || "");
        setMergeability(m);
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [issue.id]);

  const loadCommitDiff = useCallback(async (sha: string) => {
    if (selectedCommit === sha) { setSelectedCommit(null); return; }
    setSelectedCommit(sha);
    setCommitDiffLoading(true);
    try {
      setCommitDiffText(await fetchCommitDiff(issue.id, sha));
    } catch { setCommitDiffText(""); }
    finally { setCommitDiffLoading(false); }
  }, [issue.id, selectedCommit]);

  const handleAddComment = useCallback(async () => {
    if (!newComment.trim()) return;
    setCommentSaving(true);
    try {
      await addDraft(issue.id, "COMMENT", { id: `cmt-${crypto.randomUUID()}`, text: newComment });
      setNewComment("");
    } catch (err) {
      console.error(err);
    } finally {
      setCommentSaving(false);
    }
  }, [issue.id, newComment]);

  const fileDiffs = useMemo(() => parseDiffByFile(diff), [diff]);
  const commitFileDiffs = useMemo(() => parseDiffByFile(commitDiffText), [commitDiffText]);

  const filteredFiles = useMemo(() => {
    if (!fileFilter) return files;
    const q = fileFilter.toLowerCase();
    return files.filter((f) => f.path.toLowerCase().includes(q));
  }, [files, fileFilter]);

  const handleMerge = useCallback(async () => {
    setMerging(true);
    setMergeError(null);
    try {
      await mergeIssue(issue.id, { strategy: mergeStrategy });
      onMerged();
    } catch (err) {
      setMergeError(err instanceof ApiError ? err.message : "Merge failed");
    } finally { setMerging(false); }
  }, [issue.id, mergeStrategy, onMerged]);

  const strategyLabel = mergeStrategy === "squash" ? "Squash and merge"
    : mergeStrategy === "merge" ? "Create merge commit" : "Fast-forward";
  const canMerge = mergeability?.can_merge ?? true;

  if (loading) return <div className="h-full flex items-center justify-center text-sm text-[var(--color-text-muted)]">Loading...</div>;
  if (error) return (
    <div className="h-full flex flex-col items-center justify-center gap-3">
      <span className="text-sm text-[var(--color-error)]">{error}</span>
      <button onClick={onClose} className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]">Go back</button>
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-[var(--color-bg-primary)]">
      {/* ── Top bar ── */}
      <div className="shrink-0 flex items-center px-5 h-12 border-b border-[var(--color-border-default)] bg-[var(--color-bg-secondary)]">
        <div className="flex items-center gap-2">
          <StatusIcon status={issue.status} size={16} isInferred={issue.is_inferred} />
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">{issue.title}</span>
          <span className="text-sm font-mono text-[var(--color-text-muted)]">{issue.id}</span>
        </div>
        <div className="flex-1 flex justify-end">
          <button onClick={onClose} className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors">
            <X size={20} />
          </button>
        </div>
      </div>

      {/* ── Merge bar (3-column) ── */}
      <div className="shrink-0 border-b border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-5 py-3">
        <div className="grid grid-cols-3 items-center gap-3">
          {/* Left: branch direction */}
          <div className="flex items-center gap-2 text-sm">
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)] font-mono text-xs">
              <GitBranch size={12} />main
            </span>
            <span className="text-[var(--color-text-muted)]">←</span>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[var(--color-accent-primary)]/10 text-[var(--color-accent-primary)] font-mono text-xs truncate">
              <GitBranch size={12} className="shrink-0" /><span className="truncate">{bs.branch}</span>
            </span>
          </div>

          {/* Center: mergeability */}
          <div className="flex justify-center">
            {!canMerge ? (
              <div className="flex items-center gap-2 text-sm text-amber-500">
                <AlertTriangle size={16} />
                <span>{mergeability?.blockers?.[0] || "Cannot merge"}</span>
              </div>
            ) : mergeError ? (
              <div className="flex items-center gap-2 text-sm text-[var(--color-error)]">
                <AlertTriangle size={16} />
                <span>{mergeError}</span>
              </div>
            ) : (
              <div className="flex items-center gap-1.5 text-sm text-green-500">
                <Check size={16} />Ready to merge
              </div>
            )}
          </div>

          {/* Right: merge button + strategy */}
          <div className="flex justify-end items-center">
            <button
              onClick={handleMerge}
              disabled={merging || !canMerge}
              className="flex items-center h-8 gap-2 px-4 text-sm font-medium rounded-l-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
            >
              <GitMerge size={14} />
              {merging ? "Merging..." : strategyLabel}
            </button>
            <div className="relative">
              <button
                onClick={() => setStrategyOpen(!strategyOpen)}
                className="flex items-center h-8 px-2 rounded-r-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity border-l border-white/20"
              >
                <ChevronDown size={14} />
              </button>
              {strategyOpen && (
                <>
                  <div className="fixed inset-0 z-40" onClick={() => setStrategyOpen(false)} />
                  <div className="absolute right-0 top-full mt-1 z-50 w-56 rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] shadow-[var(--shadow-lg)] py-1">
                    {([
                      { key: "squash", label: "Squash and merge", desc: "Single commit on main" },
                      { key: "merge", label: "Create merge commit", desc: "Preserves branch history" },
                      { key: "ff", label: "Fast-forward", desc: "Linear, no merge commit" },
                    ] as const).map((opt) => (
                      <button key={opt.key} onClick={() => { setMergeStrategy(opt.key); setStrategyOpen(false); }}
                        className={`w-full px-3 py-2 text-left hover:bg-[var(--color-bg-hover)] transition-colors ${mergeStrategy === opt.key ? "text-[var(--color-accent-primary)]" : ""}`}>
                        <div className="text-sm">{opt.label}</div>
                        <div className="text-xs text-[var(--color-text-muted)]">{opt.desc}</div>
                      </button>
                    ))}
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* ── Tabs ── */}
      <div className="shrink-0 flex items-center gap-1 px-5 border-b border-[var(--color-border-subtle)]">
        {([
          { key: "commits" as Tab, label: "Commits", count: commits.length },
          { key: "files" as Tab, label: "Files changed", count: files.length },
          { key: "conversation" as Tab, label: "Conversation", count: issue.comments?.length || 0 },
        ]).map((t) => (
          <button key={t.key} onClick={() => setActiveTab(t.key)}
            className={`px-3 py-2.5 text-sm font-medium transition-colors relative ${activeTab === t.key ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"}`}>
            {t.label}
            <span className="ml-1.5 text-xs tabular-nums px-1.5 py-0.5 rounded-full bg-[var(--color-bg-tertiary)]">{t.count}</span>
            {activeTab === t.key && <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--color-accent-primary)]" />}
          </button>
        ))}
      </div>

      {/* ── Content ── */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "commits" && (
          <CommitsTab commits={commits} selectedCommit={selectedCommit} commitDiffs={commitFileDiffs}
            commitDiffLoading={commitDiffLoading} onSelectCommit={loadCommitDiff} />
        )}
        {activeTab === "files" && (
          <FilesTab files={filteredFiles} fileDiffs={fileDiffs} selectedFile={selectedFile}
            onSelectFile={setSelectedFile} fileFilter={fileFilter} onFilterChange={setFileFilter} />
        )}
        {activeTab === "conversation" && (
          <ConversationTab issue={issue} newComment={newComment} onNewCommentChange={setNewComment}
            onAddComment={handleAddComment} saving={commentSaving} />
        )}
      </div>
    </div>
  );
}

/* ─── Commits Tab: list left, diff right ─── */
function CommitsTab({ commits, selectedCommit, commitDiffs, commitDiffLoading, onSelectCommit }: {
  commits: CommitInfo[]; selectedCommit: string | null;
  commitDiffs: Map<string, string[]>; commitDiffLoading: boolean;
  onSelectCommit: (sha: string) => void;
}) {
  return (
    <div className="h-full flex">
      <div className="w-80 shrink-0 border-r border-[var(--color-border-default)] overflow-y-auto bg-[var(--color-bg-secondary)]">
        {commits.map((c) => (
          <button key={c.sha} onClick={() => onSelectCommit(c.sha)}
            className={`flex items-start gap-3 w-full px-4 py-3 text-left border-b border-[var(--color-border-subtle)] transition-colors ${selectedCommit === c.sha ? "bg-[var(--color-accent-primary)]/5" : "hover:bg-[var(--color-bg-hover)]"}`}>
            <GitCommitVertical size={16} className="text-[var(--color-text-muted)] shrink-0 mt-0.5" />
            <div className="min-w-0 flex-1">
              <div className="text-sm text-[var(--color-text-primary)] leading-snug">{c.message}</div>
              <div className="text-xs text-[var(--color-text-muted)] mt-1">
                <span className="font-mono">{c.sha}</span> · {c.author} · {formatRelativeTime(c.date)}
              </div>
            </div>
          </button>
        ))}
      </div>
      <div className="flex-1 overflow-auto">
        {!selectedCommit && (
          <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
            Select a commit to view its changes
          </div>
        )}
        {selectedCommit && commitDiffLoading && (
          <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">Loading...</div>
        )}
        {selectedCommit && !commitDiffLoading && <DiffViewer diffs={commitDiffs} />}
      </div>
    </div>
  );
}

/* ─── Files Tab: file list left, diff right ─── */
function FilesTab({ files, fileDiffs, selectedFile, onSelectFile, fileFilter, onFilterChange }: {
  files: FileInfo[]; fileDiffs: Map<string, string[]>;
  selectedFile: string | null; onSelectFile: (f: string | null) => void;
  fileFilter: string; onFilterChange: (v: string) => void;
}) {
  const visibleDiffs = useMemo(() => {
    if (selectedFile) {
      const d = fileDiffs.get(selectedFile);
      return d ? new Map([[selectedFile, d]]) : new Map();
    }
    return fileDiffs;
  }, [selectedFile, fileDiffs]);

  return (
    <div className="h-full flex">
      <div className="w-72 shrink-0 border-r border-[var(--color-border-default)] overflow-y-auto bg-[var(--color-bg-secondary)]">
        <div className="p-3 border-b border-[var(--color-border-subtle)]">
          <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-bg-tertiary)] border border-[var(--color-border-default)]">
            <Search size={13} className="text-[var(--color-text-muted)]" />
            <input value={fileFilter} onChange={(e) => onFilterChange(e.target.value)} placeholder="Filter files..."
              className="flex-1 text-xs bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" />
          </div>
        </div>
        {selectedFile && (
          <button onClick={() => onSelectFile(null)}
            className="w-full px-4 py-2 text-xs text-[var(--color-accent-primary)] hover:bg-[var(--color-bg-hover)] text-left border-b border-[var(--color-border-subtle)]">
            ← Show all files
          </button>
        )}
        {files.map((f) => (
          <button key={f.path} onClick={() => onSelectFile(selectedFile === f.path ? null : f.path)}
            className={`flex items-center gap-2 w-full px-4 py-2 text-xs text-left border-b border-[var(--color-border-subtle)] transition-colors ${selectedFile === f.path ? "bg-[var(--color-accent-primary)]/5 text-[var(--color-accent-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]"}`}>
            <FileDiff size={12} className={`shrink-0 ${f.status === "A" ? "text-green-500" : f.status === "D" ? "text-red-500" : "text-[var(--color-text-muted)]"}`} />
            <span className="font-mono truncate flex-1">{f.path}</span>
            <span className="shrink-0 tabular-nums"><span className="text-green-500">+{f.insertions}</span> <span className="text-red-500">-{f.deletions}</span></span>
          </button>
        ))}
      </div>
      <div className="flex-1 overflow-auto">
        <DiffViewer diffs={visibleDiffs} />
      </div>
    </div>
  );
}

/* ─── Conversation Tab ─── */
function ConversationTab({ issue, newComment, onNewCommentChange, onAddComment, saving }: {
  issue: Issue; newComment: string; onNewCommentChange: (v: string) => void;
  onAddComment: () => void; saving: boolean;
}) {
  const comments = issue.comments || [];
  return (
    <div className="h-full overflow-y-auto">
      {comments.length === 0 && !newComment && (
        <div className="px-5 py-12 text-center text-sm text-[var(--color-text-muted)]">No comments yet.</div>
      )}
      <div className="divide-y divide-[var(--color-border-subtle)]">
        {comments.map((c) => (
          <div key={c.id} className="px-5 py-4">
            <div className="flex items-center gap-2 mb-2">
              <Avatar name={c.created_by} size="sm" />
              <span className="text-sm font-medium text-[var(--color-text-primary)]">{c.created_by.split(" <")[0]}</span>
              <span className="text-xs text-[var(--color-text-muted)]">{formatRelativeTime(c.created_at)}</span>
            </div>
            <div className="prose-beats text-sm pl-8">
              <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>{c.text}</Markdown>
            </div>
          </div>
        ))}
      </div>
      {/* Comment input */}
      <div className="px-5 py-4 border-t border-[var(--color-border-subtle)]">
        <textarea
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder="Leave a comment..."
          rows={3}
          className="w-full text-sm bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] rounded-[var(--radius-md)] border border-[var(--color-border-default)] focus:border-[var(--color-border-focus)] px-3 py-2 outline-none resize-none"
        />
        <div className="flex justify-end mt-2">
          <button onClick={onAddComment} disabled={saving || !newComment.trim()}
            className="px-4 py-1.5 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-40">
            {saving ? "Saving..." : "Comment"}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Diff viewer with line numbers + collapsible cards ─── */
function DiffViewer({ diffs }: { diffs: Map<string, string[]> }) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  const toggle = useCallback((file: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(file)) next.delete(file); else next.add(file);
      return next;
    });
  }, []);

  if (diffs.size === 0) {
    return <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">No changes to display</div>;
  }

  return (
    <div>
      {Array.from(diffs.entries()).map(([file, rawLines]) => {
        const isCollapsed = collapsed.has(file);
        const numbered = isCollapsed ? [] : addLineNumbers(rawLines);
        return (
          <div key={file} className="border-b border-[var(--color-border-default)]">
            <button
              onClick={() => toggle(file)}
              className="sticky top-0 z-10 flex items-center gap-2 w-full px-4 py-2 text-xs font-mono bg-[var(--color-bg-secondary)] border-b border-[var(--color-border-subtle)] hover:bg-[var(--color-bg-hover)] transition-colors text-left"
            >
              {isCollapsed ? <ChevronRight size={13} className="text-[var(--color-text-muted)] shrink-0" /> : <ChevronDown size={13} className="text-[var(--color-text-muted)] shrink-0" />}
              <FileDiff size={13} className="text-[var(--color-text-muted)] shrink-0" />
              <span className="text-[var(--color-text-primary)] font-semibold">{file}</span>
            </button>
            {!isCollapsed && (
              <table className="w-full text-xs font-mono leading-5 border-collapse">
                <tbody>
                  {numbered.map((ln, i) => (
                    <tr key={i} className={
                      ln.type === "add" ? "bg-green-500/10"
                      : ln.type === "del" ? "bg-red-500/10"
                      : ln.type === "hunk" ? "bg-[var(--color-accent-primary)]/5"
                      : ""
                    }>
                      <td className="w-12 text-right pr-2 select-none text-[var(--color-text-muted)]/50 border-r border-[var(--color-border-subtle)]">
                        {ln.oldNum || ""}
                      </td>
                      <td className="w-12 text-right pr-2 select-none text-[var(--color-text-muted)]/50 border-r border-[var(--color-border-subtle)]">
                        {ln.newNum || ""}
                      </td>
                      <td className={`pl-3 pr-4 whitespace-pre ${
                        ln.type === "add" ? "text-green-400"
                        : ln.type === "del" ? "text-red-400"
                        : ln.type === "hunk" ? "text-[var(--color-accent-primary)]"
                        : "text-[var(--color-text-secondary)]"
                      }`}>
                        {ln.text}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        );
      })}
    </div>
  );
}

/* ─── Helpers ─── */

interface NumberedLine {
  oldNum: string;
  newNum: string;
  text: string;
  type: "ctx" | "add" | "del" | "hunk" | "meta";
}

function addLineNumbers(lines: string[]): NumberedLine[] {
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
    } else if (line.startsWith("diff --git") || line.startsWith("index ") || line.startsWith("---") || line.startsWith("+++") || line.startsWith("\\")) {
      result.push({ oldNum: "", newNum: "", text: line, type: "meta" });
    } else if (line.startsWith("+")) {
      result.push({ oldNum: "", newNum: String(newLine), text: line, type: "add" });
      newLine++;
    } else if (line.startsWith("-")) {
      result.push({ oldNum: String(oldLine), newNum: "", text: line, type: "del" });
      oldLine++;
    } else {
      result.push({ oldNum: String(oldLine), newNum: String(newLine), text: line, type: "ctx" });
      oldLine++;
      newLine++;
    }
  }
  return result;
}

function parseDiffByFile(diff: string): Map<string, string[]> {
  const result = new Map<string, string[]>();
  if (!diff) return result;
  const lines = diff.split("\n");
  let currentFile = "";
  let currentLines: string[] = [];
  for (const line of lines) {
    if (line.startsWith("diff --git")) {
      if (currentFile) result.set(currentFile, currentLines);
      const match = line.match(/b\/(.+)$/);
      currentFile = match ? match[1] : "";
      currentLines = [line];
    } else if (currentFile) {
      currentLines.push(line);
    }
  }
  if (currentFile) result.set(currentFile, currentLines);
  return result;
}
