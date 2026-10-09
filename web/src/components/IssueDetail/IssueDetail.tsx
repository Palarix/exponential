import { useState, useRef, useEffect, useCallback } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import type { Issue } from "../../api/client";
import { useArtifact, useLocalWorktree } from "../../api/queries";
import { useAddComment, useUpdateIssues, type IssuePatch } from "../../api/mutations";
import { StatusIcon, CopyableId, useToast, TopBar, Text, VSCodeIcon, ViewContainer } from "../ui";
import Tooltip from "../ui/Tooltip";
import { iconButtonClass } from "../ui/icon-button-utils";
import { ChevronRight } from "lucide-react";
import MarkdownEditor from "../MarkdownEditor";
import { isEditableTarget } from "../../utils/keyboard";
import { vscodeUrl } from "../../utils/editor";
import { linkifyIssueIds } from "../../utils/format";
import { useKeyboardHandler } from "../../keyboard";
import {
  STATUS_OPTIONS,
  ESTIMATE_OPTIONS,
  PRIORITY_OPTIONS,
} from "../../constants";
import SubIssuesTable from "./SubIssuesTable";
import ArtifactList from "./ArtifactList";
import ActivityTimeline from "./ActivityTimeline";
import PropertySidebar from "./PropertySidebar";
import MergeView from "./MergeView";

type DetailTab = "details" | "spec" | "walkthrough";

function updateToast(p: Partial<Issue>): string {
  if (p.status) return `Status changed to ${p.status}`;
  if (p.assignee !== undefined) return p.assignee ? `Assigned to ${p.assignee.split(" <")[0]}` : "Assignee removed";
  if (p.priority !== undefined) return "Priority updated";
  if (p.estimate !== undefined) return `Estimate set to ${p.estimate || "none"}`;
  if (p.title) return "Title updated";
  if (p.description !== undefined) return "Description updated";
  if (p.labels) return "Labels updated";
  return "Updated";
}

interface IssueDetailProps {
  issue: Issue;
  issues: Issue[];
  currentIndex: number;
  totalCount: number;
  onClose: () => void;
  onNavigate: (direction: "prev" | "next") => void;
  prefix: string;
  contributors: string[];
  banner?: React.ReactNode;
}

