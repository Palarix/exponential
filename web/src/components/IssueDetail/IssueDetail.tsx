import { useState, useRef, useEffect, useCallback } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { addDraft, ApiError } from "../../api/client";
import type { Issue } from "../../api/client";
import { StatusIcon, CopyableId, useToast, ChevronRightIcon } from "../ui";
import MarkdownEditor from "../MarkdownEditor";
import { isEditableTarget } from "../../utils/keyboard";
import { linkifyIssueIds } from "../../utils/format";
import { STATUS_OPTIONS, ESTIMATE_OPTIONS, PRIORITY_OPTIONS } from "../../constants";
import SubIssuesTable from "./SubIssuesTable";
import ActivityTimeline from "./ActivityTimeline";
import PropertySidebar from "./PropertySidebar";
import MergeView from "./MergeView";

interface IssueDetailProps {
  issue: Issue;
  issues: Issue[];
  currentIndex: number;
  totalCount: number;
  onClose: () => void;
  onNavigate: (direction: "prev" | "next") => void;
  onRefresh: () => void;
  prefix: string;
  contributors: string[];
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
  contributors,
  onConfigLabelsChange,
}: IssueDetailProps) {
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [optimisticTitle, setOptimisticTitle] = useState<string | null>(null);
  const [optimisticDescription, setOptimisticDescription] = useState<string | null>(null);
  const [descClickEvent, setDescClickEvent] = useState<{ clientX: number; clientY: number } | null>(null);
  const [newComment, setNewComment] = useState("");
  const [saving, setSaving] = useState(false);
  const [openPopover, setOpenPopover] = useState<string | null>(null);
  const [popoverIndex, setPopoverIndex] = useState(0);
  const [mergeViewOpen, setMergeViewOpen] = useState(false);
  const showToast = useToast();
  const commentRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setEditingField(null);
    setOpenPopover(null);
    setNewComment("");
    setOptimisticTitle(null);
    setOptimisticDescription(null);
  }, [issue.id]);

  useEffect(() => {
    if (optimisticTitle !== null && issue.title === optimisticTitle) {
      setOptimisticTitle(null);
    }
  }, [issue.title, optimisticTitle]);

  useEffect(() => {
    if (optimisticDescription !== null && issue.description === optimisticDescription) {
      setOptimisticDescription(null);
    }
  }, [issue.description, optimisticDescription]);

  const saveDraft = useCallback(
    async (type: string, payload: unknown) => {
      setSaving(true);
      try {
        await addDraft(issue.id, type, payload);
        onRefresh();
        if (type === "COMMENT") {
          showToast("Comment added");
        } else if (type === "UPDATE") {
          const p = payload as Record<string, unknown>;
          if (p.status) showToast(`Status changed to ${String(p.status)}`);
          else if (p.assignee !== undefined) showToast(p.assignee ? `Assigned to ${String(p.assignee).split(" <")[0]}` : "Assignee removed");
          else if (p.priority !== undefined) showToast("Priority updated");
          else if (p.estimate !== undefined) showToast(`Estimate set to ${p.estimate || "none"}`);
          else if (p.title) showToast("Title updated");
          else if (p.description !== undefined) showToast("Description updated");
          else if (p.labels) showToast("Labels updated");
          else showToast("Updated");
        }
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : "Failed to save";
        showToast(msg, { variant: "error" });
      } finally {
        setSaving(false);
        setEditingField(null);
        setOpenPopover(null);
      }
    },
    [issue.id, onRefresh, showToast],
  );

  const handleStatusChange = useCallback(
    (newStatus: string) => {
      if (newStatus !== issue.status) saveDraft("UPDATE", { status: newStatus });
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
          if (openPopover === "status") handleStatusChange(STATUS_OPTIONS[popoverIndex].value);
          else if (openPopover === "estimate") handleEstimateChange(ESTIMATE_OPTIONS[popoverIndex]);
          else if (openPopover === "priority") handlePriorityChange(PRIORITY_OPTIONS[popoverIndex].value);
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
        setPopoverIndex(STATUS_OPTIONS.findIndex((o) => o.value === issue.status));
      }
      if (e.key === "l") { setOpenPopover("labels"); setPopoverIndex(0); }
      if (e.key === "e") {
        setOpenPopover("estimate");
        setPopoverIndex(ESTIMATE_OPTIONS.indexOf(issue.estimate || 0));
      }
      if (e.key === "p") {
        setOpenPopover("priority");
        setPopoverIndex(PRIORITY_OPTIONS.findIndex((o) => o.value === (issue.priority || 0)));
      }
      if (e.key === "a") { setOpenPopover("assignee"); }
      if (e.key === "m") {
        e.preventDefault();
        commentRef.current?.focus();
        commentRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
      }
      if (e.key === ".") {
        navigator.clipboard.writeText(issue.id);
        showToast("Copied issue ID");
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
    onClose, onNavigate, openPopover, editingField,
    issue.status, issue.estimate, issue.priority, issue.id,
    saveDraft, handleStatusChange, handleEstimateChange, handlePriorityChange,
    popoverIndex,
  ]);

  const startEditing = (field: string) => {
    setEditingField(field);
    if (field === "title") setEditTitle(issue.title);
    if (field === "description") setEditDescription(issue.description || "");
  };

  const handleSaveTitle = () => {
    if (editTitle.trim() && editTitle !== issue.title) {
      setOptimisticTitle(editTitle.trim());
      saveDraft("UPDATE", { title: editTitle.trim() });
    } else {
      setEditingField(null);
    }
  };

  const handleSaveDescription = () => {
    if (editDescription !== (issue.description || "")) {
      setOptimisticDescription(editDescription);
      saveDraft("UPDATE", { description: editDescription });
    } else {
      setEditingField(null);
    }
  };

  const handleAddComment = () => {
    if (newComment.trim()) {
      saveDraft("COMMENT", { id: `c-${Date.now().toString(36)}`, text: newComment.trim() });
      setNewComment("");
    }
  };

  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < totalCount - 1;

  if (mergeViewOpen) {
    return (
      <MergeView
        issue={issue}
        onClose={() => setMergeViewOpen(false)}
        onMerged={() => { setMergeViewOpen(false); onRefresh(); showToast("Branch merged"); }}
      />
    );
  }

  return (
    <div className="h-full flex flex-col relative">
      {/* Top bar */}
      <div className="flex items-center justify-between px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <div className="flex items-center gap-2 text-sm min-w-0">
          <button onClick={onClose} className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors shrink-0">
            Issues
          </button>
          <ChevronRightIcon className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
          <CopyableId id={issue.id} className="text-xs shrink-0" />
          <span className="text-[var(--color-text-primary)] truncate">{optimisticTitle ?? issue.title}</span>
        </div>
        <div className="flex items-center gap-1 shrink-0 ml-4">
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums mr-1">{currentIndex + 1} / {totalCount}</span>
          <button onClick={() => onNavigate("prev")} disabled={!hasPrev} className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors disabled:opacity-20 disabled:pointer-events-none">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <button onClick={() => onNavigate("next")} disabled={!hasNext} className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors disabled:opacity-20 disabled:pointer-events-none">
            <ChevronRightIcon />
          </button>
        </div>
      </div>

      {/* Body */}
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
                onKeyDown={(e) => { if (e.key === "Enter") handleSaveTitle(); if (e.key === "Escape") setEditingField(null); }}
                onBlur={handleSaveTitle}
                className="w-full text-xl font-semibold bg-transparent text-[var(--color-text-primary)] outline-none border-none m-0 p-0 leading-tight block"
                style={{ caretColor: "var(--color-accent-primary)", height: "auto" }}
              />
            ) : (
              <h1 onClick={() => startEditing("title")} className="text-xl font-semibold text-[var(--color-text-primary)] m-0 p-0 leading-tight cursor-text">
                {optimisticTitle ?? issue.title}
              </h1>
            )}

            {/* Parent reference */}
            {issue.parent_id && (() => {
              const parent = issues.find(i => i.id === issue.parent_id);
              const siblings = parent ? issues.filter(i => i.parent_id === parent.id) : [];
              const siblingsDone = siblings.filter(i => i.status === 'DONE').length;
              return (
                <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)] mt-2 flex-wrap">
                  <span>Sub-issue of</span>
                  {parent && <StatusIcon status={parent.status} size={14} isInferred={parent.is_inferred} />}
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
                  className="prose-exponential"
                />
              ) : (
                <div
                  onClick={(e) => {
                    if ((e.target as HTMLElement).closest('a')) return;
                    setDescClickEvent({ clientX: e.clientX, clientY: e.clientY }); startEditing("description");
                  }}
                  className="cursor-text min-h-10 prose-exponential"
                >
                  {(optimisticDescription ?? issue.description) ? (
                    <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                      {linkifyIssueIds(optimisticDescription ?? issue.description ?? "", prefix)}
                    </Markdown>
                  ) : (
                    <p className="text-base text-[var(--color-text-muted)]">Add a description...</p>
                  )}
                </div>
              )}
            </div>

            <SubIssuesTable issue={issue} issues={issues} onRefresh={onRefresh} />

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

        <PropertySidebar
          issue={issue}
          issues={issues}
          openPopover={openPopover}
          setOpenPopover={setOpenPopover}
          popoverIndex={popoverIndex}
          setPopoverIndex={setPopoverIndex}
          saveDraft={saveDraft}
          onClose={onClose}
          onRefresh={onRefresh}
          contributors={contributors}
          onConfigLabelsChange={onConfigLabelsChange}
          onOpenMerge={() => setMergeViewOpen(true)}
        />
      </div>
    </div>
  );
}
