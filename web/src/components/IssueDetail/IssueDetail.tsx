import { useState, useRef, useEffect, useMemo, useCallback } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import { addDraft, fetchIssueHistory } from "../../api/client";
import type { Issue, HistoryEvent } from "../../api/client";
import { LabelBadge, StatusIcon } from "../ui";

interface IssueDetailProps {
  issue: Issue;
  issues: Issue[];
  currentIndex: number;
  onClose: () => void;
  onNavigate: (direction: "prev" | "next") => void;
  onRefresh: () => void;
}

const STATUS_OPTIONS = [
  { value: "BACKLOG", label: "Backlog" },
  { value: "PLANNED", label: "Planned" },
  { value: "DOING", label: "In Progress" },
  { value: "BLOCKED", label: "Blocked" },
  { value: "DONE", label: "Done" },
];

const ESTIMATE_OPTIONS = [0, 1, 2, 3, 5, 8];

const BUILTIN_LABELS = [
  "bug",
  "feature",
  "epic",
  "improvement",
  "UI",
  "refactor",
];

export default function IssueDetail({
  issue,
  issues,
  currentIndex,
  onClose,
  onNavigate,
  onRefresh,
}: IssueDetailProps) {
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [newComment, setNewComment] = useState("");
  const [saving, setSaving] = useState(false);
  const [openPopover, setOpenPopover] = useState<string | null>(null);
  const [popoverIndex, setPopoverIndex] = useState(0);
  const commentRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setEditingField(null);
    setOpenPopover(null);
    setNewComment("");
  }, [issue.id]);

  const saveDraft = useCallback(async (type: string, payload: unknown) => {
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
  }, [issue.id, onRefresh]);

  const handleStatusChange = useCallback((newStatus: string) => {
    if (newStatus !== issue.status) saveDraft("UPDATE", { status: newStatus });
    else setOpenPopover(null);
  }, [issue.status, saveDraft]);

  const handleEstimateChange = useCallback((est: number) => {
    if (est !== (issue.estimate || 0)) saveDraft("UPDATE", { estimate: est });
    else setOpenPopover(null);
  }, [issue.estimate, saveDraft]);

  const handleLabelToggle = useCallback((label: string) => {
    const current = issue.labels || [];
    const next = current.includes(label)
      ? current.filter((l) => l !== label)
      : [...current, label];
    saveDraft("UPDATE", { labels: next });
  }, [issue.labels, saveDraft]);

  const allKnownLabels = Array.from(
    new Set([...BUILTIN_LABELS, ...issues.flatMap((i) => i.labels || [])]),
  ).sort();

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement
      )
        return;
      if (e.key === "Escape") {
        if (openPopover) setOpenPopover(null);
        else if (editingField) setEditingField(null);
        else onClose();
        return;
      }
      if (e.metaKey || e.ctrlKey) return;
      // Keyboard nav inside open popovers
      if (openPopover) {
        const len = openPopover === "status" ? STATUS_OPTIONS.length
          : openPopover === "estimate" ? ESTIMATE_OPTIONS.length
          : openPopover === "labels" ? allKnownLabels.length : 0;
        if (e.key === "ArrowDown") { e.preventDefault(); setPopoverIndex(i => Math.min(i + 1, len - 1)); return; }
        if (e.key === "ArrowUp") { e.preventDefault(); setPopoverIndex(i => Math.max(i - 1, 0)); return; }
        if (e.key === "Enter") {
          e.preventDefault();
          if (openPopover === "status") handleStatusChange(STATUS_OPTIONS[popoverIndex].value);
          else if (openPopover === "estimate") handleEstimateChange(ESTIMATE_OPTIONS[popoverIndex]);
          else if (openPopover === "labels") handleLabelToggle(allKnownLabels[popoverIndex]);
          return;
        }
        if (openPopover === "status") {
          const num = parseInt(e.key);
          if (num >= 1 && num <= 5 && STATUS_OPTIONS[num - 1]) { handleStatusChange(STATUS_OPTIONS[num - 1].value); return; }
        }
        return;
      }
      if (e.key === "ArrowLeft" || e.key === "k") onNavigate("prev");
      if (e.key === "ArrowRight" || e.key === "j") onNavigate("next");
      if (e.key === "s") { setOpenPopover("status"); setPopoverIndex(STATUS_OPTIONS.findIndex(o => o.value === issue.status)); }
      if (e.key === "l") { setOpenPopover("labels"); setPopoverIndex(0); }
      if (e.key === "e") { setOpenPopover("estimate"); setPopoverIndex(ESTIMATE_OPTIONS.indexOf(issue.estimate || 0)); }
      if (e.key === "m") { e.preventDefault(); commentRef.current?.focus(); commentRef.current?.scrollIntoView({ behavior: "smooth", block: "center" }); }
      const num = parseInt(e.key);
      if (num >= 1 && num <= 5) {
        const status = STATUS_OPTIONS[num - 1];
        if (status && status.value !== issue.status) saveDraft("UPDATE", { status: status.value });
      }
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose, onNavigate, openPopover, editingField, issue.status, issue.estimate, saveDraft, handleStatusChange, handleEstimateChange, handleLabelToggle, allKnownLabels, popoverIndex]);

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
  const hasNext = currentIndex < issues.length - 1;

  return (
    <div className="h-full flex flex-col">
      {/* Top bar: breadcrumb + nav */}
      <div className="flex items-center justify-between px-5 h-11 border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-secondary)] shrink-0">
        <div className="flex items-center gap-1.5 text-[13px] min-w-0">
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
          <span className="text-[var(--color-text-muted)] font-mono text-[11px] shrink-0">
            {issue.id}
          </span>
          <span className="text-[var(--color-text-primary)] truncate">
            {issue.title}
          </span>
        </div>

        <div className="flex items-center gap-1 shrink-0 ml-4">
          <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums mr-1">
            {currentIndex + 1} / {issues.length}
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
          <div className="max-w-4xl mx-auto px-8 py-6">
            {/* Parent reference */}
            {issue.parent_id && (
              <div className="flex items-center gap-1.5 text-[12px] text-[var(--color-text-muted)] mb-3">
                <span>Sub-issue of</span>
                <span className="font-mono text-[var(--color-accent-primary)]">
                  {issue.parent_id}
                </span>
              </div>
            )}

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
                className="w-full text-[22px] font-semibold bg-transparent text-[var(--color-text-primary)] outline-none border-none mb-1 leading-tight"
                style={{ caretColor: "var(--color-accent-primary)" }}
              />
            ) : (
              <h1
                onClick={() => startEditing("title")}
                className="text-[22px] font-semibold text-[var(--color-text-primary)] mb-1 leading-tight cursor-text"
              >
                {issue.title}
              </h1>
            )}

            {/* Labels row */}
            {issue.labels && issue.labels.length > 0 && (
              <div className="flex items-center gap-3 mt-2 mb-5">
                {issue.labels.map((label) => (
                  <LabelBadge key={label} label={label} />
                ))}
                {issue.is_pending && (
                  <span className="text-[10px] text-[var(--color-warning)]">
                    unsaved changes
                  </span>
                )}
              </div>
            )}

            {/* Description */}
            <div className="mt-4">
              {editingField === "description" ? (
                <div>
                  <textarea
                    autoFocus
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Escape") setEditingField(null);
                    }}
                    rows={8}
                    className="w-full text-[14px] leading-relaxed bg-transparent text-[var(--color-text-secondary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] px-3 py-2.5 outline-none focus:border-[var(--color-border-focus)] resize-y"
                    placeholder="Add a description..."
                  />
                  <div className="flex gap-2 mt-2">
                    <button
                      onClick={handleSaveDescription}
                      disabled={saving}
                      className="px-2.5 py-1 text-[12px] bg-[var(--color-accent-primary)] text-white rounded-[var(--radius-md)] hover:bg-[var(--color-accent-primary-hover)] disabled:opacity-40"
                    >
                      Save
                    </button>
                    <button
                      onClick={() => setEditingField(null)}
                      className="px-2.5 py-1 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <div
                  onClick={() => startEditing("description")}
                  className="cursor-text min-h-[40px] prose-beats"
                >
                  {issue.description ? (
                    <Markdown remarkPlugins={[remarkBreaks]}>
                      {issue.description}
                    </Markdown>
                  ) : (
                    <p className="text-[14px] text-[var(--color-text-muted)]">
                      Add a description...
                    </p>
                  )}
                </div>
              )}
            </div>

            {/* Activity */}
            <ActivityTimeline
              issue={issue}
              newComment={newComment}
              onNewCommentChange={setNewComment}
              onAddComment={handleAddComment}
              saving={saving}
              commentRef={commentRef}
            />
          </div>
        </div>

        {/* Properties sidebar */}
        <div className="w-80 border-l border-[var(--color-border-subtle)] overflow-y-auto shrink-0">
          <div className="p-4 space-y-5">
            {/* Status */}
            <PropertyRow label="Status">
              <div className="relative">
                <button
                  onClick={() =>
                    setOpenPopover(openPopover === "status" ? null : "status")
                  }
                  className="flex items-center gap-2 px-1.5 py-1 rounded-[var(--radius-sm)] hover:bg-[var(--color-bg-hover)] transition-colors w-full text-left"
                >
                  <StatusIcon status={issue.status} size={14} />
                  <span className="text-[13px] text-[var(--color-text-primary)]">
                    {statusMeta?.label || issue.status}
                  </span>
                </button>
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
                          className={`flex items-center gap-2 w-full px-3 py-1.5 text-[13px] transition-colors ${isFocused ? 'bg-[var(--color-bg-hover)]' : ''} ${isCurrent ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}
                        >
                          <StatusIcon status={opt.value} size={14} />
                          <span>{opt.label}</span>
                          {isCurrent ? (
                            <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          ) : (
                            <span className="ml-auto text-[11px] text-[var(--color-text-muted)]">{i + 1}</span>
                          )}
                        </button>
                      );
                    })}
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Estimate */}
            <PropertyRow label="Estimate">
              <div className="relative">
                <button
                  onClick={() =>
                    setOpenPopover(
                      openPopover === "estimate" ? null : "estimate",
                    )
                  }
                  className="flex items-center gap-2 px-1.5 py-1 rounded-[var(--radius-sm)] hover:bg-[var(--color-bg-hover)] transition-colors w-full text-left"
                >
                  <EstimateIcon />
                  <span className="text-[13px] text-[var(--color-text-primary)]">
                    {issue.estimate
                      ? `${issue.estimate} Point${issue.estimate !== 1 ? "s" : ""}`
                      : "No estimate"}
                  </span>
                </button>
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
                          className={`flex items-center gap-2 w-full px-3 py-1.5 text-[13px] transition-colors ${isFocused ? 'bg-[var(--color-bg-hover)]' : ''} ${isCurrent ? 'text-[var(--color-accent-primary)]' : 'text-[var(--color-text-primary)]'}`}
                        >
                          <span>{est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`}</span>
                          {isCurrent && (
                            <svg className="w-3.5 h-3.5 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
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

            {/* Labels */}
            <PropertyRow label="Labels">
              <div className="relative">
                <button
                  onClick={() =>
                    setOpenPopover(openPopover === "labels" ? null : "labels")
                  }
                  className="flex items-center gap-2 flex-wrap px-1.5 py-1 rounded-[var(--radius-sm)] hover:bg-[var(--color-bg-hover)] transition-colors w-full text-left"
                >
                  {issue.labels && issue.labels.length > 0 ? (
                    issue.labels.map((label) => (
                      <LabelBadge key={label} label={label} />
                    ))
                  ) : (
                    <span className="text-[13px] text-[var(--color-text-muted)]">
                      No labels
                    </span>
                  )}
                </button>
                {openPopover === "labels" && (
                  <Popover onClose={() => setOpenPopover(null)}>
                    <PopoverHeader>Change or add labels...</PopoverHeader>
                    {allKnownLabels.map((label, i) => {
                      const isActive = (issue.labels || []).includes(label);
                      const isFocused = i === popoverIndex;
                      return (
                        <button
                          key={label}
                          onClick={() => handleLabelToggle(label)}
                          onMouseEnter={() => setPopoverIndex(i)}
                          className={`flex items-center gap-2 w-full px-3 py-1.5 text-[13px] text-[var(--color-text-primary)] transition-colors ${isFocused ? 'bg-[var(--color-bg-hover)]' : ''}`}
                        >
                          <span
                            className={`w-3.5 h-3.5 rounded-[3px] border flex items-center justify-center shrink-0 ${isActive ? "bg-[var(--color-accent-primary)] border-[var(--color-accent-primary)]" : "border-[var(--color-border-default)]"}`}
                          >
                            {isActive && (
                              <svg
                                className="w-2.5 h-2.5 text-white"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                                strokeWidth={3}
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M5 13l4 4L19 7"
                                />
                              </svg>
                            )}
                          </span>
                          <LabelBadge label={label} />
                        </button>
                      );
                    })}
                  </Popover>
                )}
              </div>
            </PropertyRow>

            {/* Assignee */}
            {issue.assignee && (
              <PropertyRow label="Assignee">
                <div className="flex items-center gap-2 px-1.5 py-1">
                  <div className="w-5 h-5 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center text-[9px] font-medium text-[var(--color-text-muted)]">
                    {issue.assignee.charAt(0).toUpperCase()}
                  </div>
                  <span className="text-[13px] text-[var(--color-text-primary)]">
                    {issue.assignee}
                  </span>
                </div>
              </PropertyRow>
            )}

            {/* Dependencies */}
            {issue.dependencies && issue.dependencies.length > 0 && (
              <PropertyRow label="Relations">
                <div className="space-y-1 px-1.5">
                  {issue.dependencies.map((dep, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-1.5 text-[13px]"
                    >
                      <span className="text-[var(--color-text-muted)]">
                        {dep.kind.replace("_", " ")}
                      </span>
                      <span className="font-mono text-[var(--color-accent-primary)]">
                        {dep.target_id}
                      </span>
                    </div>
                  ))}
                </div>
              </PropertyRow>
            )}

            {/* Metadata */}
            <div className="pt-4 border-t border-[var(--color-border-subtle)] space-y-2.5">
              <MetaRow
                label="Created"
                value={formatRelativeTime(issue.created_at)}
              />
              <MetaRow
                label="Updated"
                value={formatRelativeTime(issue.updated_at)}
              />
              {issue.created_by && (
                <MetaRow
                  label="Created by"
                  value={issue.created_by.split(" <")[0]}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function PropertyRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="text-[11px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-1.5 px-1.5">
        {label}
      </div>
      {children}
    </div>
  );
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between text-[13px] px-1.5">
      <span className="text-[var(--color-text-muted)]">{label}</span>
      <span className="text-[var(--color-text-secondary)]">{value}</span>
    </div>
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

function Popover({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose]);

  return (
    <div
      ref={ref}
      className="absolute left-0 top-full mt-1 z-50 min-w-[200px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
    >
      {children}
    </div>
  );
}

function PopoverHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-3 py-1.5 text-[11px] text-[var(--color-text-muted)]">
      {children}
    </div>
  );
}