export default function IssueDetail({
  issue,
  issues,
  currentIndex,
  totalCount,
  onClose,
  onNavigate,
  prefix,
  contributors,
  banner,
}: IssueDetailProps) {
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [descClickEvent, setDescClickEvent] = useState<{
    clientX: number;
    clientY: number;
  } | null>(null);
  const [newComment, setNewComment] = useState("");
  const [saving, setSaving] = useState(false);
  const [openPopover, setOpenPopover] = useState<string | null>(null);
  const [popoverIndex, setPopoverIndex] = useState(0);
  const [mergeViewOpen, setMergeViewOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<DetailTab>("details");
  const artifactFile = activeTab === "spec" ? "spec.md" : "walkthrough.md";
  const artifact = useArtifact(issue, artifactFile, activeTab !== "details");
  // Re-checked on status changes: `start` creates the worktree, `merge` removes it.
  const worktree = useLocalWorktree(issue);
  const showToast = useToast();
  const commentRef = useRef<HTMLTextAreaElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollTo(0, 0);
  }, [issue.id]);

  // Reset per-issue UI state when navigating to another issue.
  const [shownIssueId, setShownIssueId] = useState(issue.id);
  if (shownIssueId !== issue.id) {
    setShownIssueId(issue.id);
    setEditingField(null);
    setOpenPopover(null);
    setNewComment("");
    setActiveTab("details");
  }

  const { mutate: updateIssues } = useUpdateIssues();
  const { mutate: addComment } = useAddComment();

  const settleSave = useCallback(() => {
    setSaving(false);
    setEditingField(null);
    setOpenPopover(null);
  }, []);

  /** Updates this issue, plus `alsoUpdate` (e.g. children) in the same mutation. */
  const saveUpdate = useCallback(
    (patch: Partial<Issue>, alsoUpdate: IssuePatch[] = []) => {
      setSaving(true);
      updateIssues([{ issueId: issue.id, patch }, ...alsoUpdate], {
        onSuccess: () => showToast(updateToast(patch)),
        onSettled: settleSave,
      });
    },
    [issue.id, updateIssues, settleSave, showToast],
  );

  const saveComment = useCallback(
    (comment: { id: string; text: string }) => {
      setSaving(true);
      addComment({ issueId: issue.id, comment }, {
        onSuccess: () => showToast("Comment added"),
        onSettled: settleSave,
      });
    },
    [issue.id, addComment, settleSave, showToast],
  );

  const handleStatusChange = useCallback(
    (newStatus: string) => {
      if (newStatus !== issue.status)
        saveUpdate({ status: newStatus });
      else setOpenPopover(null);
    },
    [issue.status, saveUpdate],
  );

  const handleEstimateChange = useCallback(
    (est: number) => {
      if (est !== (issue.estimate || 0)) saveUpdate({ estimate: est });
      else setOpenPopover(null);
    },
    [issue.estimate, saveUpdate],
  );

  const handlePriorityChange = useCallback(
    (pri: number) => {
      if (pri !== (issue.priority || 0)) saveUpdate({ priority: pri });
      else setOpenPopover(null);
    },
    [issue.priority, saveUpdate],
  );

  const handleKeyboard = useCallback((e: KeyboardEvent) => {
      if (isEditableTarget(e)) return;
      if (e.key === "Escape") {
        if (openPopover) setOpenPopover(null);
        else if (editingField) setEditingField(null);
        else onClose();
        return;
      }
      if (e.metaKey || e.ctrlKey) return;
      if (openPopover) {
        if (openPopover === "labels" || openPopover === "cycle") return;
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
        showToast("Copied issue ID");
      }
      const num = parseInt(e.key);
      if (num >= 1 && num <= 5) {
        const status = STATUS_OPTIONS[num - 1];
        if (status && status.value !== issue.status)
          saveUpdate({ status: status.value });
      }
    }, [
    onClose,
    onNavigate,
    openPopover,
    editingField,
    issue.status,
    issue.estimate,
    issue.priority,
    issue.id,
    saveUpdate,
    handleStatusChange,
    handleEstimateChange,
    handlePriorityChange,
    popoverIndex,
    showToast,
  ]);

  useKeyboardHandler({
    scope: "issue-detail",
    priority: openPopover ? "control" : "view",
    handler: handleKeyboard,
    shortcuts: [
      ["Escape", "Close issue"], ["ArrowDown", "Next option"], ["ArrowUp", "Previous option"], ["Enter", "Confirm option"],
      ["ArrowLeft", "Previous issue"], ["k", "Previous issue"], ["ArrowRight", "Next issue"], ["j", "Next issue"],
      ["s", "Set status"], ["l", "Set labels"], ["e", "Set estimate"], ["p", "Set priority"], ["a", "Set assignee"],
      ["m", "Add comment"], [".", "Copy issue ID"], ["1", "Move to Backlog"], ["2", "Move to Planned"],
      ["3", "Move to In Progress"], ["4", "Move to Blocked"], ["5", "Move to Done"],
    ].map(([key, label]) => ({
      id: `issue-detail.${key}`, key, label,
      group: "Issue Detail",
      showInHelp: key !== "Escape" && !["ArrowDown", "ArrowUp", "Enter"].includes(key),
      preventDefault: false,
    })),
  });

  const startEditing = (field: string) => {
    setEditingField(field);
    if (field === "title") setEditTitle(issue.title);
    if (field === "description") setEditDescription(issue.description || "");
  };

  const handleSaveTitle = () => {
    if (editTitle.trim() && editTitle !== issue.title) {
      saveUpdate({ title: editTitle.trim() });
    } else {
      setEditingField(null);
    }
  };

  const handleSaveDescription = () => {
    if (editDescription !== (issue.description || "")) {
      saveUpdate({ description: editDescription });
    } else {
      setEditingField(null);
    }
  };

  const handleAddComment = () => {
    if (newComment.trim()) {
      saveComment({
        id: `c-${Date.now().toString(36)}`,
        text: newComment.trim(),
      });
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
        onMerged={() => {
          setMergeViewOpen(false);
          showToast("Branch merged");
        }}
      />
    );
  }

  return (
    <ViewContainer
      header={banner}
      contentRef={scrollRef}
      maxWidth="detail"
      innerClassName="flex min-h-full"
      topBar={
        <TopBar
          left={
            <div className="flex items-center gap-2 text-sm min-w-0">
              <button
                onClick={onClose}
                className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors shrink-0"
              >
                Issues
              </button>
              <ChevronRight className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
              <CopyableId id={issue.id} className="text-xs shrink-0" />
              <span className="text-[var(--color-text-primary)] truncate">
                {issue.title}
              </span>
            </div>
          }
          right={
            <div className="flex items-center gap-1 shrink-0">
              {worktree && (
                <Tooltip content="Open worktree in VSCode">
                  <a
                    href={vscodeUrl(worktree.path)}
                    className={iconButtonClass(false, "w-auto px-2 gap-1.5 mr-2 text-xs")}
                  >
                    <VSCodeIcon size={13} />
                    Open
                  </a>
                </Tooltip>
              )}
              <span className="text-xs text-[var(--color-text-muted)] tabular-nums mr-1">
                {currentIndex + 1} / {totalCount}
              </span>
              <button
                onClick={() => onNavigate("prev")}
                disabled={!hasPrev}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors disabled:opacity-20 disabled:pointer-events-none"
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
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors disabled:opacity-20 disabled:pointer-events-none"
              >
                <ChevronRight size={16} />
              </button>
            </div>
          }
        />
      }
    >
      {/* Main content */}
      <div className="flex-1 min-w-0">
        <div className="px-8 py-12">
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
              style={{
                caretColor: "var(--color-accent-primary)",
                height: "auto",
              }}
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
          {issue.parent_id &&
            (() => {
              const parent = issues.find((i) => i.id === issue.parent_id);
              const siblings = parent
                ? issues.filter((i) => i.parent_id === parent.id)
                : [];
              const siblingsDone = siblings.filter(
                (i) => i.status === "DONE",
              ).length;
              return (
                <div className="flex items-center gap-2 text-sm text-[var(--color-text-muted)] mt-2 flex-wrap">
                  <span>Sub-issue of</span>
                  {parent && (
                    <StatusIcon
                      status={parent.status}
                      size={14}
                      isInferred={parent.is_inferred}
                    />
                  )}
                  <a
                    href={`#/issues/${issue.parent_id}`}
                    className="font-mono text-[var(--color-accent-primary)] hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {issue.parent_id}
                  </a>
                  {parent && (
                    <Text color="secondary">
                      {parent.title}
                    </Text>
                  )}
                  {siblings.length > 0 && (
                    <Text>
                      ({siblingsDone}/{siblings.length})
                    </Text>
                  )}
                </div>
              );
            })()}

          {/* Tabs */}
          {(() => {
            const hasSpec = issue.artifacts?.some(
              (a) => a.artifact_type === "spec",
            );
            const hasWalkthrough = issue.artifacts?.some(
              (a) => a.artifact_type === "walkthrough",
            );
            const hasTabs = hasSpec || hasWalkthrough;

            const tabs: { key: DetailTab; label: string }[] = [
              { key: "details", label: "Details" },
              ...(hasSpec
                ? [{ key: "spec" as DetailTab, label: "Spec" }]
                : []),
              ...(hasWalkthrough
                ? [
                    {
                      key: "walkthrough" as DetailTab,
                      label: "Walkthrough",
                    },
                  ]
                : []),
            ];

            return (
              <>
                {hasTabs && (
                  <div className="flex items-center gap-1 mt-4 border-b border-[var(--color-border-subtle)]">
                    {tabs.map((tab) => (
                      <button
                        key={tab.key}
                        onClick={() => setActiveTab(tab.key)}
                        className={`px-3 py-2 text-sm font-medium transition-colors relative ${
                          activeTab === tab.key
                            ? "text-[var(--color-text-primary)]"
                            : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                        }`}
                      >
                        {tab.label}
                        {activeTab === tab.key && (
                          <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--color-accent-primary)]" />
                        )}
                      </button>
                    ))}
                  </div>
                )}

                {activeTab === "details" && (
                  <>
                    {/* Description */}
                    <div className={hasTabs ? "mt-6" : "mt-4"}>
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
                            if ((e.target as HTMLElement).closest("a"))
                              return;
                            setDescClickEvent({
                              clientX: e.clientX,
                              clientY: e.clientY,
                            });
                            startEditing("description");
                          }}
                          className="cursor-text min-h-10 prose-exponential"
                        >
                          {issue.description ? (
                            <Markdown
                              remarkPlugins={[remarkGfm, remarkBreaks]}
                            >
                              {linkifyIssueIds(
                                issue.description,
                                prefix,
                              )}
                            </Markdown>
                          ) : (
                            <Text as="p" size="base">
                              Add a description...
                            </Text>
                          )}
                        </div>
                      )}
                    </div>

                    <SubIssuesTable
                      issue={issue}
                      issues={issues}
                    />

                    <ArtifactList
                      artifacts={issue.artifacts ?? []}
                      issue={issue}
                    />

                    <ActivityTimeline
                      issue={issue}
                      newComment={newComment}
                      onNewCommentChange={setNewComment}
                      onAddComment={handleAddComment}
                      saving={saving}
                      commentRef={commentRef}
                      prefix={prefix}
                    />
                  </>
                )}

                {(activeTab === "spec" || activeTab === "walkthrough") &&
                  (() => {
                    const content = artifact.isPending ? undefined : (artifact.data ?? "");
                    return (
                      <div className="mt-6 prose-exponential">
                        {content === undefined ? (
                          <Text as="p">
                            Loading...
                          </Text>
                        ) : content ? (
                          <Markdown
                            remarkPlugins={[remarkGfm, remarkBreaks]}
                          >
                            {linkifyIssueIds(content, prefix)}
                          </Markdown>
                        ) : (
                          <Text as="p">
                            No content.
                          </Text>
                        )}
                      </div>
                    );
                  })()}
              </>
            );
          })()}
        </div>
      </div>

      <PropertySidebar
        issue={issue}
        issues={issues}
        openPopover={openPopover}
        setOpenPopover={setOpenPopover}
        popoverIndex={popoverIndex}
        setPopoverIndex={setPopoverIndex}
        saveUpdate={saveUpdate}
        onClose={onClose}
        contributors={contributors}
        onOpenMerge={() => setMergeViewOpen(true)}
      />
    </ViewContainer>
  );
}
