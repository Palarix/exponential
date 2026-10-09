import { useState, useCallback, useMemo, useRef } from "react";
import {
  buildFileTree,
  flattenSingleChildDirs,
  buildSplitLines,
  addLineNumbers,
  parseDiffByFile,
  diffFreshness,
} from "./diff-utils";
import type { FileTreeNode, SplitLine } from "./diff-utils";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkBreaks from "remark-breaks";
import { mergeIssue, addDraft, ApiError } from "../../api/client";
import type { Issue, CommitInfo } from "../../api/client";
import {
  useArtifact,
  useCommitDiff,
  useIssueCommits,
  useIssueDiff,
  useMergeability,
} from "../../api/queries";
import {
  GitCommitVertical,
  GitMerge,
  X,
  Check,
  AlertTriangle,
  FileDiff,
  ChevronDown,
  ChevronRight,
  ChevronsDownUp,
  ChevronsUpDown,
  GitBranch,
  Search,
  BookOpen,
  RefreshCw,
} from "lucide-react";
import { StatusIcon, Avatar, Text, TopBar, ViewContainer } from "../ui";
import Modal from "../ui/Modal";
import { formatRelativeTime } from "../../utils/format";
import { useSpinOnce, useSpinner } from "../../hooks/useSpinner";
import { firstQueryError } from "../../api/query-utils";
import { useToast } from "../ui/ToastContext";

const NO_COMMITS: CommitInfo[] = [];

const FLASH_MS = 1500;

/**
 * Owns its spin state so starting and stopping the spin re-renders only this
 * button, not the whole diff. A click always gets at least one full turn, then
 * briefly shows a check, or an alert plus an error toast; an automatic refresh
 * only spins if it's slow enough to notice, and reports nothing.
 */
function RefreshButton({ onRefresh, autoRefreshing }: { onRefresh: () => Promise<string | null>; autoRefreshing: boolean }) {
  const showToast = useToast();
  const [flash, setFlash] = useState<"success" | "failure" | null>(null);
  const flashTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const { spinning: clickSpin, track } = useSpinOnce<string | null>((result) => {
    const error = result.ok ? result.value : "Failed to load";
    setFlash(error ? "failure" : "success");
    if (error) showToast(`Couldn't refresh: ${error}`, { variant: "error" });
    flashTimer.current = setTimeout(() => setFlash(null), FLASH_MS);
  });
  const autoSpin = useSpinner(autoRefreshing, 150);
  const spinning = clickSpin || autoSpin;

  const handleClick = () => {
    clearTimeout(flashTimer.current);
    setFlash(null);
    track(onRefresh);
  };

  return (
    <button
      onClick={handleClick}
      title={flash === "success" ? "Refreshed" : flash === "failure" ? "Refresh failed" : "Refresh"}
      aria-busy={spinning}
      className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
    >
      {flash === "success" && !spinning ? (
        <Check size={14} className="text-[var(--color-success)]" />
      ) : flash === "failure" && !spinning ? (
        <AlertTriangle size={14} className="text-[var(--color-error)]" />
      ) : (
        <RefreshCw size={14} className={spinning ? "animate-spin" : undefined} />
      )}
    </button>
  );
}

/**
 * Reserves the label's bold width with an invisible `font-medium` copy in the
 * same grid cell, so toggling the weight never resizes (and so never shifts)
 * the centered scope toggle.
 */
function StableLabel({ active, children }: { active: boolean; children: string }) {
  return (
    <span className="inline-grid">
      <span aria-hidden className="col-start-1 row-start-1 invisible font-medium">{children}</span>
      <span className={`col-start-1 row-start-1${active ? " font-medium" : ""}`}>{children}</span>
    </span>
  );
}

function StaleDiffNotice({ asOf }: { asOf: number }) {
  const time = new Date(asOf).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  const text = `Couldn't refresh, showing changes as of ${time}`;
  return (
    <span role="status" title={text} className="mr-2 flex items-center gap-1.5 min-w-0 text-xs text-[var(--color-warning)]">
      <AlertTriangle size={12} className="shrink-0" />
      <span className="truncate">{text}</span>
    </span>
  );
}

interface MergeViewProps {
  issue: Issue;
  onClose: () => void;
  onMerged: () => void;
}

type Tab = "files" | "commits" | "conversation" | "walkthrough";
type DiffScope = "all" | "uncommitted";

