import { useState, useMemo, useCallback, useEffect, useContext } from "react";
import { addDraft, startWork, ApiError, fetchCycles } from "../../api/client";
import type { Issue, Cycle } from "../../api/client";
import { Avatar, Button, LabelBadge, DefaultLabelsContext, Modal, StatusIcon, Popover, PopoverHeader, LabelPicker } from "../ui";
import { Folder, UserRound, RefreshCw, Trash2 } from "lucide-react";
import { GitBranch, GitMerge } from "lucide-react";
import { formatRelativeTime } from "../../utils/format";
import { useAllLabels } from "../../hooks/useLabels";
import { toggleLabel, splitLabels } from "../../utils/labels";
import { isTerminal } from "../../constants";
import { PriorityIcon } from "./icons";
import { Triangle as EstimateIcon } from "lucide-react";
import { STATUS_OPTIONS, ESTIMATE_OPTIONS, PRIORITY_OPTIONS } from "../../constants";

const RELATION_TYPES = [
  { value: "blocks", label: "Blocks" },
  { value: "blocked_by", label: "Blocked by" },
  { value: "depends_on", label: "Depends on" },
  { value: "dependency_of", label: "Dependency of" },
  { value: "duplicates", label: "Duplicates" },
  { value: "duplicated_by", label: "Duplicated by" },
  { value: "relates_to", label: "Relates to" },
];

interface PropertySidebarProps {
  issue: Issue;
  issues: Issue[];
  openPopover: string | null;
  setOpenPopover: (v: string | null) => void;
  popoverIndex: number;
  setPopoverIndex: (v: number) => void;
  saveDraft: (type: string, payload: unknown) => Promise<void>;
  onClose: () => void;
  onRefresh: () => void;
  contributors: string[];
  onConfigLabelsChange: (labels: Record<string, string>) => void;
  onOpenMerge?: () => void;
}