type ActivityEntry =
  | { kind: "system"; author: string; content: React.ReactNode; time: string }
  | { kind: "comment"; author: string; text: string; time: string };

const STATUS_LABELS: Record<string, string> = {
  BACKLOG: "Backlog", PLANNED: "Planned", DOING: "In Progress", BLOCKED: "Blocked", DONE: "Done",
};

function StatusChip({ status }: { status: string }) {
  return (
    <span className="flex items-center gap-1">
      <StatusIcon status={status} size={12} />
      <span className="font-medium text-[var(--color-text-primary)]">{STATUS_LABELS[status] || status}</span>
    </span>
  );
}

function EstimateChip({ points }: { points: number }) {
  return (
    <span className="flex items-center gap-1 font-medium text-[var(--color-text-primary)]">
      <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>
      {points} {points === 1 ? "Point" : "Points"}
    </span>
  );
}

function describeEvent(evt: HistoryEvent): React.ReactNode | null {
  const p = evt.payload || {};
  switch (evt.type) {
    case "CREATE": return "created this issue";
    case "UPDATE": {
      const fragments: React.ReactNode[] = [];
      if (p.status) fragments.push(<>changed status to <StatusChip status={String(p.status)} /></>);
      if (p.estimate !== undefined) fragments.push(<>set estimate to <EstimateChip points={Number(p.estimate)} /></>);
      if (p.title) fragments.push(<>updated the title</>);
      if (p.description !== undefined) fragments.push(<>updated the description</>);
      if (p.labels) fragments.push(<>updated labels to {(p.labels as string[]).map(l => <LabelBadge key={l} label={l} />)}</>);
      if (p.assignee) fragments.push(<>assigned to <span className="font-medium text-[var(--color-text-primary)]">{String(p.assignee)}</span></>);
      if (fragments.length === 0) return null;
      return fragments.reduce<React.ReactNode[]>((acc, f, i) => {
        if (i > 0) acc.push(<span key={`sep-${i}`}> and </span>);
        acc.push(f);
        return acc;
      }, []);
    }
    case "DELETE": return "deleted this issue";
    default: return null;
  }
}

