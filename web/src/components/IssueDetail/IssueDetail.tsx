import { useState, useRef, useEffect, useMemo, useCallback } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import { addDraft, createIssue, fetchIssueHistory } from "../../api/client";
import type { Issue, HistoryEvent } from "../../api/client";
import { Avatar, Button, LabelBadge, StatusIcon, CopyableId, Popover, PopoverHeader, LabelPicker } from "../ui";
import MarkdownEditor from "../MarkdownEditor";
import { isEditableTarget } from "../../utils/keyboard";
import { STATUS_OPTIONS, ESTIMATE_OPTIONS, PRIORITY_OPTIONS } from "../../constants";

function linkifyIssueIds(text: string, prefix: string): string {
  const escaped = prefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return text.replace(
    new RegExp(`\\b(${escaped}[a-f0-9]{6})\\b`, "g"),
    "[$1](#/issues/$1)",
  );
}

interface IssueDetailProps {
  issue: Issue;
  issues: Issue[];
  currentIndex: number;
  totalCount: number;
  onClose: () => void;
  onNavigate: (direction: "prev" | "next") => void;
  onRefresh: () => void;
  prefix: string;
  onConfigLabelsChange: (labels: Record<string, string>) => void;
}

export default function IssueDetail({
  issue,
  issues,
  currentIndex,
  totalCount,
  onClose,
  onNavigate,
  onRefresh,
  prefix,
  onConfigLabelsChange,
}: IssueDetailProps) {
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [descClickEvent, setDescClickEvent] = useState<{ clientX: number; clientY: number } | null>(null);
  const [newComment, setNewComment] = useState("");
  const [saving, setSaving] = useState(false);
  const [openPopover, setOpenPopover] = useState<string | null>(null);
  const [popoverIndex, setPopoverIndex] = useState(0);
  const [toast, setToast] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const commentRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setEditingField(null);
    setOpenPopover(null);
    setNewComment("");
    setConfirmDelete(false);
  }, [issue.id]);

  const handleDelete = useCallback(async () => {
    try {
      await addDraft(issue.id, 'DELETE', {});
      onClose();
      onRefresh();
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  }, [issue.id, onClose, onRefresh]);

  const saveDraft = useCallback(
    async (type: string, payload: unknown) => {
      setSaving(true);
      try {
        await addDraft(issue.id, type, payload);
        onRefresh();
      } catch (err) {
        console.error("Failed to save draft:", err);
      } finally {
        setSaving(false);
        setEditingField(null);
        setOpenPopover(null);
      }
    },
    [issue.id, onRefresh],
  );

  const handleStatusChange = useCallback(
    (newStatus: string) => {
      if (newStatus !== issue.status)
        saveDraft("UPDATE", { status: newStatus });
      else setOpenPopover(null);
    },
    [issue.status, saveDraft],
  );

  const handleEstimateChange = useCallback(
    (est: number) => {
      if (est !== (issue.estimate || 0)) saveDraft("UPDATE", { estimate: est });
      else setOpenPopover(null);
    },
    [issue.estimate, saveDraft],
  );

  const handlePriorityChange = useCallback(
    (pri: number) => {
      if (pri !== (issue.priority || 0)) saveDraft("UPDATE", { priority: pri });
      else setOpenPopover(null);
    },
    [issue.priority, saveDraft],
  );

  const handleLabelToggle = useCallback(
    (label: string) => {
      const current = issue.labels || [];
      const next = current.includes(label)
        ? current.filter((l) => l !== label)
        : [...current, label];
      saveDraft("UPDATE", { labels: next });
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

  const [parentSearch, setParentSearch] = useState("");
  const [assigneeSearch, setAssigneeSearch] = useState("");

  const knownPeople = useMemo(() => {
    const byEmail = new Map<string, string>();
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
  }, [issues, assigneeSearch]);

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

  const allKnownLabels = useMemo(() =>
    Array.from(new Set(issues.flatMap((i) => i.labels || []))).sort(),
    [issues]
  );

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
      if (e.key === "Escape") {
        if (openPopover) setOpenPopover(null);
        else if (editingField) setEditingField(null);
        else onClose();
        return;
      }
      if (e.metaKey || e.ctrlKey) return;
      // Keyboard nav inside open popovers
      if (openPopover) {
        if (openPopover === "labels") return;
        const len =
          openPopover === "status"
            ? STATUS_OPTIONS.length
            : openPopover === "estimate"
              ? ESTIMATE_OPTIONS.length
              : openPopover === "priority"
                ? PRIORITY_OPTIONS.length
                : 0;
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setPopoverIndex((i) => Math.min(i + 1, len - 1));
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setPopoverIndex((i) => Math.max(i - 1, 0));
          return;
        }
        if (e.key === "Enter") {
          e.preventDefault();
          if (openPopover === "status")
            handleStatusChange(STATUS_OPTIONS[popoverIndex].value);
          else if (openPopover === "estimate")
            handleEstimateChange(ESTIMATE_OPTIONS[popoverIndex]);
          else if (openPopover === "priority")
            handlePriorityChange(PRIORITY_OPTIONS[popoverIndex].value);
          return;
        }
        if (openPopover === "status") {
          const num = parseInt(e.key);
          if (num >= 1 && num <= 5 && STATUS_OPTIONS[num - 1]) {
            handleStatusChange(STATUS_OPTIONS[num - 1].value);
            return;
          }
        }
        return;
      }
      if (e.key === "ArrowLeft" || e.key === "k") onNavigate("prev");
      if (e.key === "ArrowRight" || e.key === "j") onNavigate("next");
      if (e.key === "s") {
        setOpenPopover("status");
        setPopoverIndex(
          STATUS_OPTIONS.findIndex((o) => o.value === issue.status),
        );
      }
      if (e.key === "l") {
        setOpenPopover("labels");
        setPopoverIndex(0);
      }
      if (e.key === "e") {
        setOpenPopover("estimate");
        setPopoverIndex(ESTIMATE_OPTIONS.indexOf(issue.estimate || 0));
      }
      if (e.key === "p") {
        setOpenPopover("priority");
        setPopoverIndex(
          PRIORITY_OPTIONS.findIndex((o) => o.value === (issue.priority || 0)),
        );
      }
      if (e.key === "a") {
        setOpenPopover("assignee");
        setAssigneeSearch("");
      }
      if (e.key === "m") {
        e.preventDefault();
        commentRef.current?.focus();
        commentRef.current?.scrollIntoView({
          behavior: "smooth",
          block: "center",
        });
      }
      if (e.key === ".") {
        navigator.clipboard.writeText(issue.id);
        setToast("Copied issue ID");
        setTimeout(() => setToast(null), 1500);
      }
      const num = parseInt(e.key);
      if (num >= 1 && num <= 5) {
        const status = STATUS_OPTIONS[num - 1];
        if (status && status.value !== issue.status)
          saveDraft("UPDATE", { status: status.value });
      }
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [
    onClose,
    onNavigate,
    openPopover,
    editingField,
    issue.status,
    issue.estimate,
    issue.priority,
    issue.id,
    saveDraft,
    handleStatusChange,
    handleEstimateChange,
    handlePriorityChange,
    handleLabelToggle,
    allKnownLabels,
    popoverIndex,
  ]);

  const startEditing = (field: string) => {
    setEditingField(field);
    if (field === "title") setEditTitle(issue.title);
    if (field === "description") setEditDescription(issue.description || "");
  };

  const handleSaveTitle = () => {
    if (editTitle.trim() && editTitle !== issue.title) {
      saveDraft("UPDATE", { title: editTitle.trim() });
    } else {
      setEditingField(null);
    }
  };

  const handleSaveDescription = () => {
    if (editDescription !== (issue.description || "")) {
      saveDraft("UPDATE", { description: editDescription });
    } else {
      setEditingField(null);
    }
  };

  const handleAddComment = () => {
    if (newComment.trim()) {
      saveDraft("COMMENT", {
        id: `c-${Date.now().toString(36)}`,
        text: newComment.trim(),
      });
      setNewComment("");
    }
  };

  const statusMeta = STATUS_OPTIONS.find((s) => s.value === issue.status);
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < totalCount - 1;

  return (
    <div className="h-full flex flex-col relative">
      {/* Toast */}
      {toast && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-50 px-3 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] shadow-[var(--shadow-md)] text-sm text-[var(--color-text-primary)] animate-fade-in">
          {toast}
        </div>
      )}
      {/* Top bar: breadcrumb + nav */}
      <div className="flex items-center justify-between px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <div className="flex items-center gap-1.5 text-sm min-w-0">
          <button
            onClick={onClose}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors shrink-0"
          >
            Issues
          </button>
          <svg
            className="w-3 h-3 text-[var(--color-text-muted)] shrink-0"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M9 5l7 7-7 7"
            />
          </svg>
          <CopyableId id={issue.id} className="text-xs shrink-0" />
          <span className="text-[var(--color-text-primary)] truncate">
            {issue.title}
          </span>
        </div>

        <div className="flex items-center gap-1 shrink-0 ml-4">
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums mr-1">
            {currentIndex + 1} / {totalCount}
          </span>
          <button
            onClick={() => onNavigate("prev")}
            disabled={!hasPrev}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-20 disabled:pointer-events-none"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 19l-7-7 7-7"
              />
            </svg>
          </button>
          <button
            onClick={() => onNavigate("next")}
            disabled={!hasNext}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors disabled:opacity-20 disabled:pointer-events-none"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M9 5l7 7-7 7"
              />
            </svg>
          </button>
        </div>
      </div>

      {/* Body: main content + properties sidebar */}
      <div className="flex-1 flex overflow-hidden">
        {/* Main content */}
        <div className="flex-1 overflow-y-auto min-w-0">
          <div className="max-w-4xl mx-auto px-8 py-12">
            {/* Title */}
            {editingField === "title" ? (
              <input
                autoFocus
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleSaveTitle();
                  if (e.key === "Escape") setEditingField(null);
                }}
                onBlur={handleSaveTitle}
                className="w-full text-xl font-semibold bg-transparent text-[var(--color-text-primary)] outline-none border-none m-0 p-0 leading-tight block"
                style={{ caretColor: "var(--color-accent-primary)", height: "auto" }}
              />
            ) : (
              <h1
                onClick={() => startEditing("title")}
                className="text-xl font-semibold text-[var(--color-text-primary)] m-0 p-0 leading-tight cursor-text"
              >
                {issue.title}
              </h1>
            )}

            {/* Parent reference */}
            {issue.parent_id && (() => {
              const parent = issues.find(i => i.id === issue.parent_id);
              const siblings = parent ? issues.filter(i => i.parent_id === parent.id) : [];
              const siblingsDone = siblings.filter(i => i.status === 'DONE').length;
              return (
                <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-muted)] mt-2 flex-wrap">
                  <span>Sub-issue of</span>
                  {parent && <StatusIcon status={parent.status} size={14} />}
                  <a href={`#/issues/${issue.parent_id}`} className="font-mono text-[var(--color-accent-primary)] hover:underline" onClick={(e) => e.stopPropagation()}>
                    {issue.parent_id}
                  </a>
                  {parent && <span className="text-[var(--color-text-secondary)]">{parent.title}</span>}
                  {siblings.length > 0 && (
                    <span className="text-[var(--color-text-muted)]">({siblingsDone}/{siblings.length})</span>
                  )}
                </div>
              );
            })()}

            {/* Description */}
            <div className="mt-4">
              {editingField === "description" ? (
                <MarkdownEditor
                  value={editDescription}
                  onChange={setEditDescription}
                  onSave={handleSaveDescription}
                  onCancel={() => setEditingField(null)}
                  autoFocus
                  clickEvent={descClickEvent}
                  className="prose-beats"
                />
              ) : (
                <div
                  onClick={(e) => { setDescClickEvent({ clientX: e.clientX, clientY: e.clientY }); startEditing("description"); }}
                  className="cursor-text min-h-[40px] prose-beats"
                >
                  {issue.description ? (
                    <Markdown remarkPlugins={[remarkBreaks]}>
                      {linkifyIssueIds(issue.description, prefix)}
                    </Markdown>
                  ) : (
                    <p className="text-base text-[var(--color-text-muted)]">
                      Add a description...
                    </p>
                  )}
                </div>
              )}
            </div>

            {/* Sub-issues table */}
            <SubIssuesTable issue={issue} issues={issues} onRefresh={onRefresh} />

            {/* Activity */}
            <ActivityTimeline
              issue={issue}
              newComment={newComment}
              onNewCommentChange={setNewComment}
              onAddComment={handleAddComment}
              saving={saving}
              commentRef={commentRef}
              prefix={prefix}
            />
          </div>
        </div>

        {/* Properties sidebar */}
        <div className="w-80 overflow-y-auto shrink-0">
          <div className="p-5 space-y-3">
            {/* Properties card */}
            <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] px-4 py-3">
              <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Properties</div>
              <div className="space-y-1">
              {/* Status */}
              <PropertyRow>
                <div className="relative">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      setOpenPopover(openPopover === "status" ? null : "status")
                    }
                    className="w-full justify-start"
                  >
                    <StatusIcon status={issue.status} size={14} />
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
                            className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors ${isFocused ? "bg-[var(--color-bg-hover)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                          >
                            <StatusIcon status={opt.value} size={14} />
                            <span>{opt.label}</span>
                            {isCurrent ? (
                              <svg
                                className="w-3.5 h-3.5 ml-auto"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={2.5}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M5 13l4 4L19 7"
                                />
                              </svg>
                            ) : (
                              <span className="ml-auto text-xs text-[var(--color-text-muted)]">
                                {i + 1}
                              </span>
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
                    onClick={() =>
                      setOpenPopover(
                        openPopover === "estimate" ? null : "estimate",
                      )
                    }
                    className="w-full justify-start"
                  >
                    <EstimateIcon />
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
                            className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors ${isFocused ? "bg-[var(--color-bg-hover)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                          >
                            <span>
                              {est === 0
                                ? "No estimate"
                                : `${est} Point${est !== 1 ? "s" : ""}`}
                            </span>
                            {isCurrent && (
                              <svg
                                className="w-3.5 h-3.5 ml-auto"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={2.5}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M5 13l4 4L19 7"
                                />
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
                    onClick={() =>
                      setOpenPopover(
                        openPopover === "priority" ? null : "priority",
                      )
                    }
                    className="w-full justify-start"
                  >
                    <PriorityIcon priority={issue.priority || 0} />
                    <span className="text-sm text-[var(--color-text-primary)]">
                      {PRIORITY_OPTIONS.find(
                        (o) => o.value === (issue.priority || 0),
                      )?.label || "No priority"}
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
                            className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors ${isFocused ? "bg-[var(--color-bg-hover)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                          >
                            <PriorityIcon priority={opt.value} />
                            <span>{opt.label}</span>
                            {isCurrent && (
                              <svg
                                className="w-3.5 h-3.5 ml-auto"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={2.5}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M5 13l4 4L19 7"
                                />
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
                    <svg className="w-3.5 h-3.5 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
                    </svg>
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {issue.parent_id
                        ? (issues.find(i => i.id === issue.parent_id)?.title || issue.parent_id)
                        : "No parent"}
                    </span>
                  </Button>
                  {openPopover === "parent" && (
                    <Popover onClose={() => setOpenPopover(null)}>
                      <div className="px-3 py-1.5">
                        <input
                          autoFocus
                          value={parentSearch}
                          onChange={(e) => setParentSearch(e.target.value)}
                          placeholder="Search issues..."
                          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
                        />
                      </div>
                      <div className="border-t border-[var(--color-border-subtle)]" />
                      <div className="max-h-[240px] overflow-y-auto">
                        {issue.parent_id && (
                          <button
                            onClick={() => handleParentChange(null)}
                            className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] transition-colors"
                          >
                            Remove parent
                          </button>
                        )}
                        {parentCandidates.slice(0, 15).map((candidate) => {
                          const isCurrent = candidate.id === issue.parent_id;
                          return (
                            <button
                              key={candidate.id}
                              onClick={() => handleParentChange(candidate.id)}
                              className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}
                            >
                              <StatusIcon status={candidate.status} size={12} />
                              <span className="truncate">{candidate.title}</span>
                              {isCurrent && (
                                <svg className="w-3.5 h-3.5 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                                </svg>
                              )}
                            </button>
                          );
                        })}
                        {parentCandidates.length === 0 && (
                          <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching issues</div>
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
                    <Avatar name={issue.assignee || ""} size="xs" />
                    <span className="text-sm text-[var(--color-text-primary)] truncate">
                      {issue.assignee
                        ? issue.assignee.split(" <")[0]
                        : "No assignee"}
                    </span>
                  </Button>
                  {openPopover === "assignee" && (
                    <Popover onClose={() => setOpenPopover(null)}>
                      <div className="px-3 py-1.5">
                        <input
                          autoFocus
                          value={assigneeSearch}
                          onChange={(e) => setAssigneeSearch(e.target.value)}
                          placeholder="Search people..."
                          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
                        />
                      </div>
                      <div className="border-t border-[var(--color-border-subtle)]" />
                      <div className="max-h-[240px] overflow-y-auto">
                        {issue.assignee && (
                          <button
                            onClick={() => handleAssigneeChange(null)}
                            className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] transition-colors"
                          >
                            Remove assignee
                          </button>
                        )}
                        {knownPeople.map((person) => {
                          const isCurrent = person === issue.assignee;
                          return (
                            <button
                              key={person}
                              onClick={() => handleAssigneeChange(person)}
                              className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
                            >
                              <Avatar name={person} size="sm" />
                              <span className="truncate">{person.split(" <")[0]}</span>
                              {isCurrent && (
                                <svg className="w-3.5 h-3.5 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                                </svg>
                              )}
                            </button>
                          );
                        })}
                        {knownPeople.length === 0 && (
                          <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">No matching people</div>
                        )}
                      </div>
                    </Popover>
                  )}
                </div>
              </PropertyRow>


              </div>
            </div>

            {/* Labels card */}
            <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] px-4 py-3">
              <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Labels</div>
              <div className="flex items-center gap-1.5 flex-wrap">
                {issue.labels && issue.labels.length > 0 ? (
                  issue.labels.map((label) => (
                    <LabelBadge key={label} label={label} />
                  ))
                ) : (
                  <span className="text-sm text-[var(--color-text-muted)]">
                    None
                  </span>
                )}
                <div className="relative">
                  <button
                    onClick={() => {
                      const next = openPopover === "labels" ? null : "labels";
                      setOpenPopover(next);
                      if (next) { setPopoverIndex(0); }
                    }}
                    className="w-6 h-6 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                  >
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
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

            {/* Metadata card */}
            <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] px-4 py-3 space-y-1.5">
              <MetaRow
                label="Created"
                value={formatRelativeTime(issue.created_at)}
              />
              <MetaRow
                label="Updated"
                value={formatRelativeTime(issue.updated_at)}
              />
            </div>

            {/* Dependencies card */}
            {issue.dependencies && issue.dependencies.length > 0 && (
              <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] px-4 py-3">
                <div className="text-xs font-medium text-[var(--color-text-muted)] mb-3">Relations</div>
                <div className="space-y-1">
                  {issue.dependencies.map((dep, i) => (
                    <div key={i} className="flex items-center gap-1.5 text-sm">
                      <span className="text-[var(--color-text-muted)]">
                        {dep.kind.replace("_", " ")}
                      </span>
                      <a
                        href={`#/issues/${dep.target_id}`}
                        className="font-mono text-[var(--color-accent-primary)] hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {dep.target_id}
                      </a>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Delete */}
            <div className="pt-2">
              {confirmDelete ? (
                <div className="rounded-[var(--radius-md)] bg-[var(--color-bg-elevated)] border border-[var(--color-error)] p-3">
                  <p className="text-sm text-[var(--color-text-primary)] mb-3">Delete this issue? This cannot be undone.</p>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={handleDelete}
                      className="px-3 py-1.5 text-sm font-medium rounded-[var(--radius-md)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity"
                    >
                      Delete
                    </button>
                    <button
                      onClick={() => setConfirmDelete(false)}
                      className="px-3 py-1.5 text-sm font-medium rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  onClick={() => setConfirmDelete(true)}
                  className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-error)] rounded-[var(--radius-md)] hover:bg-[var(--color-bg-hover)] transition-colors"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                  </svg>
                  Delete issue
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SubIssuesTable({ issue, issues, onRefresh }: { issue: Issue; issues: Issue[]; onRefresh: () => void }) {
  const children = useMemo(() => issues.filter(i => i.parent_id === issue.id), [issues, issue.id]);
  const [expanded, setExpanded] = useState(true);
  const [inlineTitle, setInlineTitle] = useState("");
  const [showInline, setShowInline] = useState(false);
  const inlineRef = useRef<HTMLInputElement>(null);

  const hasChildren = children.length > 0;
  const doneCount = children.filter(c => c.status === 'DONE').length;

  const handleInlineCreate = useCallback(async (title: string) => {
    if (!title.trim()) return;
    await createIssue({ title: title.trim(), labels: ['feature'], parent_id: issue.id });
    setInlineTitle("");
    setShowInline(false);
    onRefresh();
  }, [issue.id, onRefresh]);

  const startInline = useCallback(() => {
    setShowInline(true);
    setExpanded(true);
    setTimeout(() => inlineRef.current?.focus(), 0);
  }, []);

  if (!hasChildren && !showInline) {
    return (
      <div className="mt-6">
        <button
          onClick={startInline}
          className="flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          Add sub-issue
        </button>
      </div>
    );
  }

  return (
    <div className="mt-6">
      <div className="flex items-center gap-2 mb-3">
        <button
          onClick={() => setExpanded(v => !v)}
          className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text-primary)] hover:text-[var(--color-text-secondary)] transition-colors"
        >
          <svg
            className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${expanded ? 'rotate-90' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          Sub-issues
          {hasChildren && <span className="text-xs font-normal text-[var(--color-text-muted)]">{doneCount}/{children.length}</span>}
        </button>
        <button
          onClick={startInline}
          className="p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
          title="Add sub-issue"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
        </button>
      </div>
      {expanded && (
        <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
          {children.map((child, i) => (
            <a
              key={child.id}
              href={`#/issues/${child.id}`}
              className={`flex items-center gap-2.5 px-3 py-2 text-sm hover:bg-[var(--color-bg-hover)] transition-colors ${i > 0 ? 'border-t border-[var(--color-border-subtle)]' : ''}`}
            >
              <StatusIcon status={child.status} size={14} />
              <span className="text-[var(--color-text-primary)] truncate min-w-0">{child.title}</span>
              {child.priority > 0 && (
                <span className={`text-xs font-medium shrink-0 ${child.priority === 1 ? 'text-[var(--color-error)]' : child.priority === 2 ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]'}`}>
                  {child.priority === 1 ? '!!!' : child.priority === 2 ? '!!' : '!'}
                </span>
              )}
              <div className="flex-1" />
              {child.labels?.map(label => <LabelBadge key={label} label={label} />)}
              {child.estimate > 0 && (
                <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
                  <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>
                  {child.estimate}
                </span>
              )}
              {child.created_by && (
                <Avatar name={child.created_by} size="sm" />
              )}
            </a>
          ))}
          {showInline && (
            <div className={`flex items-center gap-2.5 px-3 py-2 ${hasChildren ? 'border-t border-[var(--color-border-subtle)]' : ''}`}>
              <StatusIcon status="BACKLOG" size={14} />
              <input
                ref={inlineRef}
                value={inlineTitle}
                onChange={(e) => setInlineTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && inlineTitle.trim()) handleInlineCreate(inlineTitle);
                  if (e.key === 'Escape') { setShowInline(false); setInlineTitle(""); }
                }}
                onBlur={() => { if (!inlineTitle.trim()) { setShowInline(false); setInlineTitle(""); } }}
                placeholder="Sub-issue title... (Enter to create, Esc to cancel)"
                className="flex-1 bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
              />
            </div>
          )}
        </div>
      )}
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

function PriorityIcon({
  priority,
  size = 14,
}: {
  priority: number;
  size?: number;
}) {
  const colors: Record<number, string> = {
    0: "var(--color-text-muted)",
    1: "var(--color-error)",
    2: "var(--color-warning)",
    3: "var(--color-text-secondary)",
    4: "var(--color-text-muted)",
  };
  const color = colors[priority] || colors[0];

  if (priority === 0) {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 16 16"
        fill="none"
        style={{ color }}
      >
        <rect
          x="1"
          y="7"
          width="3"
          height="2"
          rx="0.5"
          fill="currentColor"
          opacity="0.4"
        />
        <rect
          x="5"
          y="7"
          width="3"
          height="2"
          rx="0.5"
          fill="currentColor"
          opacity="0.4"
        />
        <rect
          x="9"
          y="7"
          width="3"
          height="2"
          rx="0.5"
          fill="currentColor"
          opacity="0.4"
        />
        <rect
          x="13"
          y="7"
          width="2"
          height="2"
          rx="0.5"
          fill="currentColor"
          opacity="0.4"
        />
      </svg>
    );
  }
  if (priority === 1) {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 16 16"
        fill="none"
        style={{ color }}
      >
        <path
          d="M3 2.5L8 1l5 1.5v6c0 3-2.5 5-5 6.5-2.5-1.5-5-3.5-5-6.5v-6z"
          stroke="currentColor"
          strokeWidth="1.5"
          fill="currentColor"
          fillOpacity="0.15"
        />
        <path
          d="M7.5 4.5v4M7.5 10.5v0"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
        />
      </svg>
    );
  }
  const filled = priority === 2 ? 3 : priority === 3 ? 2 : 1;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      style={{ color }}
    >
      {[0, 1, 2].map((i) => (
        <rect
          key={i}
          x={1 + i * 5}
          y={11 - (i + 1) * 3}
          width="4"
          height={(i + 1) * 3}
          rx="1"
          fill="currentColor"
          opacity={i < filled ? 1 : 0.2}
        />
      ))}
    </svg>
  );
}

function EstimateIcon() {
  return (
    <svg
      className="w-3.5 h-3.5 text-[var(--color-text-muted)]"
      viewBox="0 0 16 16"
      fill="none"
    >
      <path
        d="M8 2L14 14H2L8 2Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
    </svg>
  );
}

type ActivityEntry =
  | { kind: "system"; author: string; content: React.ReactNode; time: string }
  | { kind: "comment"; author: string; text: string; time: string };

const STATUS_LABELS: Record<string, string> = {
  BACKLOG: "Backlog",
  PLANNED: "Planned",
  DOING: "In Progress",
  BLOCKED: "Blocked",
  DONE: "Done",
};

function StatusChip({ status }: { status: string }) {
  return (
    <span className="flex items-center gap-1">
      <StatusIcon status={status} size={12} />
      <span className="font-medium text-[var(--color-text-primary)]">
        {STATUS_LABELS[status] || status}
      </span>
    </span>
  );
}

function EstimateChip({ points }: { points: number }) {
  return (
    <span className="flex items-center gap-1 font-medium text-[var(--color-text-primary)]">
      <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none">
        <path
          d="M8 2L14 14H2L8 2Z"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinejoin="round"
        />
      </svg>
      {points} {points === 1 ? "Point" : "Points"}
    </span>
  );
}

function describeEvent(evt: HistoryEvent): React.ReactNode | null {
  const p = evt.payload || {};
  switch (evt.type) {
    case "CREATE":
      return "created this issue";
    case "UPDATE": {
      const fragments: React.ReactNode[] = [];
      if (p.status)
        fragments.push(
          <>
            changed status to <StatusChip status={String(p.status)} />
          </>,
        );
      if (p.estimate !== undefined)
        fragments.push(
          <>
            set estimate to <EstimateChip points={Number(p.estimate)} />
          </>,
        );
      if (p.title) fragments.push(<>updated the title</>);
      if (p.description !== undefined)
        fragments.push(<>updated the description</>);
      if (p.labels)
        fragments.push(
          <>
            updated labels to{" "}
            {(p.labels as string[]).map((l) => (
              <LabelBadge key={l} label={l} />
            ))}
          </>,
        );
      if (p.assignee)
        fragments.push(
          <>
            assigned to{" "}
            <span className="font-medium text-[var(--color-text-primary)]">
              {String(p.assignee)}
            </span>
          </>,
        );
      if (fragments.length === 0) return null;
      return fragments.reduce<React.ReactNode[]>((acc, f, i) => {
        if (i > 0) acc.push(<span key={`sep-${i}`}> and </span>);
        acc.push(f);
        return acc;
      }, []);
    }
    case "DELETE":
      return "deleted this issue";
    default:
      return null;
  }
}

function ActivityTimeline({
  issue,
  newComment,
  onNewCommentChange,
  onAddComment,
  saving,
  commentRef,
  prefix,
}: {
  issue: Issue;
  newComment: string;
  onNewCommentChange: (v: string) => void;
  onAddComment: () => void;
  saving: boolean;
  commentRef: React.RefObject<HTMLTextAreaElement | null>;
  prefix: string;
}) {
  const [history, setHistory] = useState<HistoryEvent[]>([]);
  const [sortNewest, setSortNewest] = useState(true);

  useEffect(() => {
    fetchIssueHistory(issue.id)
      .then(setHistory)
      .catch(() => {});
  }, [issue.id, issue.updated_at]);

  const entries = useMemo(() => {
    const items: ActivityEntry[] = [];

    for (const evt of history) {
      if (evt.type === "COMMENT") {
        const p = evt.payload || {};
        items.push({
          kind: "comment",
          author: evt.created_by,
          text: String(p.text || ""),
          time: evt.created_at,
        });
      } else {
        const desc = describeEvent(evt);
        if (desc) {
          items.push({
            kind: "system",
            author: evt.created_by,
            content: desc,
            time: evt.created_at,
          });
        }
      }
    }

    const dir = sortNewest ? -1 : 1;
    items.sort(
      (a, b) => dir * (new Date(a.time).getTime() - new Date(b.time).getTime()),
    );
    return items;
  }, [history, sortNewest]);

  return (
    <div className="mt-8 pt-6 border-t border-[var(--color-border-subtle)]">
      <div className="flex items-center justify-between mb-5">
        <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
          Activity
        </h3>
        <button
          onClick={() => setSortNewest(!sortNewest)}
          className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          title={sortNewest ? "Showing newest first" : "Showing oldest first"}
        >
          {sortNewest ? "Newest" : "Oldest"}
          <svg
            className={`w-3 h-3 transition-transform ${sortNewest ? "" : "rotate-180"}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>
      </div>

      <div className="space-y-4">
        {entries.map((entry, i) => {
          if (entry.kind === "system") {
            return (
              <div
                key={`sys-${i}`}
                className="flex items-center gap-2 px-3.5 py-1.5 flex-wrap text-sm text-[var(--color-text-muted)]"
              >
                <Avatar name={entry.author} size="sm" />
                <span>{shortName(entry.author)}</span>
                {entry.content}
                <span>·</span>
                <span>{formatRelativeTime(entry.time)}</span>
              </div>
            );
          }

          return (
            <div
              key={`cmt-${i}`}
              className="rounded-[var(--radius-lg)] bg-[var(--color-bg-secondary)] py-3 px-3.5 border border-[var(--color-border-card)]"
            >
              <div className="flex items-center gap-2.5 mb-2">
                <Avatar name={entry.author} size="sm" />
                <span className="text-sm font-medium text-[var(--color-text-primary)]">
                  {shortName(entry.author)}
                </span>
                <span className="text-sm text-[var(--color-text-muted)]">
                  {formatRelativeTime(entry.time)}
                </span>
              </div>
              <div className="prose-beats text-base">
                <Markdown remarkPlugins={[remarkBreaks]}>
                  {linkifyIssueIds(entry.text, prefix)}
                </Markdown>
              </div>
            </div>
          );
        })}
      </div>

      {/* Comment input */}
      <div className="mt-5 rounded-[var(--radius-lg)] bg-[var(--color-bg-secondary)] border border-[var(--color-border-card)] overflow-hidden">
        <textarea
          ref={commentRef}
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder="Leave a comment..."
          rows={1}
          className="w-full text-base bg-transparent text-[var(--color-text-primary)] px-3.5 py-2.5 outline-none placeholder:text-[var(--color-text-muted)] resize-none"
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onAddComment();
          }}
        />
        <div className="flex items-center justify-end gap-2 px-3 py-2">
          <button
            onClick={onAddComment}
            disabled={!newComment.trim() || saving}
            className="w-7 h-7 flex items-center justify-center rounded-full bg-[var(--color-accent-primary)] text-white disabled:opacity-20 hover:bg-[var(--color-accent-primary-hover)] transition-colors"
          >
            <svg
              className="w-3.5 h-3.5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4.5 10.5L12 3m0 0l7.5 7.5M12 3v18"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

function shortName(fullName: string): string {
  return fullName.split(" <")[0];
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}
