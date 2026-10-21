import { useId, useState, useRef, useEffect, useMemo } from "react";
import { createPortal } from "react-dom";
import { useCreateIssue, useCreateIssueFromDraft, useDeleteIssueDraft, useSaveIssueDraft } from "../../api/mutations";
import type { Issue, IssueDraft } from "../../api/client";
import { computeAppendKey } from "../../utils/sort";
import { Dialog, DialogOverlay, Button, ConfirmDialog, LabelBadge, LabelPicker, Avatar, Popover, PopoverPanel, StatusIcon, InlineDropdown } from "../ui";
import { ChevronDown, Folder, UserRound, Tag, Package } from "lucide-react";
import { useReleases } from "../../api/queries";
import { pickerOptions } from "../Releases/release-assign-utils";
import ReleasePicker from "../Releases/ReleasePicker";
import { releaseDropRef } from "../Releases/releases-utils";
import type { DropdownOption } from "../ui";
import { fieldTriggerClass } from "../ui/field-trigger";
import { cn } from "../../utils/cn";
import { isTerminal } from "../../constants";
import { collectKnownPeople } from "../../utils/issues";
import MarkdownEditor from "../MarkdownEditor";
import { useAllLabels } from "../../hooks/useLabels";
import { EMPTY_ISSUE_FORM, draftPayloadFromForm, formFromDraftPayload, formHasContent, formsEqual, shownRelease, type IssueForm } from "./issue-form";

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
  /** The issue draft to edit. The form state is read once, so remount to switch drafts. */
  draft?: IssueDraft | null;
  onClose: () => void;
  /** `draftDeleted` is false when the issue exists but its draft could not be deleted. */
  onCreated: (result: { draftDeleted: boolean }) => void;
  onDraftSaved: () => void;
  issues: Issue[];
  contributors: string[];
}