export default function MergeView({
  issue,
  onClose,
  onMerged,
}: MergeViewProps) {
  const hasWalkthrough = issue.artifacts?.some(
    (a) => a.artifact_type === "walkthrough",
  );

  const [activeTab, setActiveTab] = useState<Tab>(
    hasWalkthrough ? "walkthrough" : "files",
  );

  // Files tab
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileFilter, setFileFilter] = useState("");
  const [diffScope, setDiffScope] = useState<DiffScope>("all");

  // Commits tab
  const [selectedCommit, setSelectedCommit] = useState<string | null>(null);

  // Conversation tab
  const [newComment, setNewComment] = useState("");
  const [commentSaving, setCommentSaving] = useState(false);

  // Merge
  const [strategyOpen, setStrategyOpen] = useState(false);
  const [mergeStrategy, setMergeStrategy] = useState("squash");
  const [merging, setMerging] = useState(false);
  const [mergeError, setMergeError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [commitMessage, setCommitMessage] = useState("");
  const [deleteBranch, setDeleteBranch] = useState(false);
  const [blockerDetailOpen, setBlockerDetailOpen] = useState(false);

  const bs = issue.branch_stats;

  const commitsQuery = useIssueCommits(issue);
  const diffQuery = useIssueDiff(issue);
  const mergeabilityQuery = useMergeability(issue);
  const walkthroughQuery = useArtifact(issue, "walkthrough.md", !!hasWalkthrough);
  // Polls only while the uncommitted diff is on screen.
  const pollUncommitted = diffScope === "uncommitted" && activeTab === "files";
  const uncommittedQuery = useIssueDiff(issue, "uncommitted", pollUncommitted ? 3000 : false);
  // When the uncommitted view was last entered; data older than this is shown as refreshing.
  const [enteredAt, setEnteredAt] = useState(0);
  const freshness = diffFreshness({
    dataUpdatedAt: uncommittedQuery.dataUpdatedAt,
    errorUpdatedAt: uncommittedQuery.errorUpdatedAt,
    enteredAt,
  });

  // Entering the uncommitted view refetches at once; the 3s poll only covers later changes.
  const showView = (tab: Tab, scope: DiffScope) => {
    if (tab === "files" && scope === "uncommitted" && !pollUncommitted) {
      setEnteredAt(Date.now());
      uncommittedQuery.refetch();
    }
    setActiveTab(tab);
    setDiffScope(scope);
  };
  const commitDiffQuery = useCommitDiff(issue.id, selectedCommit);

  const commits = commitsQuery.data ?? NO_COMMITS;
  const diff = diffQuery.data ?? "";
  const mergeability = mergeabilityQuery.data ?? null;
  const walkthroughContent = walkthroughQuery.data ?? "";
  const uncommittedDiff = uncommittedQuery.data ?? "";
  const uncommittedLoading = uncommittedQuery.isFetching;
  const commitDiffText = commitDiffQuery.data ?? "";
  const commitDiffLoading = commitDiffQuery.isLoading;

  const loading =
    commitsQuery.isPending ||
    diffQuery.isPending ||
    mergeabilityQuery.isPending ||
    (!!hasWalkthrough && walkthroughQuery.isPending) ||
    uncommittedQuery.isPending;
  // A failed poll keeps the last uncommitted diff instead of replacing the view.
  const error = firstQueryError([
    commitsQuery,
    diffQuery,
    mergeabilityQuery,
    !!hasWalkthrough && walkthroughQuery,
    uncommittedQuery.data === undefined && uncommittedQuery,
  ]);

  /** Refetches everything; resolves to the first failure's message, or null. */
  const refreshData = () =>
    Promise.all([
      commitsQuery.refetch(),
      diffQuery.refetch(),
      mergeabilityQuery.refetch(),
      !!hasWalkthrough && walkthroughQuery.refetch(),
      uncommittedQuery.refetch(),
    ]).then(firstQueryError);

  const loadCommitDiff = useCallback((sha: string) => {
    setSelectedCommit((prev) => (prev === sha ? null : sha));
  }, []);

  const handleAddComment = useCallback(async () => {
    if (!newComment.trim()) return;
    setCommentSaving(true);
    try {
      await addDraft(issue.id, "COMMENT", {
        id: `cmt-${crypto.randomUUID()}`,
        text: newComment,
      });
      setNewComment("");
    } catch (err) {
      console.error(err);
    } finally {
      setCommentSaving(false);
    }
  }, [issue.id, newComment]);

  const fileDiffs = useMemo(() => parseDiffByFile(diff), [diff]);
  const uncommittedFileDiffs = useMemo(
    () => parseDiffByFile(uncommittedDiff),
    [uncommittedDiff],
  );
  const activeFileDiffs =
    diffScope === "uncommitted"
      ? uncommittedFileDiffs
      : fileDiffs.size > 0
        ? fileDiffs
        : uncommittedFileDiffs;
  const dirtyFilesSet = useMemo(
    () => new Set(mergeability?.dirty_files ?? []),
    [mergeability],
  );
  const commitFileDiffs = useMemo(
    () => parseDiffByFile(commitDiffText),
    [commitDiffText],
  );

  const handleScopeChange = (scope: DiffScope) => {
    showView(activeTab, scope);
    setSelectedFile(null);
  };

  const defaultCommitMessage = useCallback(
    (strategy: string) => {
      const commitList =
        commits.length > 0
          ? "\n\n" + commits.map((c) => `* ${c.message}`).join("\n")
          : "";
      switch (strategy) {
        case "squash":
          return `${issue.id}: ${issue.title}${commitList}`;
        case "ff":
          return `xpo: merge ${issue.id}`;
        default:
          return `Merge branch '${issue.branch_stats?.branch ?? issue.id}'${commitList}`;
      }
    },
    [issue, commits],
  );

  const openConfirm = useCallback(() => {
    setCommitMessage(defaultCommitMessage(mergeStrategy));
    setConfirmOpen(true);
  }, [mergeStrategy, defaultCommitMessage]);

  const handleMerge = useCallback(async () => {
    setMerging(true);
    setMergeError(null);
    try {
      await mergeIssue(issue.id, {
        strategy: mergeStrategy,
        commit_message: commitMessage,
        delete_branch: deleteBranch,
      });
      setConfirmOpen(false);
      onMerged();
    } catch (err) {
      setMergeError(err instanceof ApiError ? err.message : "Merge failed");
    } finally {
      setMerging(false);
    }
  }, [issue.id, mergeStrategy, commitMessage, deleteBranch, onMerged]);

  const strategyLabel =
    mergeStrategy === "squash"
      ? "Squash and merge"
      : mergeStrategy === "merge"
        ? "Create merge commit"
        : "Fast-forward";
  const hasCommits = (bs?.commits ?? 0) > 0;
  const canMerge = hasCommits && (mergeability?.can_merge ?? true);

  if (loading)
    return (
      <div className="h-full flex items-center justify-center text-sm text-[var(--color-text-muted)]">
        Loading...
      </div>
    );
  if (error)
    return (
      <div className="h-full flex flex-col items-center justify-center gap-3">
        <Text color="error">{error}</Text>
        <button
          onClick={onClose}
          className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
        >
          Go back
        </button>
      </div>
    );

  if (!bs)
    return (
      <div className="h-full flex flex-col items-center justify-center gap-4">
        <div className="flex items-center gap-2 text-green-500">
          <GitMerge size={24} />
          <span className="text-sm font-medium">Branch merged</span>
        </div>
        <Text as="p">
          This branch has been merged and is no longer available.
        </Text>
        <button
          onClick={onClose}
          className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
        >
          Close
        </button>
      </div>
    );

  return (
    <ViewContainer
      scroll={false}
      className="bg-[var(--color-surface)]"
      topBar={
        <TopBar
          className="bg-[var(--color-surface-1)] border-[var(--color-border-default)]"
          left={
            <div className="flex items-center gap-2">
              <StatusIcon
                status={issue.status}
                size={16}
                isInferred={issue.is_inferred}
              />
              <Text weight="semibold" color="primary">
                {issue.title}
              </Text>
              <Text mono>
                {issue.id}
              </Text>
            </div>
          }
          right={
            <button
              onClick={onClose}
              className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
            >
              <X size={20} />
            </button>
          }
        />
      }
      header={
        <>
          {/* ── Merge bar (3-column) ── */}
          <div className="shrink-0 border-b border-[var(--color-border-default)] bg-[var(--color-surface-1)] px-5 py-3">
            <div className="grid grid-cols-3 items-center gap-3">
              {/* Left: branch direction */}
              <div className="flex items-center gap-2 text-sm">
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[var(--color-surface-1)] text-[var(--color-text-secondary)] font-mono text-xs">
                  <GitBranch size={12} />
                  main
                </span>
                <span className="text-[var(--color-text-muted)]">←</span>
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-[var(--color-accent-primary)]/10 text-[var(--color-accent-primary)] font-mono text-xs truncate">
                  <GitBranch size={12} className="shrink-0" />
                  <span className="truncate">{bs.branch}</span>
                </span>
              </div>

              {/* Center: mergeability */}
              <div className="relative flex flex-col items-center">
                {!canMerge ? (
                  <button
                    type="button"
                    onClick={() => {
                      if (mergeability?.blockers?.some((b) => b.files?.length))
                        setBlockerDetailOpen(!blockerDetailOpen);
                    }}
                    className="flex items-center gap-2 text-sm text-amber-500 hover:text-amber-400 transition-colors"
                  >
                    <AlertTriangle size={16} className="shrink-0" />
                    <span>
                      {!hasCommits
                        ? "No commits to merge"
                        : mergeability?.blockers?.[0]?.message || "Cannot merge"}
                    </span>
                  </button>
                ) : mergeError ? (
                  <div className="flex items-center gap-2 text-sm text-[var(--color-error)]">
                    <AlertTriangle size={16} />
                    <span>{mergeError}</span>
                  </div>
                ) : (
                  <div className="flex items-center gap-1.5 text-sm text-green-500">
                    <Check size={16} />
                    Ready to merge
                  </div>
                )}
                <span className="text-xs text-[var(--color-text-muted)] mt-0.5">
                  {!canMerge && !hasCommits && bs.has_uncommitted
                    ? "Commit your changes to enable merging"
                    : !canMerge && hasCommits
                      ? "Resolve blockers to enable merging"
                      : hasCommits
                        ? "Merging will close this issue"
                        : "Nothing to merge yet"}
                </span>
                <Modal
                  isOpen={blockerDetailOpen}
                  onClose={() => setBlockerDetailOpen(false)}
                  title="Unable to merge"
                  size="xl"
                >
                  <p className="text-sm text-[var(--color-text-secondary)] mb-4">
                    This branch cannot be merged because the worktree contains
                    uncommitted or untracked files that would be lost.
                  </p>
                  {mergeability?.blockers?.map((b, i) => (
                    <div key={i} className={i > 0 ? "mt-4 pt-4 border-t border-[var(--color-border-subtle)]" : ""}>
                      <div className="flex items-center gap-2 text-sm font-medium text-amber-500 mb-3">
                        <AlertTriangle size={14} className="shrink-0" />
                        {b.message}
                      </div>
                      {b.files?.length ? (
                        <ul className="space-y-2 max-h-54 overflow-y-auto rounded-lg bg-[var(--color-surface-1)] p-3">
                          {b.files.map((f) => (
                            <li
                              key={f}
                              className="text-sm font-mono text-[var(--color-text-muted)]"
                            >
                              {f}
                            </li>
                          ))}
                        </ul>
                      ) : null}
                    </div>
                  ))}
                  <p className="mt-5 text-sm text-[var(--color-text-muted)]">
                    Commit, move, or remove these files before merging.
                  </p>
                </Modal>
              </div>

              {/* Right: merge button + strategy */}
              <div className="flex justify-end items-center">
                <button
                  onClick={openConfirm}
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
                      <div
                        className="fixed inset-0 z-40"
                        onClick={() => setStrategyOpen(false)}
                      />
                      <div className="absolute right-0 top-full mt-1 z-50 w-56 rounded-[var(--radius-md)] bg-[var(--color-surface-3)] border border-[var(--color-border-default)] shadow-[var(--shadow-lg)] py-1">
                        {(
                          [
                            {
                              key: "squash",
                              label: "Squash and merge",
                              desc: "Single commit on main",
                            },
                            {
                              key: "merge",
                              label: "Create merge commit",
                              desc: "Preserves branch history",
                            },
                            {
                              key: "ff",
                              label: "Fast-forward",
                              desc: "Linear, no merge commit",
                            },
                          ] as const
                        ).map((opt) => (
                          <button
                            key={opt.key}
                            onClick={() => {
                              setMergeStrategy(opt.key);
                              setCommitMessage(defaultCommitMessage(opt.key));
                              setStrategyOpen(false);
                            }}
                            className={`w-full px-3 py-2 text-left hover:bg-[var(--color-hover-surface)] transition-colors ${mergeStrategy === opt.key ? "text-[var(--color-accent-primary)]" : ""}`}
                          >
                            <div className="text-sm">{opt.label}</div>
                            <div className="text-xs text-[var(--color-text-muted)]">
                              {opt.desc}
                            </div>
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
          <div className="shrink-0 flex items-center gap-1 px-5 border-b border-[var(--color-border-subtle)] relative">
            {hasWalkthrough && (
              <button
                onClick={() => showView("walkthrough", diffScope)}
                className={`px-3 py-2.5 text-sm font-medium transition-colors relative ${activeTab === "walkthrough" ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"}`}
              >
                <span className="inline-flex items-center gap-1.5">
                  <BookOpen size={14} />
                  Walkthrough
                </span>
                {activeTab === "walkthrough" && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--color-accent-primary)]" />
                )}
              </button>
            )}
            {[
              { key: "commits" as Tab, label: "Commits", count: commits.length },
              { key: "files" as Tab, label: "Files changed", count: activeFileDiffs.size },
              {
                key: "conversation" as Tab,
                label: "Conversation",
                count: issue.comments?.length || 0,
              },
            ].map((t) => (
              <button
                key={t.key}
                onClick={() => showView(t.key, diffScope)}
                className={`px-3 py-2.5 text-sm font-medium transition-colors relative ${activeTab === t.key ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"}`}
              >
                {t.label}
                <span className="ml-1.5 text-xs tabular-nums px-1.5 py-0.5 rounded-full bg-[var(--color-surface-1)]">
                  {t.count}
                </span>
                {activeTab === t.key && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--color-accent-primary)]" />
                )}
              </button>
            ))}
            <div className="flex-1" />
            {activeTab === "files" && (
              <div className="absolute left-1/2 -translate-x-1/2 flex items-center gap-2">
                <div className="inline-flex rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
                  <button
                    onClick={() => handleScopeChange("all")}
                    className={`px-3 py-1 text-sm transition-colors ${
                      diffScope === "all"
                        ? "bg-[var(--color-surface-2)] text-[var(--color-text-primary)]"
                        : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                    }`}
                  >
                    <StableLabel active={diffScope === "all"}>All changes</StableLabel>
                  </button>
                  <button
                    onClick={() => handleScopeChange("uncommitted")}
                    className={`px-3 py-1 text-sm border-l border-[var(--color-border-default)] transition-colors ${
                      diffScope === "uncommitted"
                        ? "bg-[var(--color-surface-2)] text-[var(--color-text-primary)]"
                        : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                    }`}
                  >
                    <StableLabel active={diffScope === "uncommitted"}>Uncommitted</StableLabel>
                  </button>
                </div>
              </div>
            )}
            {pollUncommitted && freshness === "failed" && (
              <StaleDiffNotice asOf={uncommittedQuery.dataUpdatedAt} />
            )}
            <RefreshButton
              onRefresh={refreshData}
              autoRefreshing={pollUncommitted && freshness === "refreshing"}
            />
          </div>
        </>
      }
    >
      {activeTab === "walkthrough" && (
        <div className="h-full scroll-stable px-8 py-6">
          <div className="prose-exponential text-sm max-w-[52rem] mx-auto">
            <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
              {walkthroughContent}
            </Markdown>
          </div>
        </div>
      )}
      {activeTab === "commits" && (
        <CommitsTab
          commits={commits}
          selectedCommit={selectedCommit}
          commitDiffs={commitFileDiffs}
          commitDiffLoading={commitDiffLoading}
          onSelectCommit={loadCommitDiff}
        />
      )}
      {activeTab === "files" && (
        <FilesTab
          fileDiffs={activeFileDiffs}
          selectedFile={selectedFile}
          onSelectFile={setSelectedFile}
          fileFilter={fileFilter}
          onFilterChange={setFileFilter}
          dirtyFiles={dirtyFilesSet}
          isEmpty={diffScope === "uncommitted" && activeFileDiffs.size === 0 && !uncommittedLoading}
        />
      )}
      {activeTab === "conversation" && (
        <ConversationTab
          issue={issue}
          newComment={newComment}
          onNewCommentChange={setNewComment}
          onAddComment={handleAddComment}
          saving={commentSaving}
        />
      )}

    {/* ── Merge confirmation dialog ── */}
    {confirmOpen && (
      <>
        <div
          className="fixed inset-0 z-50 bg-black/50"
          onClick={() => setConfirmOpen(false)}
        />
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="w-full max-w-2xl rounded-[var(--radius-lg)] bg-[var(--color-surface-3)] border border-[var(--color-border-default)] shadow-[var(--shadow-xl)]">
            <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border-subtle)]">
              <div className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text-primary)]">
                <GitMerge
                  size={16}
                  className="text-[var(--color-accent-primary)]"
                />
                Confirm merge
              </div>
              <button
                onClick={() => setConfirmOpen(false)}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)]"
              >
                <X size={16} />
              </button>
            </div>
            <div className="px-5 py-4 space-y-4">
              <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
                <Text size="xs">
                  Strategy:
                </Text>
                <span className="font-medium">{strategyLabel}</span>
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">
                {mergeStrategy === "squash"
                  ? "All commits will be combined into a single commit on main. The branch history is not preserved."
                  : mergeStrategy === "merge"
                    ? "A merge commit will be created on main. The full branch commit history is preserved."
                    : "Main will be fast-forwarded to the branch tip. No merge commit is created. Only possible if main has no new commits since the branch was created."}{" "}
                The issue will be marked as done.
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--color-text-muted)] mb-1.5">
                  Commit message
                </label>
                <textarea
                  value={commitMessage}
                  onChange={(e) => setCommitMessage(e.target.value)}
                  rows={6}
                  className="w-full text-sm font-mono bg-[var(--color-surface-1)] text-[var(--color-text-primary)] rounded-[var(--radius-md)] border border-[var(--color-border-default)] focus:border-[var(--color-border-focus)] px-3 py-2 outline-none resize-none"
                />
              </div>
              {mergeError && (
                <div className="text-sm text-[var(--color-error)]">
                  {mergeError}
                </div>
              )}
            </div>
            <div className="flex items-center px-5 py-3 border-t border-[var(--color-border-subtle)]">
              <label className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)] cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={deleteBranch}
                  onChange={(e) => setDeleteBranch(e.target.checked)}
                  className="h-4 w-4 rounded border-[var(--color-border-default)] shrink-0"
                />
                <span className="leading-none">
                  Delete branch after merge
                </span>
              </label>
              <div className="ml-auto flex items-center gap-2">
                <button
                  onClick={() => setConfirmOpen(false)}
                  className="px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleMerge}
                  disabled={merging || !commitMessage.trim()}
                  className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
                >
                  <GitMerge size={14} />
                  {merging ? "Merging..." : "Merge and close issue"}
                </button>
              </div>
            </div>
          </div>
        </div>
      </>
    )}
    </ViewContainer>
  );
}

