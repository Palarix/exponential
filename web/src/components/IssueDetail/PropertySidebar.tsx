import { useState, useMemo, useCallback, useContext, useRef } from "react";
import type { Issue } from "../../api/client";
import { useCycles, useReleases } from "../../api/queries";
import { useDeleteIssue, type IssuePatch } from "../../api/mutations";
import { useStartWorkChat } from "../Agent/useWorkMode";
import { IssueChatsPopover, IssueChatStateIcon } from "../Agent/IssueChats";
import { noGitHint } from "../Agent/issue-chats";
import { chatCount, useIssueChats } from "../Agent/useIssueChats";
import { ActorName, Avatar, Button, DeleteIssueDialog, LabelBadge, DefaultLabelsContext, Modal, StatusIcon, Popover, PopoverPanel, LabelPicker, CyclePicker, StatusPicker, EstimatePicker, PriorityPicker, AssigneePicker, Text } from "../ui";
import { Flag, UserRound, RefreshCw, Trash2, FlagOff, Package } from "lucide-react";
import { useReleaseAssign } from "../Releases/release-assign-context";
import { reopenHint } from "../Releases/release-assign-utils";
import ReleasePickerPopover from "../Releases/ReleasePickerPopover";
import Tooltip from "../ui/Tooltip";
import { GitBranch, GitMerge } from "lucide-react";
import { cn } from "../../utils/cn";
import { formatRelativeTime } from "../../utils/format";
import { useAllLabels } from "../../hooks/useLabels";
import { useScrollFade } from "../../hooks/useScrollFade";
import { toggleLabel, splitLabels } from "../../utils/labels";
import { isTerminal } from "../../constants";
import { collectKnownPeople } from "../../utils/issues";
import { PriorityIcon } from "./icons";
import { RelationsAddPopover, RelationsCard } from "./Relations";
import { relationshipsLabel, removeLink, unresolvedBlockers } from "./relation-utils";
import { popoverTriggerProps } from "../../utils/popover-trigger";
import { Triangle as EstimateIcon } from "lucide-react";
import { STATUS_OPTIONS, PRIORITY_OPTIONS } from "../../constants";

interface PropertySidebarProps {
  issue: Issue;
  issues: Issue[];
  openPopover: string | null;
  setOpenPopover: (v: string | null) => void;
  saveUpdate: (patch: Partial<Issue>, alsoUpdate?: IssuePatch[]) => void;
  onClose: () => void;
  contributors: string[];
  onOpenMerge?: () => void;
}