export default function PropertySidebar({
  issue,
  issues,
  openPopover,
  setOpenPopover,
  popoverIndex,
  setPopoverIndex,
  saveDraft,
  onClose,
  onRefresh,
  contributors,
  onConfigLabelsChange,
  onOpenMerge,
}: PropertySidebarProps) {
  const [starting, setStarting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<false | "confirm" | "choose">(false);
  const [addRelStep, setAddRelStep] = useState<"kind" | "issue" | null>(null);
  const [addRelKind, setAddRelKind] = useState("");
  const [addRelSearch, setAddRelSearch] = useState("");
  const [parentSearch, setParentSearch] = useState("");
  const [assigneeSearch, setAssigneeSearch] = useState("");
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const defaultLabels = useContext(DefaultLabelsContext);
  const [moveChildrenPrompt, setMoveChildrenPrompt] = useState<{
    status: string;
    doneCount: number;
    children: { id: string; title: string; status: string }[];
  } | null>(null);

  useEffect(() => {
    fetchCycles().then(data => {
      if (data.enabled && data.cycles) setCycles(data.cycles);
    }).catch(() => {});
  }, []);

  const handleStartWork = useCallback(async () => {
    setStarting(true);
    try {
      await startWork(issue.id);
      onRefresh();
    } catch (err) {
      console.error('Failed to start work:', err instanceof ApiError ? err.message : err);
    } finally {
      setStarting(false);
    }
  }, [issue.id, onRefresh]);

  const hasChildren = useMemo(() => issues.some(i => i.parent_id === issue.id), [issues, issue.id]);

  const handleDelete = useCallback(async (cascade = false) => {
    try {
      await addDraft(issue.id, 'DELETE', { cascade });
      onClose();
      onRefresh();
    } catch (err) {
      console.error('Failed to delete:', err instanceof ApiError ? err.message : err);
    }
  }, [issue.id, onClose, onRefresh]);

  const applyStatusChange = useCallback(
    async (status: string, includeChildren: boolean) => {
      await saveDraft("UPDATE", { status });
      if (includeChildren) {
        const children = issues.filter(
          (i) => i.parent_id === issue.id && i.status !== status && !isTerminal(i.status),
        );
        for (const child of children) {
          await addDraft(child.id, "UPDATE", { status });
        }
        onRefresh();
      }
    },
    [issue.id, issues, saveDraft, onRefresh],
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
      saveDraft("UPDATE", { status: newStatus });
    },
    [issue.id, issue.status, issues, saveDraft, setOpenPopover],
  );

  const handleEstimateChange = useCallback(
    (est: number) => {
      if (est !== (issue.estimate || 0)) saveDraft("UPDATE", { estimate: est });
      else setOpenPopover(null);
    },
    [issue.estimate, saveDraft, setOpenPopover],
  );

  const handlePriorityChange = useCallback(
    (pri: number) => {
      if (pri !== (issue.priority || 0)) saveDraft("UPDATE", { priority: pri });
      else setOpenPopover(null);
    },
    [issue.priority, saveDraft, setOpenPopover],
  );

  const handleLabelToggle = useCallback(
    (label: string) => {
      saveDraft("UPDATE", { labels: toggleLabel(issue.labels || [], label) });
    },
    [issue.labels, saveDraft],
  );

  const handleParentChange = useCallback(
    (parentId: string | null) => {
      saveDraft("UPDATE", { parent_id: parentId || "" });
    },
    [saveDraft],
  );

  const handleAssigneeChange = useCallback(
    (assignee: string | null) => {
      saveDraft("UPDATE", { assignee: assignee || "" });
    },
    [saveDraft],
  );

  const handleCycleChange = useCallback(
    (cycleId: string | null) => {
      saveDraft("UPDATE", { cycle_id: cycleId || "" });
    },
    [saveDraft],
  );

  const handleAddRelation = useCallback(
    (targetId: string, kind: string) => {
      const existing = issue.dependencies || [];
      saveDraft("UPDATE", {
        dependencies: [...existing, { source_id: issue.id, target_id: targetId, kind }],
      });
      setAddRelStep(null);
      setAddRelKind("");
      setAddRelSearch("");
    },
    [issue.id, issue.dependencies, saveDraft],
  );

  const handleRemoveRelation = useCallback(
    (index: number) => {
      const existing = issue.dependencies || [];
      saveDraft("UPDATE", {
        dependencies: existing.filter((_, i) => i !== index),
      });
    },
    [issue.dependencies, saveDraft],
  );

  const relCandidates = useMemo(() => {
    const linkedIds = new Set((issue.dependencies || []).map(d => d.target_id));
    const q = addRelSearch.toLowerCase();
    return issues.filter(i =>
      i.id !== issue.id &&
      !linkedIds.has(i.id) &&
      (!q || i.title.toLowerCase().includes(q) || i.id.toLowerCase().includes(q))
    );
  }, [issues, issue.id, issue.dependencies, addRelSearch]);

  const knownPeople = useMemo(() => {
    const byEmail = new Map<string, string>();
    for (const val of contributors) {
      const email = val.match(/<([^>]+)>/)?.[1]?.toLowerCase() || val;
      if (!byEmail.has(email)) byEmail.set(email, val);
    }
    for (const i of issues) {
      for (const val of [i.created_by, i.assignee]) {
        if (!val) continue;
        const email = val.match(/<([^>]+)>/)?.[1]?.toLowerCase() || val;
        if (!byEmail.has(email)) byEmail.set(email, val);
      }
    }
    const all = Array.from(byEmail.values()).sort((a, b) =>
      a.split(" <")[0].localeCompare(b.split(" <")[0])
    );
    const q = assigneeSearch.toLowerCase();
    if (!q) return all;
    return all.filter(p => p.toLowerCase().includes(q));
  }, [issues, contributors, assigneeSearch]);

  const parentCandidates = useMemo(() => {
    const descendants = new Set<string>();
    const collectDescendants = (id: string) => {
      for (const i of issues) {
        if (i.parent_id === id) {
          descendants.add(i.id);
          collectDescendants(i.id);
        }
      }
    };
    collectDescendants(issue.id);

    const q = parentSearch.toLowerCase();
    return issues.filter(i =>
      i.id !== issue.id &&
      !descendants.has(i.id) &&
      !i.parent_id &&
      i.status !== 'DONE' &&
      (!q || i.title.toLowerCase().includes(q) || i.id.toLowerCase().includes(q))
    );
  }, [issues, issue.id, parentSearch]);

  const allKnownLabels = useAllLabels(issues);

  const statusMeta = STATUS_OPTIONS.find((s) => s.value === issue.status);

  return (
    <div className="w-80 shrink-0 sticky top-0 self-start max-h-screen overflow-y-auto pt-12">
      <div className="px-5 pb-5 space-y-3">
        {/* Start Work button */}
        {(issue.status === "BACKLOG" || issue.status === "PLANNED") && (
          <button
            onClick={handleStartWork}
            disabled={starting}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
            </svg>
            {starting ? "Starting..." : "Start Work"}
          </button>
        )}

        {/* Merge / Review button */}
        {issue.branch_stats && (issue.branch_stats.commits > 0 || issue.branch_stats.has_uncommitted) && !isTerminal(issue.status) && onOpenMerge && (
          <button
            onClick={onOpenMerge}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-opacity"
          >
            <GitMerge size={16} />
            {issue.branch_stats.commits > 0 ? "Merge Branch" : "Review Changes"}
          </button>
        )}

        {/* Properties card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Properties</div>
          <div className="space-y-1">
            {/* Status */}
            <PropertyRow>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOpenPopover(openPopover === "status" ? null : "status")}
                  className="w-full justify-start"
                >
                  <StatusIcon status={issue.status} size={16} isInferred={issue.is_inferred} />
                  <span className="text-sm text-[var(--color-text-primary)]">
                    {statusMeta?.label || issue.status}
                  </span>
                </Button>
                {openPopover === "status" && (
                  <Popover onClose={() => setOpenPopover(null)}>
                    <PopoverHeader>Change status...</PopoverHeader>
                    {STATUS_OPTIONS.map((opt, i) => {
                      const isCurrent = opt.value === issue.status;
                      const isFocused = i === popoverIndex;
                      return (
                        <button
                          key={opt.value}
                          onClick={() => handleStatusChange(opt.value)}
                          onMouseEnter={() => setPopoverIndex(i)}
                          className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors ${isFocused ? "bg-[var(--color-hover-surface-2)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                        >
                          <StatusIcon status={opt.value} size={16} />
                          <span>{opt.label}</span>
                          {isCurrent ? (
                            <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          ) : (
                            <span className="ml-auto text-xs text-[var(--color-text-muted)]">{i + 1}</span>
                          )}
                        </button>
                      );
                    })}
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Estimate */}
            <PropertyRow>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
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
                  <Popover onClose={() => setOpenPopover(null)}>
                    <PopoverHeader>Change estimate to...</PopoverHeader>
                    {ESTIMATE_OPTIONS.map((est, i) => {
                      const isCurrent = est === (issue.estimate || 0);
                      const isFocused = i === popoverIndex;
                      return (
                        <button
                          key={est}
                          onClick={() => handleEstimateChange(est)}
                          onMouseEnter={() => setPopoverIndex(i)}
                          className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors ${isFocused ? "bg-[var(--color-hover-surface-2)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                        >
                          <span>{est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`}</span>
                          {isCurrent && (
                            <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          )}
                        </button>
                      );
                    })}
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Priority */}
            <PropertyRow>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOpenPopover(openPopover === "priority" ? null : "priority")}
                  className="w-full justify-start"
                >
                  <PriorityIcon priority={issue.priority || 0} />
                  <span className="text-sm text-[var(--color-text-primary)]">
                    {PRIORITY_OPTIONS.find((o) => o.value === (issue.priority || 0))?.label || "No priority"}
                  </span>
                </Button>
                {openPopover === "priority" && (
                  <Popover onClose={() => setOpenPopover(null)}>
                    <PopoverHeader>Set priority...</PopoverHeader>
                    {PRIORITY_OPTIONS.map((opt, i) => {
                      const isCurrent = opt.value === (issue.priority || 0);
                      const isFocused = i === popoverIndex;
                      return (
                        <button
                          key={opt.value}
                          onClick={() => handlePriorityChange(opt.value)}
                          onMouseEnter={() => setPopoverIndex(i)}
                          className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors ${isFocused ? "bg-[var(--color-hover-surface-2)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                        >
                          <PriorityIcon priority={opt.value} />
                          <span>{opt.label}</span>
                          {isCurrent && (
                            <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          )}
                        </button>
                      );
                    })}
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Parent */}
            <PropertyRow>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setOpenPopover(openPopover === "parent" ? null : "parent");
                    setParentSearch("");
                  }}
                  className="w-full justify-start"
                >
                  <Folder className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  <span className="text-sm text-[var(--color-text-primary)] truncate">
                    {issue.parent_id
                      ? (issues.find(i => i.id === issue.parent_id)?.title || issue.parent_id)
                      : "No parent"}
                  </span>
                </Button>
                {openPopover === "parent" && (
                  <Popover onClose={() => setOpenPopover(null)}>
                    <div className="px-3 py-2">
                      <input autoFocus value={parentSearch} onChange={(e) => setParentSearch(e.target.value)} placeholder="Search issues..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" />
                    </div>
                    <div className="border-t border-[var(--color-border-subtle)]" />
                    <div className="max-h-60 overflow-y-auto">
                      {issue.parent_id && (
                        <button onClick={() => handleParentChange(null)} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-2)] transition-colors">
                          Remove parent
                        </button>
                      )}
                      {parentCandidates.slice(0, 15).map((candidate) => {
                        const isCurrent = candidate.id === issue.parent_id;
                        return (
                          <button key={candidate.id} onClick={() => handleParentChange(candidate.id)} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-2)] ${isCurrent ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}>
                            <StatusIcon status={candidate.status} size={12} isInferred={candidate.is_inferred} />
                            <span className="truncate">{candidate.title}</span>
                            {isCurrent && (
                              <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                              </svg>
                            )}
                          </button>
                        );
                      })}
                      {parentCandidates.length === 0 && (
                        <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching issues</div>
                      )}
                    </div>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Assignee */}
            <PropertyRow>
              <div className="relative">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setOpenPopover(openPopover === "assignee" ? null : "assignee");
                    setAssigneeSearch("");
                  }}
                  className="w-full justify-start"
                >
                  {issue.assignee ? (
                    <Avatar name={issue.assignee} size="xs" />
                  ) : (
                    <UserRound className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  )}
                  <span className="text-sm text-[var(--color-text-primary)] truncate">
                    {issue.assignee ? issue.assignee.split(" <")[0] : "No assignee"}
                  </span>
                </Button>
                {openPopover === "assignee" && (
                  <Popover onClose={() => setOpenPopover(null)}>
                    <div className="px-3 py-2">
                      <input autoFocus value={assigneeSearch} onChange={(e) => setAssigneeSearch(e.target.value)} placeholder="Search people..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" />
                    </div>
                    <div className="border-t border-[var(--color-border-subtle)]" />
                    <div className="max-h-60 overflow-y-auto">
                      {issue.assignee && (
                        <button onClick={() => handleAssigneeChange(null)} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-2)] transition-colors">
                          Remove assignee
                        </button>
                      )}
                      {knownPeople.map((person) => {
                        const isCurrent = person === issue.assignee;
                        return (
                          <button key={person} onClick={() => handleAssigneeChange(person)} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-2)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}>
                            <Avatar name={person} size="sm" />
                            <span className="truncate">{person.split(" <")[0]}</span>
                            {isCurrent && (
                              <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                              </svg>
                            )}
                          </button>
                        );
                      })}
                      {knownPeople.length === 0 && (
                        <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching people</div>
                      )}
                    </div>
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Cycle */}
            {cycles.length > 0 && (
              <PropertyRow>
                <div className="relative">
                  <Button
                    variant="ghost"
                    size="sm"
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
                    <Popover onClose={() => setOpenPopover(null)}>
                      <PopoverHeader>Move to cycle...</PopoverHeader>
                      {issue.cycle_id && (
                        <button
                          onClick={() => handleCycleChange(null)}
                          className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
                        >
                          No cycle
                        </button>
                      )}
                      {cycles.filter(c => c.status !== 'completed').map(c => {
                        const isCurrent = c.id === issue.cycle_id;
                        return (
                          <button
                            key={c.id}
                            onClick={() => handleCycleChange(c.id)}
                            className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-2)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                          >
                            <span>Cycle {c.number}</span>
                            <span className="text-xs text-[var(--color-text-muted)] capitalize">{c.status}</span>
                            {isCurrent && (
                              <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                              </svg>
                            )}
                          </button>
                        );
                      })}
                    </Popover>
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
              <span className="text-sm text-[var(--color-text-muted)]">None</span>
            )}
            <div className="relative">
              <button
                onClick={() => {
                  const next = openPopover === "labels" ? null : "labels";
                  setOpenPopover(next);
                  if (next) setPopoverIndex(0);
                }}
                className="w-6 h-6 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                </svg>
              </button>
              {openPopover === "labels" && (
                <Popover onClose={() => setOpenPopover(null)}>
                  <LabelPicker
                    allLabels={allKnownLabels}
                    selected={issue.labels || []}
                    onToggle={handleLabelToggle}
                    onConfigLabelsChange={onConfigLabelsChange}
                    onClose={() => setOpenPopover(null)}
                  />
                </Popover>
              )}
            </div>
          </div>
        </div>

        {/* Branch stats card */}
        {issue.branch_stats && (
          <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
            <div className="flex items-center justify-between mb-3">
              <span className="text-xs font-medium text-[var(--color-text-muted)]">Branch</span>
              {issue.branch_stats.head_sha && (
                <span className="inline-flex items-center gap-1.5 h-6 px-2 rounded-2xl border border-[var(--color-border-label)] text-xs font-mono text-[var(--color-text-secondary)]">
                  <GitBranch size={12} strokeWidth={1.5} />
                  {issue.branch_stats.head_sha}
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
          </div>
        )}

        {/* Metadata card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3 space-y-2">
          <MetaRow label="Created" value={formatRelativeTime(issue.created_at)} />
          <MetaRow label="Updated" value={formatRelativeTime(issue.updated_at)} />
        </div>

        {/* Relations card */}
        <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3">
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Relations</div>
          {issue.dependencies && issue.dependencies.length > 0 && (
            <div className="space-y-1 mb-2">
              {issue.dependencies.map((dep, i) => {
                const target = issues.find(t => t.id === dep.target_id) ?? issues.find(t => t.id.endsWith(dep.target_id));
                return (
                  <div key={i} className="group flex items-center gap-2 text-sm">
                    <span className="text-[var(--color-text-muted)] shrink-0">{dep.kind.replace(/_/g, " ")}</span>
                    {target ? (
                      <>
                        <StatusIcon status={target.status} size={12} isInferred={target.is_inferred} />
                        <a href={`#/issues/${target.id}`} className="text-[var(--color-text-primary)] hover:text-[var(--color-accent-primary)] truncate" onClick={(e) => e.stopPropagation()}>
                          {target.title}
                        </a>
                      </>
                    ) : (
                      <span className="text-[var(--color-text-muted)] italic truncate">{dep.target_id} (deleted)</span>
                    )}
                    <button onClick={() => handleRemoveRelation(i)} className="ml-auto shrink-0 p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-error)] opacity-0 group-hover:opacity-100 transition-opacity">
                      <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                );
              })}
            </div>
          )}
          <button
            onClick={() => { setAddRelStep("kind"); setAddRelKind(""); setAddRelSearch(""); }}
            className="flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            Add relation
          </button>
          <Modal
            isOpen={addRelStep !== null}
            onClose={() => { setAddRelStep(null); setAddRelKind(""); setAddRelSearch(""); }}
            title="Add relation"
            size="2xl"
          >
            <div className="space-y-4">
              <div className="flex items-center gap-2 flex-wrap">
                {RELATION_TYPES.map((rt) => (
                  <button
                    key={rt.value}
                    onClick={() => { setAddRelKind(rt.value); setAddRelSearch(""); }}
                    className={`px-3 py-2 text-sm rounded-[var(--radius-md)] border transition-colors ${addRelKind === rt.value ? "border-[var(--color-accent-primary)] text-[var(--color-accent-primary)] bg-[var(--color-accent-primary)]/10" : "border-[var(--color-border-default)] text-[var(--color-text-secondary)] hover:border-[var(--color-border-focus)] hover:text-[var(--color-text-primary)]"}`}
                  >
                    {rt.label}
                  </button>
                ))}
              </div>
              {addRelKind && (
                <>
                  <input
                    autoFocus
                    value={addRelSearch}
                    onChange={(e) => setAddRelSearch(e.target.value)}
                    placeholder="Search issues by title or ID..."
                    className="w-full text-sm bg-[var(--color-surface-1)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none rounded-[var(--radius-md)] border border-[var(--color-border-default)] focus:border-[var(--color-border-focus)] px-3 py-2"
                  />
                  <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
                    <div className="flex items-center gap-3 px-3 py-2 text-xs font-medium text-[var(--color-text-muted)] border-b border-[var(--color-border-subtle)] bg-[var(--color-surface-1)]">
                      <span className="w-4" />
                      <span className="flex-1">Title</span>
                      <span className="w-24 shrink-0">Labels</span>
                      <span className="w-12 shrink-0 text-right">Est</span>
                      <span className="w-8 shrink-0" />
                    </div>
                    <div className="max-h-[50vh] overflow-y-auto">
                      {relCandidates.slice(0, 30).map((candidate) => (
                        <button
                          key={candidate.id}
                          onClick={() => handleAddRelation(candidate.id, addRelKind)}
                          className="flex items-center gap-3 w-full px-3 py-2 text-sm text-left hover:bg-[var(--color-hover-surface-2)] transition-colors border-b border-[var(--color-border-subtle)] last:border-b-0"
                        >
                          <StatusIcon status={candidate.status} size={14} />
                          <div className="flex-1 min-w-0">
                            <div className="text-[var(--color-text-primary)] truncate">{candidate.title}</div>
                            <div className="text-xs text-[var(--color-text-muted)] font-mono">{candidate.id}</div>
                          </div>
                          <div className="w-24 shrink-0 flex items-center gap-1 flex-wrap">
                            {candidate.labels?.slice(0, 2).map(l => <LabelBadge key={l} label={l} />)}
                          </div>
                          <span className="w-12 shrink-0 text-right text-xs text-[var(--color-text-muted)] tabular-nums">
                            {candidate.estimate > 0 ? candidate.estimate : "—"}
                          </span>
                          {candidate.assignee ? (
                            <Avatar name={candidate.assignee} size="sm" />
                          ) : <span className="w-4 shrink-0" />}
                        </button>
                      ))}
                      {relCandidates.length === 0 && (
                        <div className="px-3 py-6 text-sm text-[var(--color-text-muted)] text-center">No matching issues</div>
                      )}
                    </div>
                  </div>
                </>
              )}
              {!addRelKind && (
                <div className="text-sm text-[var(--color-text-muted)] text-center py-6">
                  Select a relationship type above to continue
                </div>
              )}
            </div>
          </Modal>
        </div>

        {/* Delete */}
        <div className="pt-2">
          {confirmDelete === "choose" ? (
            <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-error)] p-3">
              <p className="text-sm text-[var(--color-text-primary)] mb-3">This issue has sub-issues. What should happen to them?</p>
              <div className="flex flex-col gap-2">
                <button onClick={() => handleDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity text-left">
                  Keep sub-issues
                </button>
                <button onClick={() => handleDelete(true)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] border border-[var(--color-error)] text-[var(--color-error)] hover:bg-[var(--color-error)] hover:text-white transition-colors text-left">
                  Delete sub-issues too
                </button>
                <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-2)] transition-colors text-left">
                  Cancel
                </button>
              </div>
            </div>
          ) : confirmDelete === "confirm" ? (
            <div className="rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-error)] p-3">
              <p className="text-sm text-[var(--color-text-primary)] mb-3">Delete this issue? This cannot be undone.</p>
              <div className="flex items-center gap-2">
                <button onClick={() => handleDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity">
                  Delete
                </button>
                <button onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-2)] transition-colors">
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <button
              onClick={() => setConfirmDelete(hasChildren ? "choose" : "confirm")}
              className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-error)] rounded-[var(--radius-md)] hover:bg-[var(--color-hover-surface-2)] transition-colors"
            >
              <Trash2 size={16} />
              Delete issue
            </button>
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
                  onClick={async () => {
                    const { status } = moveChildrenPrompt;
                    setMoveChildrenPrompt(null);
                    await applyStatusChange(status, false);
                  }}
                >
                  Just this issue
                </button>
                <button
                  className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-colors"
                  onClick={async () => {
                    const { status } = moveChildrenPrompt;
                    setMoveChildrenPrompt(null);
                    await applyStatusChange(status, true);
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