/* ─── Commits Tab: list left, diff right ─── */
function CommitsTab({
  commits,
  selectedCommit,
  commitDiffs,
  commitDiffLoading,
  onSelectCommit,
}: {
  commits: CommitInfo[];
  selectedCommit: string | null;
  commitDiffs: Map<string, string[]>;
  commitDiffLoading: boolean;
  onSelectCommit: (sha: string) => void;
}) {
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  if (commits.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
        <div className="text-center">
          <GitCommitVertical size={24} className="mx-auto mb-2 opacity-40" />
          <p>No commits yet</p>
          <p className="text-xs mt-1">
            Uncommitted changes are shown in the Files tab
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex">
      <div className="w-80 shrink-0 border-r border-[var(--color-border-default)] scroll-stable bg-[var(--color-surface-1)]">
        {commits.map((c) => (
          <button
            key={c.sha}
            onClick={() => onSelectCommit(c.sha)}
            className={`flex items-start gap-3 w-full px-4 py-3 text-left border-b border-[var(--color-border-subtle)] transition-colors ${selectedCommit === c.sha ? "bg-[var(--color-accent-primary)]/5" : "hover:bg-[var(--color-hover-surface)]"}`}
          >
            <GitCommitVertical
              size={16}
              className="text-[var(--color-text-muted)] shrink-0 mt-0.5"
            />
            <div className="min-w-0 flex-1">
              <div className="text-sm text-[var(--color-text-primary)] leading-snug">
                {c.message}
              </div>
              <div className="text-xs text-[var(--color-text-muted)] mt-1">
                <span className="font-mono">{c.sha}</span> · {c.author} ·{" "}
                {formatRelativeTime(c.date)}
              </div>
            </div>
          </button>
        ))}
      </div>
      <div className="flex-1 scroll-stable">
        {!selectedCommit && (
          <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
            Select a commit to view its changes
          </div>
        )}
        {selectedCommit && commitDiffLoading && (
          <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
            Loading...
          </div>
        )}
        {selectedCommit && !commitDiffLoading && (
          <DiffViewer
            diffs={commitDiffs}
            collapsed={collapsed}
            setCollapsed={setCollapsed}
          />
        )}
      </div>
    </div>
  );
}