export default function PropertySidebar({
  issue,
  issues,
  openPopover,
  setOpenPopover,
  saveUpdate,
  onClose,
  contributors,
  onOpenMerge,
}: PropertySidebarProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const { cycles } = useCycles();
  const { data: releases } = useReleases();
  const { assign: assignRelease } = useReleaseAssign();
  const releasedIn = reopenHint(issue, releases);
  const defaultLabels = useContext(DefaultLabelsContext);
  const popoverAnchorRef = useRef<HTMLDivElement>(null);
  const [moveChildrenPrompt, setMoveChildrenPrompt] = useState<{
    status: string;
    doneCount: number;
    children: { id: string; title: string; status: string }[];
  } | null>(null);

  const { startWork, starting, canChat, noGitRoot } = useStartWorkChat();
  const issueChats = useIssueChats(issue.id);
  const chatsRef = useRef<HTMLButtonElement>(null);
  const [chatsOpen, setChatsOpen] = useState(false);
  const { mutate: deleteIssue } = useDeleteIssue();

  // Start Work opens a new chat in Work Mode on the issue (xpo-00f0e5).
  // Continue Work does the same for an issue already DOING, without starting
  // it again, so closing that chat never strands the issue.
  const handleStartWork = useCallback(() => startWork(issue.id), [issue.id, startWork]);
  const showReview = !!(issue.branch_stats && (issue.branch_stats.commits > 0 || issue.branch_stats.has_uncommitted) && !isTerminal(issue.status) && onOpenMerge);
  // A merged issue's changes, read from its merge commit (xpo-64be51).
  const showMerged = !!(issue.merged && issue.status === "DONE" && onOpenMerge);
  const workAction =
    issue.status === "BACKLOG" || issue.status === "PLANNED"
      ? "Start Work"
      : issue.status === "DOING" && issue.branch_stats && canChat
        ? "Continue Work"
        : null;
  // Secondary while reviewing is the next step (xpo-926f0b), or while a chat
  // already works on the issue in Work Mode (xpo-374792).
  const workSecondary = showReview || (workAction === "Continue Work" && !!issueChats.holder);
  // Start Work waits for every blocker to be DONE, as the start gate does (xpo-35943d).
  const blockers = useMemo(
    () => (workAction === "Start Work" ? unresolvedBlockers(issue, issues) : []),
    [workAction, issue, issues],
  );
  // start refuses a BACKLOG issue, so Start Work waits for PLANNED (xpo-31520e).
  const inBacklog = workAction === "Start Work" && issue.status === "BACKLOG";
  // Work Mode can't open without a git repository (xpo-e627e1).
  const startDisabled = !!noGitRoot || inBacklog || blockers.length > 0;
  const handleRemoveBlocker = useCallback(
    (id: string) => saveUpdate({ dependencies: removeLink(issue.dependencies || [], id, "blocked_by") }),
    [issue.dependencies, saveUpdate],
  );

  const subIssueCount = useMemo(() => issues.filter(i => i.parent_id === issue.id).length, [issues, issue.id]);

  const handleDelete = useCallback((cascade: boolean) => {
    setConfirmDelete(false);
    deleteIssue({ issueId: issue.id, cascade }, { onSuccess: onClose });
  }, [issue.id, onClose, deleteIssue]);

  const applyStatusChange = useCallback(
    (status: string, includeChildren: boolean) => {
      const children = includeChildren
        ? issues.filter((i) => i.parent_id === issue.id && i.status !== status && !isTerminal(i.status))
        : [];
      saveUpdate({ status }, children.map((c) => ({ issueId: c.id, patch: { status } })));
    },
    [issue.id, issues, saveUpdate],
  );

  const handleStatusChange = useCallback(
    (newStatus: string) => {
      if (newStatus === issue.status) {
        setOpenPopover(null);
        return;
      }
      const children = issues.filter(
        (i) => i.parent_id === issue.id && i.status !== newStatus && !isTerminal(i.status),
      );
      setOpenPopover(null);
      if (children.length > 0) {
        const doneCount = issues.filter(
          (i) => i.parent_id === issue.id && isTerminal(i.status),
        ).length;
        setMoveChildrenPrompt({
          status: newStatus,
          doneCount,
          children: children.map((c) => ({ id: c.id, title: c.title, status: c.status })),
        });
        return;
      }
      saveUpdate({ status: newStatus });
    },
    [issue.id, issue.status, issues, saveUpdate, setOpenPopover],
  );

  const handleEstimateChange = useCallback(
    (est: number) => {
      if (est !== (issue.estimate || 0)) saveUpdate({ estimate: est });
      else setOpenPopover(null);
    },
    [issue.estimate, saveUpdate, setOpenPopover],
  );

  const handlePriorityChange = useCallback(
    (pri: number) => {
      if (pri !== (issue.priority || 0)) saveUpdate({ priority: pri });
      else setOpenPopover(null);
    },
    [issue.priority, saveUpdate, setOpenPopover],
  );

  const handleLabelToggle = useCallback(
    (label: string) => {
      saveUpdate({ labels: toggleLabel(issue.labels || [], label) });
    },
    [issue.labels, saveUpdate],
  );

  const handleAssigneeChange = useCallback(
    (assignee: string | null) => {
      saveUpdate({ assignee: assignee || "" });
    },
    [saveUpdate],
  );

  const handleCycleChange = useCallback(
    (cycleId: string | null) => {
      saveUpdate({ cycle_id: cycleId || "" });
    },
    [saveUpdate],
  );

  const knownPeople = useMemo(() => collectKnownPeople(issues, contributors), [issues, contributors]);

  const allKnownLabels = useAllLabels(issues);

  const statusMeta = STATUS_OPTIONS.find((s) => s.value === issue.status);

  // The sidebar scrolls on its own but hides its bar so it doesn't sit next
  // to the pane's; the bottom fade marks content below the fold.
  const [scrollRef, fadeBottom] = useScrollFade<HTMLDivElement>();

  return (
    <div
      ref={scrollRef}
      className={cn("w-80 shrink-0 sticky top-0 self-start max-h-[100cqh] overflow-y-auto scrollbar-none pt-12", fadeBottom && "fade-bottom")}
    >
      <div className="px-5 pb-5 space-y-3">
        {/* Review Changes: the primary action when it shows, so it comes first (xpo-926f0b). */}
        {showReview && (
          <button
            onClick={onOpenMerge}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity"
          >
            <GitMerge size={16} />
            Review Changes
          </button>
        )}

        {/* View changes: secondary, since nothing is left to do on a merged issue. */}
        {showMerged && (
          <button
            onClick={onOpenMerge}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
          >
            <GitMerge size={16} />
            View changes
          </button>
        )}

        {/* Start Work / Continue Work button: secondary while Review Changes shows or a chat holds the issue. */}
        {workAction && (() => {
          const button = (
          <button
            onClick={handleStartWork}
            disabled={starting || startDisabled}
            className={`flex items-center justify-center gap-2 w-full px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] transition-opacity disabled:opacity-50 disabled:pointer-events-none ${
              workSecondary
                ? "bg-[var(--color-surface-2)] border border-[var(--color-border-default)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)]"
                : "bg-[var(--color-accent-primary)] text-white hover:opacity-90"
            }`}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
            </svg>
            {starting ? "Starting..." : workAction}
          </button>
          );
          // A disabled button fires no pointer events, so a span triggers the tooltip.
          return startDisabled ? (
            <Tooltip content={<StartWorkBlockedReasons noGitRoot={noGitRoot} inBacklog={inBacklog} blockers={blockers} onRemoveBlocker={handleRemoveBlocker} />}>
              <span className="block cursor-not-allowed">{button}</span>
            </Tooltip>
          ) : button;
        })()}

        {/* An open issue left in a shipped release: offer the way out, move nothing on its own. */}
        {releasedIn && releases && (
          <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-2)] px-4 py-3 text-sm">
            <p className="flex items-center gap-2 text-[var(--color-text-secondary)]">
              <Package size={14} className="shrink-0 text-[var(--color-text-muted)]" />
              <span>In released release {releasedIn.name}.</span>
            </p>
            <button
              type="button"
              onClick={() => assignRelease([issue.id], releases.current)}
              className="mt-2 px-2.5 py-1 text-xs font-medium rounded-[var(--radius-sm)] bg-[var(--color-surface-3)] border border-[var(--color-border-default)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
            >
              Move to Current
            </button>
          </div>
        )}

        {/* Properties card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Properties</div>
          <div className="space-y-1">
            {/* Status */}
            <PropertyRow>
              <div className="relative" ref={openPopover === "status" ? popoverAnchorRef : undefined}>
                <Button
                  variant="ghost"
                  size="sm"
                  {...popoverTriggerProps("status", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "status" ? null : "status")}
                  className="w-full justify-start"
                >
                  <StatusIcon status={issue.status} size={16} isInferred={issue.is_inferred} />
                  <span className="text-sm text-[var(--color-text-primary)]">
                    {statusMeta?.label || issue.status}
                  </span>
                </Button>
                {openPopover === "status" && (
                  <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                    <PopoverPanel>
                      <StatusPicker current={issue.status} onSelect={handleStatusChange} onClose={() => setOpenPopover(null)} />
                    </PopoverPanel>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Estimate */}
            <PropertyRow>
              <div className="relative" ref={openPopover === "estimate" ? popoverAnchorRef : undefined}>
                <Button
                  variant="ghost"
                  size="sm"
                  {...popoverTriggerProps("estimate", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "estimate" ? null : "estimate")}
                  className="w-full justify-start"
                >
                  <EstimateIcon size={16} />
                  <span className="text-sm text-[var(--color-text-primary)]">
                    {issue.estimate
                      ? `${issue.estimate} Point${issue.estimate !== 1 ? "s" : ""}`
                      : "No estimate"}
                  </span>
                </Button>
                {openPopover === "estimate" && (
                  <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                    <PopoverPanel>
                      <EstimatePicker current={issue.estimate || 0} onSelect={handleEstimateChange} onClose={() => setOpenPopover(null)} />
                    </PopoverPanel>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Priority */}
            <PropertyRow>
              <div className="relative" ref={openPopover === "priority" ? popoverAnchorRef : undefined}>
                <Button
                  variant="ghost"
                  size="sm"
                  {...popoverTriggerProps("priority", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "priority" ? null : "priority")}
                  className="w-full justify-start"
                >
                  <PriorityIcon priority={issue.priority || 0} />
                  <span className="text-sm text-[var(--color-text-primary)]">
                    {PRIORITY_OPTIONS.find((o) => o.value === (issue.priority || 0))?.label || "No priority"}
                  </span>
                </Button>
                {openPopover === "priority" && (
                  <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                    <PopoverPanel>
                      <PriorityPicker current={issue.priority || 0} onSelect={handlePriorityChange} onClose={() => setOpenPopover(null)} />
                    </PopoverPanel>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Relationships */}
            <PropertyRow>
              <div className="relative" ref={openPopover === "relations" ? popoverAnchorRef : undefined}>
                <Button
                  variant="ghost"
                  size="sm"
                  {...popoverTriggerProps("relations", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "relations" ? null : "relations")}
                  className="w-full justify-start"
                >
                  <Flag className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  <span className="text-sm text-[var(--color-text-primary)] truncate">
                    {relationshipsLabel(issue, issues)}
                  </span>
                </Button>
                {openPopover === "relations" && (
                  <RelationsAddPopover
                    issue={issue}
                    issues={issues}
                    anchorRef={popoverAnchorRef}
                    onClose={() => setOpenPopover(null)}
                    saveUpdate={saveUpdate}
                  />
                )}
              </div>
            </PropertyRow>

            {/* Chats that worked on the issue (xpo-219de9) */}
            {issueChats.state && (
              <PropertyRow>
                <div className="relative">
                  <Button
                    ref={chatsRef}
                    variant="ghost"
                    size="sm"
                    aria-haspopup="menu"
                    aria-expanded={chatsOpen}
                    onClick={() => setChatsOpen((o) => !o)}
                    className="w-full justify-start"
                  >
                    <IssueChatStateIcon state={issueChats.state} size={16} />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">{chatCount(issueChats.chats.length)}</span>
                  </Button>
                  {chatsOpen && <IssueChatsPopover issueId={issue.id} anchorRef={chatsRef} onClose={() => setChatsOpen(false)} />}
                </div>
              </PropertyRow>
            )}

            {/* Assignee */}
            <PropertyRow>
              <div className="relative" ref={openPopover === "assignee" ? popoverAnchorRef : undefined}>
                <Button
                  variant="ghost"
                  size="sm"
                  {...popoverTriggerProps("assignee", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "assignee" ? null : "assignee")}
                  className="w-full justify-start"
                >
                  {issue.assignee ? (
                    <Avatar name={issue.assignee} size="xs" />
                  ) : (
                    <UserRound className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  )}
                  {issue.assignee ? (
                    <ActorName
                      name={issue.assignee}
                      via={issue.assignee_via}
                      focusable={false}
                      className="text-sm text-[var(--color-text-primary)] truncate"
                    />
                  ) : (
                    <span className="text-sm text-[var(--color-text-primary)] truncate">No assignee</span>
                  )}
                </Button>
                {openPopover === "assignee" && (
                  <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                    <PopoverPanel>
                      <AssigneePicker
                        people={knownPeople}
                        current={issue.assignee}
                        onSelect={handleAssigneeChange}
                        onClose={() => setOpenPopover(null)}
                      />
                    </PopoverPanel>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Cycle */}
            {cycles.length > 0 && (
              <PropertyRow>
                <div className="relative" ref={openPopover === "cycle" ? popoverAnchorRef : undefined}>
                  <Button
                    variant="ghost"
                    size="sm"
                    {...popoverTriggerProps("cycle", openPopover)}
                  onClick={() => setOpenPopover(openPopover === "cycle" ? null : "cycle")}
                    className="w-full justify-start"
                  >
                    <RefreshCw className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {issue.cycle_id
                        ? `Cycle ${cycles.find(c => c.id === issue.cycle_id)?.number || issue.cycle_id}`
                        : "No cycle"}
                    </span>
                  </Button>
                  {openPopover === "cycle" && (
                    <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                      <PopoverPanel>
                        <CyclePicker
                          cycles={cycles}
                          current={issue.cycle_id}
                          onSelect={handleCycleChange}
                          onClose={() => setOpenPopover(null)}
                        />
                      </PopoverPanel>
                    </Popover>
                  )}
                </div>
              </PropertyRow>
            )}

            {/* Release */}
            {releases && (
              <PropertyRow>
                <div className="relative" ref={openPopover === "release" ? popoverAnchorRef : undefined}>
                  <Button
                    variant="ghost"
                    size="sm"
                    {...popoverTriggerProps("release", openPopover)}
                    onClick={() => setOpenPopover(openPopover === "release" ? null : "release")}
                    className="w-full justify-start"
                  >
                    <Package className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {issue.release || "No release"}
                    </span>
                  </Button>
                  {openPopover === "release" && (
                    <ReleasePickerPopover issue={issue} anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)} />
                  )}
                </div>
              </PropertyRow>
            )}
          </div>
        </div>

        {/* Labels card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Labels</div>
          <div className="flex items-center gap-2 flex-wrap">
            {issue.labels && issue.labels.length > 0 ? (
              (() => {
                const { primary, metadata } = splitLabels(issue.labels, defaultLabels);
                return (
                  <>
                    {metadata.map((label) => <LabelBadge key={label} label={label} />)}
                    {primary.map((label) => <LabelBadge key={label} label={label} />)}
                  </>
                );
              })()
            ) : (
              <Text>None</Text>
            )}
            <div className="relative" ref={openPopover === "labels" ? popoverAnchorRef : undefined}>
              <button
                aria-label="Add label"
                {...popoverTriggerProps("labels", openPopover)}
                onClick={() => setOpenPopover(openPopover === "labels" ? null : "labels")}
                className="w-6 h-6 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                </svg>
              </button>
              {openPopover === "labels" && (
                <Popover anchorRef={popoverAnchorRef} onClose={() => setOpenPopover(null)}>
                  <PopoverPanel>
                    <LabelPicker
                      allLabels={allKnownLabels}
                      selected={issue.labels || []}
                      onToggle={handleLabelToggle}
                      canCreateLabels
                      onClose={() => setOpenPopover(null)}
                    />
                  </PopoverPanel>
                </Popover>
              )}
            </div>
          </div>
        </div>

        {/* Branch stats card */}
        {issue.branch_stats && (
          <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
            <div className="flex items-center justify-between mb-3">
              <Text size="xs" weight="medium">Branch</Text>
              {issue.branch_stats.commits > 0 && issue.branch_stats.head_sha ? (
                <span className="inline-flex items-center gap-1.5 h-6 px-2 rounded-2xl border border-[var(--color-border-label)] text-xs font-mono text-[var(--color-text-secondary)]">
                  <GitBranch size={12} strokeWidth={1.5} />
                  {issue.branch_stats.head_sha.slice(0, 7)}
                </span>
              ) : (
                <span className="text-xs text-[var(--color-text-muted)] italic">
                  No commits
                </span>
              )}
            </div>
            <div className="text-sm text-[var(--color-text-primary)] font-mono truncate mb-2">
              {issue.branch_stats.branch}
            </div>
            {issue.branch_stats.commits > 0 && (
              <div className="flex items-center gap-3 text-sm">
                <span className="text-[var(--color-text-secondary)]">
                  {issue.branch_stats.commits} {issue.branch_stats.commits === 1 ? "commit" : "commits"}
                </span>
                <span className="text-[var(--color-text-muted)]">·</span>
                <span className="text-[var(--color-text-secondary)]">
                  {issue.branch_stats.files_changed} {issue.branch_stats.files_changed === 1 ? "file" : "files"}
                </span>
                <span className="text-[var(--color-text-muted)]">·</span>
                <span className="text-green-500">+{issue.branch_stats.insertions}</span>
                <span className="text-red-500">-{issue.branch_stats.deletions}</span>
              </div>
            )}
            {issue.branch_stats.has_uncommitted && (
              <div className="flex items-center justify-between text-sm mt-1">
                <span className="text-[var(--color-text-muted)]">
                  {issue.branch_stats.commits === 0
                    ? `${issue.branch_stats.files_changed} uncommitted ${issue.branch_stats.files_changed === 1 ? "change" : "changes"}`
                    : "Uncommitted changes"}
                </span>
                {issue.branch_stats.commits === 0 && (
                  <span className="flex items-center gap-1.5">
                    <span className="text-green-500">+{issue.branch_stats.insertions}</span>
                    <span className="text-red-500">-{issue.branch_stats.deletions}</span>
                  </span>
                )}
              </div>
            )}
          </div>
        )}

        {/* Metadata card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3 space-y-2">
          <MetaRow label="Created" value={formatRelativeTime(issue.created_at)} />
          <MetaRow label="Updated" value={formatRelativeTime(issue.updated_at)} />
        </div>

        <RelationsCard issue={issue} issues={issues} saveUpdate={saveUpdate} />

        {/* Delete */}
        <div className="pt-2">
          <button
            onClick={() => setConfirmDelete(true)}
            className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-error)] rounded-[var(--radius-md)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
          >
            <Trash2 size={16} />
            Delete issue
          </button>
          {confirmDelete && (
            <DeleteIssueDialog issue={issue} subIssueCount={subIssueCount} onDelete={handleDelete} onCancel={() => setConfirmDelete(false)} />
          )}
        </div>
      </div>
      <Modal
        isOpen={!!moveChildrenPrompt}
        onClose={() => setMoveChildrenPrompt(null)}
        title="Update sub-issues?"
        size="xl"
        showCloseButton={false}
      >
        {moveChildrenPrompt && (() => {
          const targetLabel = STATUS_OPTIONS.find((s) => s.value === moveChildrenPrompt.status)?.label || moveChildrenPrompt.status;
          const fullChildren = moveChildrenPrompt.children.map((c) => {
            const full = issues.find((i) => i.id === c.id);
            return full || c;
          });
          return (
            <>
              <p className="text-sm text-[var(--color-text-secondary)] mb-5 leading-6">
                <span className="text-text-primary">{issue.title}</span>{" "}
                has {moveChildrenPrompt.children.length}{" "}
                {moveChildrenPrompt.children.length === 1 ? "sub-issue " : "sub-issues "}
                in a different status. Do you want to change their status to{" "}
                <StatusIcon status={moveChildrenPrompt.status} size={12} className="inline-block align-[-1px] mx-0.5" />
                <strong className="text-[var(--color-text-primary)]">{targetLabel}</strong>{" "}
                at the same time?
                {moveChildrenPrompt.doneCount > 0 && (
                  <>{" "}{moveChildrenPrompt.doneCount} completed {moveChildrenPrompt.doneCount === 1 ? "issue" : "issues"} will not be updated.</>
                )}
              </p>
              <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden mb-8 max-h-64 overflow-y-auto">
                {fullChildren.map((child, i) => {
                  const rawLabels = "labels" in child ? (child as Issue).labels || [] : [];
                  const { primary: pl, metadata: ml } = splitLabels(rawLabels, defaultLabels);
                  const allLabels = [...pl, ...ml];
                  const extraCount = Math.max(0, allLabels.length - 2);
                  return (
                    <div
                      key={child.id}
                      className={`flex items-center gap-3 px-4 py-2.5 text-sm ${i > 0 ? "border-t border-[var(--color-border-subtle)]" : ""}`}
                    >
                      <StatusIcon status={child.status} size={14} />
                      <span className="text-[var(--color-text-primary)] truncate min-w-0 flex-1">{child.title}</span>
                      {allLabels.slice(0, 2).map((label: string) => (
                        <LabelBadge key={label} label={label} />
                      ))}
                      {extraCount > 0 && (
                        <span className="text-xs text-[var(--color-text-muted)] shrink-0">+{extraCount}</span>
                      )}
                    </div>
                  );
                })}
              </div>
              <div className="flex items-center gap-4">
                <button
                  className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                  onClick={() => setMoveChildrenPrompt(null)}
                >
                  Abort
                </button>
                <div className="flex-1" />
                <button
                  className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                  onClick={() => {
                    const { status } = moveChildrenPrompt;
                    setMoveChildrenPrompt(null);
                    applyStatusChange(status, false);
                  }}
                >
                  Just this issue
                </button>
                <button
                  className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-colors"
                  onClick={() => {
                    const { status } = moveChildrenPrompt;
                    setMoveChildrenPrompt(null);
                    applyStatusChange(status, true);
                  }}
                >
                  Update all
                </button>
              </div>
            </>
          );
        })()}
      </Modal>
    </div>
  );
}

