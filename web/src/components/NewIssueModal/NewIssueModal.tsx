import { useState, useRef, useEffect, useMemo, useCallback } from "react";
import { createPortal } from "react-dom";
import { createIssue, addDraft, ApiError } from "../../api/client";
import type { Issue } from "../../api/client";
import { computeAppendKey } from "../../utils/sort";
import { Modal, Button, LabelBadge, LabelPicker, Avatar, Popover, StatusIcon, InlineDropdown } from "../ui";
import { ChevronDown, Folder, UserRound, Tag } from "lucide-react";
import type { DropdownOption } from "../ui";
import { isTerminal } from "../../constants";
import MarkdownEditor from "../MarkdownEditor";
import { useAllLabels } from "../../hooks/useLabels";

const STATUS_OPTIONS: DropdownOption[] = [
  { value: "BACKLOG", label: "Backlog", icon: <StatusIcon status="BACKLOG" size={14} /> },
  { value: "PLANNED", label: "Planned", icon: <StatusIcon status="PLANNED" size={14} /> },
  { value: "DOING", label: "In Progress", icon: <StatusIcon status="DOING" size={14} /> },
];

const ESTIMATE_OPTIONS: DropdownOption[] = [
  { value: "0", label: "No estimate" },
  { value: "1", label: "1 Point" },
  { value: "2", label: "2 Points" },
  { value: "3", label: "3 Points" },
  { value: "5", label: "5 Points" },
  { value: "8", label: "8 Points" },
];

interface NewIssueModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated: () => void;
  issues: Issue[];
  contributors: string[];
  onConfigLabelsChange: (labels: Record<string, string>) => void;
}