export default function NewIssueModal({ isOpen, draft = null, onClose, onCreated, onDraftSaved, issues, contributors }: NewIssueModalProps) {
  const [baseline] = useState<IssueForm>(() => (draft ? formFromDraftPayload(draft.payload) : EMPTY_ISSUE_FORM));
  const [form, setForm] = useState<IssueForm>(baseline);
  const { title, description, labels, status, estimate, parentId, assignee, additionalLabels, release } = form;
  const setField = <K extends keyof IssueForm>(key: K) => (value: IssueForm[K]) => setForm((f) => ({ ...f, [key]: value }));
  const setTitle = setField("title");
  const setDescription = setField("description");
  const setStatus = setField("status");
  const setEstimate = setField("estimate");
  const setParentId = setField("parentId");
  const setAssignee = setField("assignee");
  const setRelease = setField("release");
  const { data: releases } = useReleases();
  const releaseShown = shownRelease(form, issues, releases);

  const { mutate: createIssue, isPending: creatingIssue } = useCreateIssue();
  const { mutate: createFromDraft, isPending: creatingFromDraft } = useCreateIssueFromDraft();
  const { mutate: saveDraft, isPending: savingDraft } = useSaveIssueDraft();
  const { mutate: deleteDraft, isPending: deletingDraft } = useDeleteIssueDraft();
  const saving = creatingIssue || creatingFromDraft;
  const [labelOpen, setLabelOpen] = useState(false);
  const [confirm, setConfirm] = useState<"close" | "delete" | null>(null);
  const labelBtnRef = useRef<HTMLButtonElement>(null);
  const labelMenuRef = useRef<HTMLDivElement>(null);
  const [labelPos, setLabelPos] = useState({ top: 0, left: 0 });

  const [moreOpen, setMoreOpen] = useState(() => !!(baseline.parentId || baseline.assignee || baseline.additionalLabels.length || baseline.release));
  const [morePopover, setMorePopover] = useState<"parent" | "assignee" | "labels" | "release" | null>(null);
  const morePopoverAnchorRef = useRef<HTMLDivElement>(null);
  const [parentSearch, setParentSearch] = useState("");
  const [assigneeSearch, setAssigneeSearch] = useState("");

  const allKnownLabels = useAllLabels(issues);

  const knownPeople = useMemo(() => {
    const all = collectKnownPeople(issues, contributors);
    const q = assigneeSearch.toLowerCase();
    if (!q) return all;
    return all.filter((p) => p.toLowerCase().includes(q));
  }, [issues, contributors, assigneeSearch]);

  const parentCandidates = useMemo(() => {
    const q = parentSearch.toLowerCase();
    return issues.filter((i) => !i.parent_id && !isTerminal(i.status) && (!q || i.title.toLowerCase().includes(q) || i.id.toLowerCase().includes(q)));
  }, [issues, parentSearch]);

  const selectLabel = (l: string) => { setField("labels")([l]); setLabelOpen(false); };
  const toggleAdditionalLabel = (l: string) => {
    setForm((f) => ({ ...f, additionalLabels: f.additionalLabels.includes(l) ? f.additionalLabels.filter((x) => x !== l) : [...f.additionalLabels, l] }));
  };

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

  const hasContent = formHasContent(form);
  // A new form is worth keeping once it has text; a reopened draft once it differs from the saved version.
  const isDirty = draft ? !formsEqual(form, baseline) : hasContent;

  const handleClose = () => {
    if (confirm) { setConfirm(null); return; }
    if (!isDirty) { onClose(); return; }
    setConfirm("close");
  };

  const canCreate = title.trim() && labels.length > 0;
  const canSaveDraft = hasContent && !savingDraft;

  const handleSaveDraft = () => {
    if (!canSaveDraft) return;
    saveDraft({ draftId: draft?.id, payload: draftPayloadFromForm(form) }, { onSuccess: onDraftSaved });
  };

  const handleDeleteDraft = () => {
    if (!draft) return;
    deleteDraft(draft.id, { onSuccess: onClose });
  };

  const handleCreate = () => {
    if (!canCreate) return;
    const combinedLabels = [...labels, ...additionalLabels.filter((l) => !labels.includes(l))];
    const sortOrder = computeAppendKey(issues);
    const update: Partial<Issue> = {};
    if (status !== "BACKLOG") update.status = status;
    const estimateNum = parseInt(estimate, 10);
    if (estimateNum > 0) update.estimate = estimateNum;
    const vars = {
      issue: { title: title.trim(), description: description.trim() || undefined, labels: combinedLabels, parent_id: parentId || undefined, assignee: assignee || undefined, release_id: release || undefined, sort_order: sortOrder },
      update,
    };
    if (draft) createFromDraft({ ...vars, draftId: draft.id }, { onSuccess: onCreated });
    else createIssue(vars, { onSuccess: () => onCreated({ draftDeleted: true }) });
  };

  const dialogId = useId();

  const handleModalKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && canCreate) { e.preventDefault(); handleCreate(); }
  };

  if (!isOpen) return null;
  return (
    <>
      <DialogOverlay scope="new-issue" onClose={handleClose} enabled={!confirm}>
        <Dialog
          id={dialogId}
          title={draft ? "Edit Draft" : "New Issue"}
          size="2xl"
          onClose={handleClose}
          footerLeft={draft ? <Button variant="ghost" size="sm" className="-ml-3" onClick={() => setConfirm("delete")}>Discard draft</Button> : undefined}
          footer={<>
            <Button variant="ghost" size="sm" onClick={handleClose}>Cancel</Button>
            <Button variant="secondary" size="sm" onClick={handleSaveDraft} disabled={!canSaveDraft} loading={savingDraft}>Save draft</Button>
            <Button size="sm" onClick={handleCreate} disabled={!canCreate || saving} loading={saving}>Create Issue</Button>
          </>}
        >
          <div className="space-y-4 relative" onKeyDown={handleModalKeyDown}>
            <div className="flex items-center gap-2">
              <div>
                <button ref={labelBtnRef} type="button" onClick={handleLabelOpen} className={cn("flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm transition-colors", fieldTriggerClass(labels.length === 0))}>
                  {labels[0] ? <LabelBadge label={labels[0]} borderless /> : <span className="text-[var(--color-text-muted)]">Label *</span>}
                  <ChevronDown className="w-3 h-3 text-[var(--color-text-muted)]" />
                </button>
                {labelOpen && createPortal(
                  <div ref={labelMenuRef} className="fixed z-[100] min-w-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1" style={{ top: labelPos.top, left: labelPos.left }}>
                    <LabelPicker allLabels={allKnownLabels} selected={labels} onToggle={selectLabel} canCreateLabels onClose={() => setLabelOpen(false)} singleSelect />
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
                  <div className="relative" ref={morePopover === "parent" ? morePopoverAnchorRef : undefined}>
                    <button type="button" onClick={() => { setMorePopover(morePopover === "parent" ? null : "parent"); setParentSearch(""); }} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                      <Folder className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                      <span className={`truncate max-w-50 ${parentId ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)]"}`}>{parentId ? (issues.find((i) => i.id === parentId)?.title || parentId) : "No parent"}</span>
                    </button>
                    {morePopover === "parent" && (
                      <Popover anchorRef={morePopoverAnchorRef} onClose={() => setMorePopover(null)}>
                        <PopoverPanel>
                          <div className="px-3 py-2"><input autoFocus value={parentSearch} onChange={(e) => setParentSearch(e.target.value)} placeholder="Search issues..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" /></div>
                          <div className="border-t border-[var(--color-border-default)]" />
                          <div className="max-h-60 scroll-stable">
                            {parentId && <button onClick={() => { setParentId(""); setMorePopover(null); }} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">Remove parent</button>}
                            {parentCandidates.slice(0, 15).map((c) => (<button key={c.id} onClick={() => { setParentId(c.id); setMorePopover(null); }} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${c.id === parentId ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}><StatusIcon status={c.status} size={12} /><span className="truncate">{c.title}</span>{c.id === parentId && <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}</button>))}
                            {parentCandidates.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching issues</div>}
                          </div>
                        </PopoverPanel>
                      </Popover>
                    )}
                  </div>
                  <div className="relative" ref={morePopover === "assignee" ? morePopoverAnchorRef : undefined}>
                    <button type="button" onClick={() => { setMorePopover(morePopover === "assignee" ? null : "assignee"); setAssigneeSearch(""); }} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                      {assignee ? <Avatar name={assignee} size="xs" /> : <UserRound className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />}
                      <span className={`truncate max-w-40 ${assignee ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)]"}`}>{assignee ? assignee.split(" <")[0] : "No assignee"}</span>
                    </button>
                    {morePopover === "assignee" && (
                      <Popover anchorRef={morePopoverAnchorRef} onClose={() => setMorePopover(null)}>
                        <PopoverPanel>
                          <div className="px-3 py-2"><input autoFocus value={assigneeSearch} onChange={(e) => setAssigneeSearch(e.target.value)} placeholder="Search people..." className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none" /></div>
                          <div className="border-t border-[var(--color-border-default)]" />
                          <div className="max-h-60 scroll-stable">
                            {assignee && <button onClick={() => { setAssignee(""); setMorePopover(null); }} className="flex items-center gap-2 w-full px-3 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface-3)] transition-colors">Remove assignee</button>}
                            {knownPeople.map((p) => (<button key={p} onClick={() => { setAssignee(p); setMorePopover(null); }} className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${p === assignee ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}><Avatar name={p} size="sm" /><span className="truncate">{p.split(" <")[0]}</span>{p === assignee && <svg className="w-4 h-4 ml-auto shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>}</button>))}
                            {knownPeople.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching people</div>}
                          </div>
                        </PopoverPanel>
                      </Popover>
                    )}
                  </div>
                  {releases && (
                    <div className="relative" ref={morePopover === "release" ? morePopoverAnchorRef : undefined}>
                      <button type="button" onClick={() => setMorePopover(morePopover === "release" ? null : "release")} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                        <Package className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                        <span className={`truncate max-w-40 ${releaseShown ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)]"}`}>{releaseShown?.name ?? "No release"}</span>
                      </button>
                      {morePopover === "release" && (
                        <Popover anchorRef={morePopoverAnchorRef} onClose={() => setMorePopover(null)}>
                          <PopoverPanel>
                            <ReleasePicker
                              options={pickerOptions(releases)}
                              current={releaseShown?.id}
                              onSelect={(r) => setRelease(r ? releaseDropRef(r) : "none")}
                              onClose={() => setMorePopover(null)}
                            />
                          </PopoverPanel>
                        </Popover>
                      )}
                    </div>
                  )}
                  <div className="relative" ref={morePopover === "labels" ? morePopoverAnchorRef : undefined}>
                    <button type="button" onClick={() => setMorePopover(morePopover === "labels" ? null : "labels")} className="flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] transition-colors">
                      <Tag className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
                      {additionalLabels.length > 0 ? <span className="flex items-center gap-2">{additionalLabels.map((l) => <LabelBadge key={l} label={l} borderless />)}</span> : <span className="text-[var(--color-text-muted)]">Additional labels</span>}
                    </button>
                    {morePopover === "labels" && (
                      <Popover anchorRef={morePopoverAnchorRef} onClose={() => setMorePopover(null)}>
                        <PopoverPanel>
                          <LabelPicker allLabels={allKnownLabels} selected={additionalLabels} onToggle={toggleAdditionalLabel} canCreateLabels onClose={() => setMorePopover(null)} exclude={labels} />
                        </PopoverPanel>
                      </Popover>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </Dialog>
      </DialogOverlay>
      {confirm === "close" && (
        <ConfirmDialog
          scope="new-issue.confirm-discard"
          title={draft ? "Discard changes?" : "Discard this issue?"}
          description={draft ? "Your changes to this draft are not saved." : "Your issue has unsaved changes."}
          onCancel={() => setConfirm(null)}
        >
          <Button variant="secondary" size="sm" onClick={() => setConfirm(null)}>Cancel</Button>
          <Button variant="secondary" size="sm" onClick={handleSaveDraft} disabled={!canSaveDraft} loading={savingDraft}>Save draft</Button>
          <Button variant="danger" size="sm" onClick={onClose}>Discard</Button>
        </ConfirmDialog>
      )}
      {confirm === "delete" && (
        <ConfirmDialog
          scope="new-issue.confirm-delete-draft"
          title="Discard this draft?"
          description="The draft will be deleted. This cannot be undone."
          onCancel={() => setConfirm(null)}
        >
          <Button variant="secondary" size="sm" onClick={() => setConfirm(null)}>Cancel</Button>
          <Button variant="danger" size="sm" onClick={handleDeleteDraft} loading={deletingDraft}>Discard draft</Button>
        </ConfirmDialog>
      )}
    </>
  );
}