/**
 * Why Start Work is disabled: the project has no git repository (xpo-e627e1),
 * the issue is in BACKLOG (xpo-31520e), then any unresolved blockers.
 */
export function StartWorkBlockedReasons({ noGitRoot, inBacklog, blockers, onRemoveBlocker }: {
  noGitRoot?: string;
  inBacklog: boolean;
  blockers: Issue[];
  onRemoveBlocker: (id: string) => void;
}) {
  return (
    <div className="space-y-1">
      {noGitRoot && (
        <div className="flex items-center gap-2 py-0.5">
          <GitBranch size={12} className="shrink-0" />
          <span className="text-[var(--color-text-primary)]">{noGitHint(noGitRoot)}</span>
        </div>
      )}
      {inBacklog && (
        <div className="flex items-center gap-2 py-0.5">
          <StatusIcon status="BACKLOG" size={12} />
          <span className="text-[var(--color-text-primary)]">Move the issue to Planned to start work</span>
        </div>
      )}
      {blockers.length > 0 && <BlockerList blockers={blockers} onRemove={onRemoveBlocker} />}
    </div>
  );
}

/** Each unresolved blocker, with Remove link for stale ones (xpo-35943d). */
export function BlockerList({ blockers, onRemove }: { blockers: Issue[]; onRemove: (id: string) => void }) {
  return (
    <ul className="space-y-1 py-0.5">
      {blockers.map((b) => {
        const stale = b.status === "CANCELED" || b.status === "DUPLICATE";
        return (
          <li key={b.id} className="flex items-center gap-2">
            <StatusIcon status={b.status} size={12} />
            <span className="text-[var(--color-text-primary)]">
              Blocked by {b.id}
              {stale && ` (${b.status.toLowerCase()})`}
            </span>
            {stale && (
              <button
                type="button"
                onClick={() => onRemove(b.id)}
                className="ml-auto flex items-center gap-1 px-1.5 py-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
              >
                <FlagOff size={12} />
                Remove link
              </button>
            )}
          </li>
        );
      })}
    </ul>
  );
}

function PropertyRow({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between text-sm">
      <span className="text-[var(--color-text-muted)]">{label}</span>
      <span className="text-[var(--color-text-secondary)]">{value}</span>
    </div>
  );
}