/* ─── File tree ─── */

function FileTreeView({
  nodes,
  depth,
  selectedFile,
  onSelectFile,
  collapsedDirs,
  onToggleDir,
  fileDiffs,
  dirtyFiles,
}: {
  nodes: FileTreeNode[];
  depth: number;
  selectedFile: string | null;
  onSelectFile: (f: string | null) => void;
  collapsedDirs: Set<string>;
  onToggleDir: (path: string) => void;
  fileDiffs: Map<string, string[]>;
  dirtyFiles: Set<string>;
}) {
  return (
    <>
      {nodes.map((node) => {
        const isCollapsed = collapsedDirs.has(node.path);
        if (!node.isFile) {
          return (
            <div key={node.path}>
              <button
                onClick={() => onToggleDir(node.path)}
                className="flex items-center gap-1.5 w-full py-1.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors text-left"
                style={{ paddingLeft: `${12 + depth * 12}px` }}
              >
                {isCollapsed ? (
                  <ChevronRight size={12} className="shrink-0" />
                ) : (
                  <ChevronDown size={12} className="shrink-0" />
                )}
                <span className="font-mono truncate">{node.name}/</span>
              </button>
              {!isCollapsed && (
                <FileTreeView
                  nodes={node.children}
                  depth={depth + 1}
                  selectedFile={selectedFile}
                  onSelectFile={onSelectFile}
                  collapsedDirs={collapsedDirs}
                  onToggleDir={onToggleDir}
                  fileDiffs={fileDiffs}
                  dirtyFiles={dirtyFiles}
                />
              )}
            </div>
          );
        }
        const isDirty = dirtyFiles.has(node.path);
        const diffLines = fileDiffs.get(node.path);
        const isNewFile = diffLines?.some((l) => l.startsWith("new file"));
        const isDeleted = diffLines?.some((l) =>
          l.startsWith("deleted file"),
        );
        const statusColor = isDirty
          ? "text-amber-500"
          : isNewFile
            ? "text-green-500"
            : isDeleted
              ? "text-red-500"
              : "text-[var(--color-text-muted)]";
        return (
          <button
            key={node.path}
            onClick={() =>
              onSelectFile(selectedFile === node.path ? null : node.path)
            }
            className={`flex items-center gap-1.5 w-full py-1.5 pr-3 text-xs text-left transition-colors ${selectedFile === node.path ? "bg-[var(--color-accent-primary)]/5 text-[var(--color-accent-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"}`}
            style={{ paddingLeft: `${12 + depth * 12}px` }}
            title={isDirty ? "Uncommitted changes" : undefined}
          >
            <FileDiff size={12} className={`shrink-0 ${statusColor}`} />
            <span className="font-mono truncate">{node.name}</span>
          </button>
        );
      })}
    </>
  );
}