export default function NewIssueModal({ isOpen, onClose, onCreated, issues, contributors, onConfigLabelsChange }: NewIssueModalProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [labels, setLabels] = useState<string[]>(["feature"]);
  const [status, setStatus] = useState("BACKLOG");
  const [estimate, setEstimate] = useState("0");
  const [saving, setSaving] = useState(false);
  const [labelOpen, setLabelOpen] = useState(false);
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const labelBtnRef = useRef<HTMLButtonElement>(null);
  const labelMenuRef = useRef<HTMLDivElement>(null);
  const [labelPos, setLabelPos] = useState({ top: 0, left: 0 });

  const [moreOpen, setMoreOpen] = useState(false);
  const [parentId, setParentId] = useState("");
  const [assignee, setAssignee] = useState("");
  const [additionalLabels, setAdditionalLabels] = useState<string[]>([]);
  const [morePopover, setMorePopover] = useState<"parent" | "assignee" | "labels" | null>(null);
  const [parentSearch, setParentSearch] = useState("");
  const [assigneeSearch, setAssigneeSearch] = useState("");

  const allKnownLabels = useAllLabels(issues);

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
    const all = Array.from(byEmail.values()).sort((a, b) => a.split(" <")[0].localeCompare(b.split(" <")[0]));
    const q = assigneeSearch.toLowerCase();
    if (!q) return all;
    return all.filter((p) => p.toLowerCase().includes(q));
  }, [issues, contributors, assigneeSearch]);

  const parentCandidates = useMemo(() => {
    const q = parentSearch.toLowerCase();
    return issues.filter((i) => !i.parent_id && !isTerminal(i.status) && (!q || i.title.toLowerCase().includes(q) || i.id.toLowerCase().includes(q)));
  }, [issues, parentSearch]);

  const selectLabel = (l: string) => { setLabels([l]); setLabelOpen(false); };
  const toggleAdditionalLabel = (l: string) => { setAdditionalLabels((prev) => prev.includes(l) ? prev.filter((x) => x !== l) : [...prev, l]); };

  useEffect(() => {
    if (!labelOpen) return;
    const handler = (e: MouseEvent) => {
      if (labelMenuRef.current && !labelMenuRef.current.contains(e.target as Node) && labelBtnRef.current && !labelBtnRef.current.contains(e.target as Node)) setLabelOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [labelOpen]);

  const handleLabelOpen = () => {
    if (labelBtnRef.current) { const rect = labelBtnRef.current.getBoundingClientRect(); setLabelPos({ top: rect.bottom + 4, left: rect.left }); }
    setLabelOpen(!labelOpen);
  };

  const hasContent = title.trim() !== "" || description.trim() !== "";

  const resetForm = () => {
    setTitle(""); setDescription(""); setLabels(["feature"]); setStatus("BACKLOG"); setEstimate("0"); setParentId(""); setAssignee(""); setAdditionalLabels([]); setMoreOpen(false); setConfirmDiscard(false);
  };

  const handleClose = () => {
    if (confirmDiscard) { setConfirmDiscard(false); return; }
    if (!hasContent) { resetForm(); onClose(); return; }
    setConfirmDiscard(true);
  };

  const handleDiscard = () => { resetForm(); onClose(); };
  const handleCancelDiscard = useCallback(() => setConfirmDiscard(false), []);

  useEffect(() => {
    if (!confirmDiscard) return;
    const trap = (e: KeyboardEvent) => {
      if (e.key === "Escape") { e.stopImmediatePropagation(); e.preventDefault(); setConfirmDiscard(false); }
    };
    document.addEventListener("keydown", trap, true);
    return () => document.removeEventListener("keydown", trap, true);
  }, [confirmDiscard]);

  const canCreate = title.trim() && labels.length > 0;

  const handleCreate = async () => {
    if (!canCreate) return;
    setSaving(true);
    try {
      const combinedLabels = [...labels, ...additionalLabels.filter((l) => !labels.includes(l))];
      const sortOrder = computeAppendKey(issues, status, parentId || undefined);
      const issueId = await createIssue({ title: title.trim(), description: description.trim() || undefined, labels: combinedLabels, parent_id: parentId || undefined, assignee: assignee || undefined, sort_order: sortOrder });
      const update: Record<string, unknown> = {};
      if (status !== "BACKLOG") update.status = status;
      const estimateNum = parseInt(estimate, 10);
      if (estimateNum > 0) update.estimate = estimateNum;
      if (Object.keys(update).length > 0) await addDraft(issueId, "UPDATE", update);
      resetForm();
      onCreated();
    } catch (err) {
      console.error("Failed to create issue:", err instanceof ApiError ? err.message : err);
    } finally {
      setSaving(false);
    }
  };

  const handleModalKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && canCreate) { e.preventDefault(); handleCreate(); }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="New Issue" size="2xl">
      <div className="space-y-4 relative" onKeyDown={handleModalKeyDown}>
        <div className="flex items-center gap-2">
          <div>
            <button ref={labelBtnRef} type="button" onClick={handleLabelOpen} className={`flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm transition-colors border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] ${labels.length === 0 ? "border-[var(--color-error)]/40" : ""}`}>
              {labels[0] ? <LabelBadge label={labels[0]} borderless /> : <span className="text-[var(--color-text-muted)]">Label *</span>}
              <ChevronDown className="w-3 h-3 text-[var(--color-text-muted)]" />
            </button>
            {labelOpen && createPortal(
              <div ref={labelMenuRef} className="fixed z-[100] min-w-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1" style={{ top: labelPos.top, left: labelPos.left }}>
                <LabelPicker allLabels={allKnownLabels} selected={labels} onToggle={selectLabel} onConfigLabelsChange={onConfigLabelsChange} onClose={() => setLabelOpen(false)} singleSelect borderlessBadges />
              </div>,
              document.body,
            )}
          </div>
          <InlineDropdown placeholder="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} />
          <div className="flex-1" />
          <InlineDropdown placeholder="Estimate" options={ESTIMATE_OPTIONS} value={estimate} onChange={setEstimate} />
        </div>
        <input autoFocus value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Issue title *" className="w-full h-10 text-lg bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none border-none p-0" onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey && canCreate) handleCreate(); }} />
        <MarkdownEditor value={description} onChange={setDescription} placeholder="Add description (markdown supported)..." className="prose-exponential min-h-50" />
        <div className="border-t border-[var(--color-border-subtle)] pt-3">
          <button type="button" onClick={() => setMoreOpen((v) => !v)} className="flex items-center gap-2 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors">
            <svg className={`w-3 h-3 transition-transform duration-100 ${moreOpen ? "rotate-90" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" /></svg>
            More options
          </button>
          {moreOpen && (
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <div className="relative">
                <button type="button" onClick={() => { setMorePopover(morePopover === "parent" ? null : "parent"); setParentSearch(""); }} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                  <Folder className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  <span className={`truncate max-w-50 ${parentId ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)]"}`}>{parentId ? (issues.find((i) => i.id === parentId)?.title || parentId) : "No parent"}</span>
                </button>
                {morePopover === "parent" && (
                  <Popover onClose={() => setMorePopover(null)}>
                    <div className="px-3 py-2"><input autoFocus value={parentSearch} onChange={(e) => setParentSearch(e.target.value)} placeholder="Search issues..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" /></div>
                    <div className="border-t border-[var(--color-border-default)]" />
                    <div className="max-h-60 overflow-y-auto">
                      {parentId && <button onClick={() => { setParentId(""); setMorePopover(null); }} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">Remove parent</button>}
                      {parentCandidates.slice(0, 15).map((c) => (<button key={c.id} onClick={() => { setParentId(c.id); setMorePopover(null); }} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${c.id === parentId ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}><StatusIcon status={c.status} size={12} /><span className="truncate">{c.title}</span>{c.id === parentId && <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}</button>))}
                      {parentCandidates.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching issues</div>}
                    </div>
                  </Popover>
                )}
              </div>
              <div className="relative">
                <button type="button" onClick={() => { setMorePopover(morePopover === "assignee" ? null : "assignee"); setAssigneeSearch(""); }} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                  {assignee ? <Avatar name={assignee} size="xs" /> : <UserRound className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />}
                  <span className={`truncate max-w-40 ${assignee ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)]"}`}>{assignee ? assignee.split(" <")[0] : "No assignee"}</span>
                </button>
                {morePopover === "assignee" && (
                  <Popover onClose={() => setMorePopover(null)}>
                    <div className="px-3 py-2"><input autoFocus value={assigneeSearch} onChange={(e) => setAssigneeSearch(e.target.value)} placeholder="Search people..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" /></div>
                    <div className="border-t border-[var(--color-border-default)]" />
                    <div className="max-h-60 overflow-y-auto">
                      {assignee && <button onClick={() => { setAssignee(""); setMorePopover(null); }} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">Remove assignee</button>}
                      {knownPeople.map((p) => (<button key={p} onClick={() => { setAssignee(p); setMorePopover(null); }} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${p === assignee ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}><Avatar name={p} size="sm" /><span className="truncate">{p.split(" <")[0]}</span>{p === assignee && <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}</button>))}
                      {knownPeople.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching people</div>}
                    </div>
                  </Popover>
                )}
              </div>
              <div className="relative">
                <button type="button" onClick={() => setMorePopover(morePopover === "labels" ? null : "labels")} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                  <Tag className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                  {additionalLabels.length > 0 ? <span className="flex items-center gap-2">{additionalLabels.map((l) => <LabelBadge key={l} label={l} borderless />)}</span> : <span className="text-[var(--color-text-muted)]">Additional labels</span>}
                </button>
                {morePopover === "labels" && (
                  <Popover onClose={() => setMorePopover(null)}>
                    <LabelPicker allLabels={allKnownLabels} selected={additionalLabels} onToggle={toggleAdditionalLabel} onConfigLabelsChange={onConfigLabelsChange} onClose={() => setMorePopover(null)} exclude={labels} borderlessBadges />
                  </Popover>
                )}
              </div>
            </div>
          )}
        </div>
        <div className="flex items-center justify-end gap-2 pt-2">
          <Button variant="ghost" size="sm" onClick={handleClose}>Cancel</Button>
          <Button size="sm" onClick={handleCreate} disabled={!canCreate || saving} loading={saving}>Create Issue</Button>
        </div>
        {confirmDiscard && createPortal(
          <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-black/30" onClick={handleCancelDiscard} aria-hidden="true" />
            <div role="alertdialog" aria-labelledby="discard-title" aria-describedby="discard-desc" className="relative bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] p-5 max-w-sm w-full">
              <h3 id="discard-title" className="text-sm font-semibold text-[var(--color-text-primary)]">Discard this issue?</h3>
              <p id="discard-desc" className="text-sm text-[var(--color-text-secondary)] mt-3">Your issue draft has unsaved changes.</p>
              <div className="flex items-center justify-end gap-2 mt-4">
                <Button variant="ghost" size="sm" onClick={handleCancelDiscard}>Cancel</Button>
                <Button variant="danger" size="sm" onClick={handleDiscard}>Discard</Button>
              </div>
            </div>
          </div>,
          document.body,
        )}
      </div>
    </Modal>
  );
}