function ActivityTimeline({
  issue,
  newComment,
  onNewCommentChange,
  onAddComment,
  saving,
  commentRef,
}: {
  issue: Issue;
  newComment: string;
  onNewCommentChange: (v: string) => void;
  onAddComment: () => void;
  saving: boolean;
  commentRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  const [history, setHistory] = useState<HistoryEvent[]>([]);
  const [sortNewest, setSortNewest] = useState(true);

  useEffect(() => {
    fetchIssueHistory(issue.id).then(setHistory).catch(() => {});
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
    items.sort((a, b) => dir * (new Date(a.time).getTime() - new Date(b.time).getTime()));
    return items;
  }, [history, sortNewest]);

  return (
    <div className="mt-8 pt-6 border-t border-[var(--color-border-subtle)]">
      <div className="flex items-center justify-between mb-5">
        <h3 className="text-[13px] font-semibold text-[var(--color-text-primary)]">Activity</h3>
        <button
          onClick={() => setSortNewest(!sortNewest)}
          className="flex items-center gap-1 text-[11px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          title={sortNewest ? "Showing newest first" : "Showing oldest first"}
        >
          {sortNewest ? "Newest" : "Oldest"}
          <svg className={`w-3 h-3 transition-transform ${sortNewest ? "" : "rotate-180"}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </div>

      <div className="space-y-4">
        {entries.map((entry, i) => {
          if (entry.kind === "system") {
            return (
              <div key={`sys-${i}`} className="flex items-center gap-2 px-3.5 py-1.5 flex-wrap text-[12px] text-[var(--color-text-muted)]">
                <Avatar name={entry.author} />
                <span>{shortName(entry.author)}</span>
                {entry.content}
                <span>·</span>
                <span>{formatRelativeTime(entry.time)}</span>
              </div>
            );
          }

          return (
            <div key={`cmt-${i}`} className="rounded-[var(--radius-lg)] bg-[var(--color-bg-secondary)] py-3 px-3.5">
              <div className="flex items-center gap-2.5 mb-2">
                <Avatar name={entry.author} />
                <span className="text-[12px] font-medium text-[var(--color-text-primary)]">{shortName(entry.author)}</span>
                <span className="text-[12px] text-[var(--color-text-muted)]">{formatRelativeTime(entry.time)}</span>
              </div>
              <div className="prose-beats text-[15px]">
                <Markdown remarkPlugins={[remarkBreaks]}>{entry.text}</Markdown>
              </div>
            </div>
          );
        })}
      </div>

      {/* Comment input */}
      <div className="mt-5 rounded-[var(--radius-lg)] bg-[var(--color-bg-secondary)] overflow-hidden">
        <textarea
          ref={commentRef}
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder="Leave a comment..."
          rows={3}
          className="w-full text-[14px] bg-transparent text-[var(--color-text-primary)] px-3.5 py-3 outline-none placeholder:text-[var(--color-text-muted)] resize-none leading-relaxed"
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
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 10.5L12 3m0 0l7.5 7.5M12 3v18" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

function Avatar({ name }: { name: string }) {
  const [imgFailed, setImgFailed] = useState(false);
  const emailMatch = name.match(/<([^>]+)>/);
  const email = emailMatch ? emailMatch[1].toLowerCase().trim() : null;
  const gravatarUrl = email ? `https://www.gravatar.com/avatar/${md5(email)}?s=48&d=404` : null;

  if (gravatarUrl && !imgFailed) {
    return (
      <img
        src={gravatarUrl}
        onError={() => setImgFailed(true)}
        className="w-6 h-6 rounded-full shrink-0"
        alt=""
      />
    );
  }

  return (
    <div className="w-6 h-6 rounded-full bg-[var(--color-bg-tertiary)] flex items-center justify-center text-[10px] font-medium text-[var(--color-text-muted)] shrink-0">
      {name.charAt(0).toUpperCase()}
    </div>
  );
}

function md5(input: string): string {
  let hash = 0x67452301;
  let a = 0xefcdab89;
  let b = 0x98badcfe;
  let c = 0x10325476;
  const bytes: number[] = [];
  for (let i = 0; i < input.length; i++) {
    bytes.push(input.charCodeAt(i) & 0xff);
  }
  bytes.push(0x80);
  while (bytes.length % 64 !== 56) bytes.push(0);
  const bitLen = input.length * 8;
  bytes.push(bitLen & 0xff, (bitLen >> 8) & 0xff, (bitLen >> 16) & 0xff, (bitLen >> 24) & 0xff, 0, 0, 0, 0);

  const rl = (v: number, s: number) => (v << s) | (v >>> (32 - s));

  const K = [
    0xd76aa478,0xe8c7b756,0x242070db,0xc1bdceee,0xf57c0faf,0x4787c62a,0xa8304613,0xfd469501,
    0x698098d8,0x8b44f7af,0xffff5bb1,0x895cd7be,0x6b901122,0xfd987193,0xa679438e,0x49b40821,
    0xf61e2562,0xc040b340,0x265e5a51,0xe9b6c7aa,0xd62f105d,0x02441453,0xd8a1e681,0xe7d3fbc8,
    0x21e1cde6,0xc33707d6,0xf4d50d87,0x455a14ed,0xa9e3e905,0xfcefa3f8,0x676f02d9,0x8d2a4c8a,
    0xfffa3942,0x8771f681,0x6d9d6122,0xfde5380c,0xa4beea44,0x4bdecfa9,0xf6bb4b60,0xbebfbc70,
    0x289b7ec6,0xeaa127fa,0xd4ef3085,0x04881d05,0xd9d4d039,0xe6db99e5,0x1fa27cf8,0xc4ac5665,
    0xf4292244,0x432aff97,0xab9423a7,0xfc93a039,0x655b59c3,0x8f0ccc92,0xffeff47d,0x85845dd1,
    0x6fa87e4f,0xfe2ce6e0,0xa3014314,0x4e0811a1,0xf7537e82,0xbd3af235,0x2ad7d2bb,0xeb86d391,
  ];
  const S = [7,12,17,22,7,12,17,22,7,12,17,22,7,12,17,22,5,9,14,20,5,9,14,20,5,9,14,20,5,9,14,20,4,11,16,23,4,11,16,23,4,11,16,23,4,11,16,23,6,10,15,21,6,10,15,21,6,10,15,21,6,10,15,21];

  for (let off = 0; off < bytes.length; off += 64) {
    const M: number[] = [];
    for (let j = 0; j < 16; j++) {
      M[j] = bytes[off+j*4] | (bytes[off+j*4+1]<<8) | (bytes[off+j*4+2]<<16) | (bytes[off+j*4+3]<<24);
    }
    let aa = hash, bb = a, cc = b, dd = c;
    for (let i = 0; i < 64; i++) {
      let f: number, g: number;
      if (i < 16) { f = (bb & cc) | (~bb & dd); g = i; }
      else if (i < 32) { f = (dd & bb) | (~dd & cc); g = (5*i+1)%16; }
      else if (i < 48) { f = bb ^ cc ^ dd; g = (3*i+5)%16; }
      else { f = cc ^ (bb | ~dd); g = (7*i)%16; }
      const tmp = dd;
      dd = cc; cc = bb;
      bb = (bb + rl((aa + f + K[i] + M[g]) | 0, S[i])) | 0;
      aa = tmp;
    }
    hash = (hash + aa) | 0; a = (a + bb) | 0; b = (b + cc) | 0; c = (c + dd) | 0;
  }

  const hex = (v: number) => {
    let s = '';
    for (let i = 0; i < 4; i++) s += ((v >> (i*8)) & 0xff).toString(16).padStart(2, '0');
    return s;
  };
  return hex(hash) + hex(a) + hex(b) + hex(c);
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