/* ─── Files Tab: file tree left, diff right ─── */
function FilesTab({
  fileDiffs,
  selectedFile,
  onSelectFile,
  fileFilter,
  onFilterChange,
  dirtyFiles,
  isEmpty,
}: {
  fileDiffs: Map<string, string[]>;
  selectedFile: string | null;
  onSelectFile: (f: string | null) => void;
  fileFilter: string;
  onFilterChange: (v: string) => void;
  dirtyFiles: Set<string>;
  isEmpty: boolean;
}) {
  const [collapsedDirs, setCollapsedDirs] = useState<Set<string>>(new Set());
  const [collapsedCards, setCollapsedCards] = useState<Set<string>>(new Set());

  const toggleDir = useCallback((path: string) => {
    setCollapsedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  const diffPaths = useMemo(() => Array.from(fileDiffs.keys()), [fileDiffs]);
  const filteredPaths = useMemo(() => {
    if (!fileFilter) return diffPaths;
    const q = fileFilter.toLowerCase();
    return diffPaths.filter((p) => p.toLowerCase().includes(q));
  }, [diffPaths, fileFilter]);
  const tree = useMemo(
    () => flattenSingleChildDirs(buildFileTree(filteredPaths)),
    [filteredPaths],
  );

  const diffRef = useRef<HTMLDivElement>(null);

  const handleSelectFile = useCallback(
    (file: string | null) => {
      onSelectFile(file);
      if (file) {
        setCollapsedCards((prev) => {
          if (!prev.has(file)) return prev;
          const next = new Set(prev);
          next.delete(file);
          return next;
        });
        requestAnimationFrame(() => {
          if (!diffRef.current) return;
          const el = diffRef.current.querySelector(
            `[data-diff-file="${CSS.escape(file)}"]`,
          ) as HTMLElement | null;
          if (el) {
            const container = diffRef.current;
            const offset = el.offsetTop - container.offsetTop - 50;
            container.scrollTo({ top: offset, behavior: "smooth" });
          }
        });
      }
    },
    [onSelectFile],
  );

  return (
    <div className="h-full flex">
      <div className="w-72 shrink-0 border-r border-[var(--color-border-default)] scroll-stable bg-[var(--color-surface-1)]">
        <div className="p-3 border-b border-[var(--color-border-subtle)]">
          <div className="flex items-center gap-2 px-2.5 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)]">
            <Search size={13} className="text-[var(--color-text-muted)]" />
            <input
              value={fileFilter}
              onChange={(e) => onFilterChange(e.target.value)}
              placeholder="Filter files..."
              className="flex-1 text-xs bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
            />
          </div>
        </div>
        {!isEmpty && (
          <div className="py-1">
            <FileTreeView
              nodes={tree}
              depth={0}
              selectedFile={selectedFile}
              onSelectFile={handleSelectFile}
              collapsedDirs={collapsedDirs}
              onToggleDir={toggleDir}
              fileDiffs={fileDiffs}
              dirtyFiles={dirtyFiles}
            />
          </div>
        )}
      </div>
      {isEmpty ? (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-[var(--color-text-muted)]">
            No uncommitted changes
          </p>
        </div>
      ) : (
        <div ref={diffRef} className="flex-1 scroll-stable">
          <DiffViewer
            diffs={fileDiffs}
            collapsed={collapsedCards}
            setCollapsed={setCollapsedCards}
          />
        </div>
      )}
    </div>
  );
}

/* ─── Conversation Tab ─── */
function ConversationTab({
  issue,
  newComment,
  onNewCommentChange,
  onAddComment,
  saving,
}: {
  issue: Issue;
  newComment: string;
  onNewCommentChange: (v: string) => void;
  onAddComment: () => void;
  saving: boolean;
}) {
  const comments = issue.comments || [];
  return (
    <div className="h-full scroll-stable">
      <div className="max-w-[52rem] mx-auto">
        {comments.length === 0 && !newComment && (
          <div className="px-5 py-12 text-center text-sm text-[var(--color-text-muted)]">
            No comments yet.
          </div>
        )}
        <div className="divide-y divide-[var(--color-border-subtle)]">
          {comments.map((c) => (
            <div key={c.id} className="px-5 py-4">
              <div className="flex items-center gap-2 mb-2">
                <Avatar name={c.created_by} size="sm" />
                <Text weight="medium" color="primary">{c.created_by.split(" <")[0]}</Text>
                <Text size="xs">
                  {formatRelativeTime(c.created_at)}
                </Text>
              </div>
              <div className="prose-exponential text-sm pl-8">
                <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                  {c.text}
                </Markdown>
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
            className="w-full text-sm bg-[var(--color-surface-1)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] rounded-[var(--radius-md)] border border-[var(--color-border-default)] focus:border-[var(--color-border-focus)] px-3 py-2 outline-none resize-none"
          />
          <div className="flex justify-end mt-2">
            <button
              onClick={onAddComment}
              disabled={saving || !newComment.trim()}
              className="px-4 py-1.5 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
            >
              {saving ? "Saving..." : "Comment"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ─── Diff viewer with line numbers + collapsible cards ─── */
type DiffMode = "unified" | "split";

function DiffViewer({
  diffs,
  collapsed,
  setCollapsed,
}: {
  diffs: Map<string, string[]>;
  collapsed: Set<string>;
  setCollapsed: React.Dispatch<React.SetStateAction<Set<string>>>;
}) {
  const [mode, setMode] = useState<DiffMode>(
    () =>
      (localStorage.getItem("exponential-diff-mode") as DiffMode) || "unified",
  );

  const toggleFile = useCallback(
    (file: string) => {
      setCollapsed((prev) => {
        const next = new Set(prev);
        if (next.has(file)) next.delete(file);
        else next.add(file);
        return next;
      });
    },
    [setCollapsed],
  );

  const setDiffMode = useCallback((m: DiffMode) => {
    setMode(m);
    localStorage.setItem("exponential-diff-mode", m);
  }, []);

  const expandAll = useCallback(() => setCollapsed(new Set()), [setCollapsed]);
  const collapseAll = useCallback(
    () => setCollapsed(new Set(diffs.keys())),
    [diffs, setCollapsed],
  );

  if (diffs.size === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
        No changes to display
      </div>
    );
  }

  return (
    <div>
      {/* Toolbar */}
      <div className="sticky top-0 z-20 flex items-center justify-end gap-2 px-4 py-1.5 bg-[var(--color-surface)] border-b border-[var(--color-border-subtle)]">
        <div className="inline-flex rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
          <button
            onClick={expandAll}
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          >
            <ChevronsUpDown size={13} />
            Expand all
          </button>
          <button
            onClick={collapseAll}
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs border-l border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          >
            <ChevronsDownUp size={13} />
            Collapse all
          </button>
        </div>
        <div className="inline-flex rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
          <button
            onClick={() => setDiffMode("unified")}
            className={`px-2.5 py-1 text-xs transition-colors ${mode === "unified" ? "bg-[var(--color-surface-2)] text-[var(--color-text-primary)] font-medium" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"}`}
          >
            Unified
          </button>
          <button
            onClick={() => setDiffMode("split")}
            className={`px-2.5 py-1 text-xs border-l border-[var(--color-border-default)] transition-colors ${mode === "split" ? "bg-[var(--color-surface-2)] text-[var(--color-text-primary)] font-medium" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"}`}
          >
            Split
          </button>
        </div>
      </div>

      <div className="p-4 space-y-4">
        {Array.from(diffs.entries()).map(([file, rawLines]) => {
          const isCollapsed = collapsed.has(file);
          return (
            <div
              key={file}
              data-diff-file={file}
              className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden"
            >
              <button
                onClick={() => toggleFile(file)}
                className="flex items-center gap-2 w-full px-4 py-2 text-xs font-mono bg-[var(--color-surface-1)] border-b border-[var(--color-border-subtle)] hover:bg-[var(--color-hover-surface)] transition-colors text-left"
              >
                {isCollapsed ? (
                  <ChevronRight
                    size={13}
                    className="text-[var(--color-text-muted)] shrink-0"
                  />
                ) : (
                  <ChevronDown
                    size={13}
                    className="text-[var(--color-text-muted)] shrink-0"
                  />
                )}
                <FileDiff
                  size={13}
                  className="text-[var(--color-text-muted)] shrink-0"
                />
                <span className="text-[var(--color-text-primary)] font-semibold flex-1">
                  {file}
                </span>
                {(() => {
                  const add = rawLines.filter(
                    (l) => l.startsWith("+") && !l.startsWith("+++"),
                  ).length;
                  const del = rawLines.filter(
                    (l) => l.startsWith("-") && !l.startsWith("---"),
                  ).length;
                  if (add > 0 || del > 0) {
                    return (
                      <span className="text-[var(--color-text-muted)] tabular-nums shrink-0">
                        <span className="text-green-500">+{add}</span>{" "}
                        <span className="text-red-500">-{del}</span>
                      </span>
                    );
                  }
                  return null;
                })()}
              </button>
              {!isCollapsed && (
                <div className="overflow-x-auto">
                  {mode === "unified" ? (
                    <UnifiedDiff lines={rawLines} />
                  ) : (
                    <SplitDiff lines={rawLines} />
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function rowBg(type: string): string {
  if (type === "add") return "bg-green-500/10";
  if (type === "del") return "bg-red-500/10";
  if (type === "hunk") return "bg-[var(--color-accent-primary)]/5";
  return "";
}

function gutterBg(type: string): string {
  if (type === "add") return "bg-[var(--color-diff-gutter-add)]";
  if (type === "del") return "bg-[var(--color-diff-gutter-del)]";
  return "bg-[var(--color-surface)]";
}

function UnifiedDiff({ lines }: { lines: string[] }) {
  const numbered = addLineNumbers(lines);
  return (
    <table
      className="text-xs font-mono leading-5 border-collapse"
      style={{ minWidth: "100%" }}
    >
      <tbody>
        {numbered
          .filter((ln) => ln.type !== "meta")
          .map((ln, i) => (
            <tr key={i} className={rowBg(ln.type)}>
              {ln.type === "hunk" ? (
                <>
                  <td
                    className="sticky left-0 z-[1] w-[50px] min-w-[50px] max-w-[50px] bg-[var(--color-accent-primary)]/10"
                    style={{ boxShadow: "1px 0 0 var(--color-border-subtle)" }}
                  />
                  <td
                    className="sticky left-[50px] z-[1] w-[50px] min-w-[50px] max-w-[50px] bg-[var(--color-accent-primary)]/10"
                    style={{ boxShadow: "1px 0 0 var(--color-border-subtle)" }}
                  />
                  <td className="bg-[var(--color-accent-primary)]/5 text-[var(--color-accent-primary)] whitespace-pre">
                    <span className="sticky left-[100px] pl-3 pr-4">
                      {ln.text}
                    </span>
                  </td>
                </>
              ) : (
                <>
                  <td
                    className={`sticky left-0 z-[1] w-[50px] min-w-[50px] max-w-[50px] text-right pr-2 select-none text-[var(--color-text-muted)]/50 ${gutterBg(ln.type)}`}
                    style={{ boxShadow: "1px 0 0 var(--color-border-subtle)" }}
                  >
                    {ln.oldNum || ""}
                  </td>
                  <td
                    className={`sticky left-[50px] z-[1] w-[50px] min-w-[50px] max-w-[50px] text-right pr-2 select-none text-[var(--color-text-muted)]/50 ${gutterBg(ln.type)}`}
                    style={{ boxShadow: "1px 0 0 var(--color-border-subtle)" }}
                  >
                    {ln.newNum || ""}
                  </td>
                  <td
                    className={`pl-3 pr-4 whitespace-pre ${
                      ln.type === "add"
                        ? "text-green-400"
                        : ln.type === "del"
                          ? "text-red-400"
                          : "text-[var(--color-text-secondary)]"
                    }`}
                  >
                    {ln.text}
                  </td>
                </>
              )}
            </tr>
          ))}
      </tbody>
    </table>
  );
}

function splitCellBg(type: string): string {
  if (type === "del") return "bg-red-500/10";
  if (type === "add") return "bg-green-500/10";
  if (type === "empty") return "bg-[var(--color-surface-1)]";
  return "";
}

function SplitDiff({ lines }: { lines: string[] }) {
  const chunks = useMemo(() => buildSplitLines(lines), [lines]);

  const rows: { left: SplitLine; right: SplitLine; isHunk: boolean }[] = [];
  for (const chunk of chunks) {
    for (let ri = 0; ri < chunk.left.length; ri++) {
      const left = chunk.left[ri];
      const right = chunk.right[ri] || {
        num: "",
        text: "",
        type: "empty" as const,
      };
      const isHunk = left.type === "ctx" && left.text.startsWith("@@");
      rows.push({ left, right, isHunk });
    }
  }

  return (
    <table
      className="w-full text-xs font-mono leading-5 border-collapse"
      style={{ tableLayout: "fixed" }}
    >
      <colgroup>
        <col style={{ width: 50 }} />
        <col />
        <col style={{ width: 50 }} />
        <col />
      </colgroup>
      <tbody>
        {rows.map((row, i) => {
          if (row.isHunk) {
            return (
              <tr key={i}>
                <td className="bg-[var(--color-accent-primary)]/10 border-r border-[var(--color-border-subtle)]" />
                <td className="bg-[var(--color-accent-primary)]/5 text-[var(--color-accent-primary)] pl-3 pr-4 whitespace-pre truncate">
                  {row.left.text}
                </td>
                <td className="bg-[var(--color-accent-primary)]/10 border-l border-r border-[var(--color-border-subtle)]" />
                <td className="bg-[var(--color-accent-primary)]/5 text-[var(--color-accent-primary)] pl-3 pr-4 whitespace-pre truncate">
                  {row.right.text}
                </td>
              </tr>
            );
          }
          return (
            <tr key={i}>
              <td
                className={`text-right pr-2 select-none text-[var(--color-text-muted)]/50 border-r border-[var(--color-border-subtle)] ${splitCellBg(row.left.type)}`}
              >
                {row.left.num}
              </td>
              <td
                className={`whitespace-pre truncate pl-3 pr-4 ${splitCellBg(row.left.type)} ${
                  row.left.type === "del"
                    ? "text-red-400"
                    : row.left.type === "empty"
                      ? ""
                      : "text-[var(--color-text-secondary)]"
                }`}
              >
                {row.left.text}
              </td>
              <td
                className={`text-right pr-2 select-none text-[var(--color-text-muted)]/50 border-l border-r border-[var(--color-border-subtle)] ${splitCellBg(row.right.type)}`}
              >
                {row.right.num}
              </td>
              <td
                className={`whitespace-pre truncate pl-3 pr-4 ${splitCellBg(row.right.type)} ${
                  row.right.type === "add"
                    ? "text-green-400"
                    : row.right.type === "empty"
                      ? ""
                      : "text-[var(--color-text-secondary)]"
                }`}
              >
                {row.right.text}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
