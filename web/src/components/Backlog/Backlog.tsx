import {
  useState,
  useRef,
  useEffect,
  useCallback,
  useMemo,
  useContext,
} from "react";
import { createPortal } from "react-dom";
import { generateKeyBetween } from "fractional-indexing";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragOverEvent,
  type DragEndEvent,
  type DragMoveEvent,
} from "@dnd-kit/core";
import { createIssue, addDraft, fetchCycles } from "../../api/client";
import type { Issue } from "../../api/client";
import {
  EmptyState,
  EstimateBadge,
  LabelBadge,
  DefaultLabelsContext,
  StatusIcon,
  PriorityIcon,
  ContextMenu,
  useToast,
  Modal,
  TopBar,
} from "../ui";
import {
  ChevronsDownUp,
  ChevronsUpDown,
  ListTree,
  Settings2,
} from "lucide-react";
import { useAllLabels } from "../../hooks/useLabels";
import { toggleLabel, splitLabels } from "../../utils/labels";
import { computeAppendKey, SORT_OPTIONS } from "../../utils/sort";
import type { SortKey } from "../../utils/sort";
import { isTerminal } from "../../constants";
import { buildChildrenByParent } from "../../utils/issues";
import { isEditableTarget } from "../../utils/keyboard";
import Tooltip from "../ui/Tooltip";
import FilterMenu from "./FilterMenu";
import { type BacklogFilters, hasActiveFilters, matchesFilters } from "./filters";
import { DragOverlayCard } from "./DndComponents";
import { backlogCollision } from "./backlogCollision";
import {
  useBacklogRows,
  type RowItem,
  type HierarchyMode,
} from "./useBacklogRows";
import { BacklogGroupHeader } from "./BacklogGroupHeader";
import { BacklogIssueRow } from "./BacklogIssueRow";
import "./backlog-dnd.css";

export type Tab = "all" | "active" | "backlog" | "done";

const TAB_CONFIGS: Record<Tab, { label: string; statuses: string[] }> = {
  all: {
    label: "All Issues",
    statuses: ["BACKLOG", "PLANNED", "DOING", "BLOCKED", "DONE", "CANCELED", "DUPLICATE"],
  },
  active: { label: "Active", statuses: ["PLANNED", "DOING", "BLOCKED"] },
  backlog: { label: "Backlog", statuses: ["BACKLOG"] },
  done: { label: "Completed", statuses: ["DONE", "CANCELED", "DUPLICATE"] },
};

const GROUP_VISIBLE_COUNT = 100;

type DropIndicatorValue = { rowIndex: number; position: "above" | "below" } | null;

function applyIndicatorDOM(
  listEl: HTMLElement | null,
  prev: DropIndicatorValue,
  next: DropIndicatorValue,
) {
  if (!listEl) return;
  if (prev) {
    const el = listEl.querySelector<HTMLElement>(`[data-row="${prev.rowIndex}"]`);
    const w = el?.closest<HTMLElement>("[data-context-issue]");
    if (w) {
      delete w.dataset.drop;
      w.style.removeProperty("--indicator-left");
    }
  }
  if (next) {
    const el = listEl.querySelector<HTMLElement>(`[data-row="${next.rowIndex}"]`);
    const w = el?.closest<HTMLElement>("[data-context-issue]");
    if (w) {
      const depth = parseInt(w.dataset.depth || "0", 10);
      w.style.setProperty("--indicator-left", `${20 + depth * 24}px`);
      w.dataset.drop = next.position;
    }
  }
}

function applyGroupDOM(
  listEl: HTMLElement | null,
  prev: string | null,
  next: string | null,
) {
  if (!listEl || prev === next) return;
  if (prev) {
    const w = listEl.querySelector<HTMLElement>(`[data-group-status="${prev}"]`);
    if (w) {
      w.style.boxShadow = "";
      w.style.background = "";
      const h = w.querySelector<HTMLElement>("[data-group-header]");
      if (h) h.style.backgroundColor = "";
    }
  }
  if (next) {
    const w = listEl.querySelector<HTMLElement>(`[data-group-status="${next}"]`);
    if (w) {
      w.style.boxShadow = "inset 0 0 0 2px var(--color-accent-primary)";
      w.style.background =
        "color-mix(in srgb, var(--color-accent-primary) 5%, transparent)";
      const h = w.querySelector<HTMLElement>("[data-group-header]");
      if (h) h.style.backgroundColor = "transparent";
    }
  }
}

function applyNestDOM(
  listEl: HTMLElement | null,
  prev: string | null,
  next: string | null,
) {
  if (!listEl || prev === next) return;
  if (prev) {
    const el = listEl.querySelector<HTMLElement>(
      `[data-context-issue="${prev}"] [data-backlog-row]`,
    );
    if (el) {
      el.style.boxShadow = "";
      el.style.background = "";
    }
  }
  if (next) {
    const el = listEl.querySelector<HTMLElement>(
      `[data-context-issue="${next}"] [data-backlog-row]`,
    );
    if (el) {
      el.style.boxShadow = "inset 0 0 0 2px var(--color-accent-primary)";
      el.style.background =
        "color-mix(in srgb, var(--color-accent-primary) 10%, transparent)";
    }
  }
}

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
  sortKey: SortKey;
  onSortChange: (key: SortKey) => void;
  onNavigationOrderChange?: (ids: string[]) => void;
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
  contributors: string[];
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  onNewIssue?: () => void;
  filters: BacklogFilters;
  onFiltersChange: (filters: BacklogFilters) => void;
  patchIssue?: (issueId: string, patch: Partial<Issue>) => void;
}

export default function Backlog({
  issues,
  onRefresh,
  onIssueClick,
  sortKey,
  onSortChange,
  onNavigationOrderChange,
  activeTab,
  onTabChange,
  contributors,
  onConfigLabelsChange,
  onNewIssue,
  filters,
  onFiltersChange,
  patchIssue,
}: BacklogProps) {
  const [search, setSearch] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(
    () => new Set(),
  );
  const [nodeToggleCount, setNodeToggleCount] = useState(0);
  const [showAllGroups, setShowAllGroups] = useState<Set<string>>(
    () => new Set(),
  );
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [keyboardNav, setKeyboardNav] = useState(false);
  const [inlineCreateStatus, setInlineCreateStatus] = useState<string | null>(
    null,
  );
  const [inlineTitle, setInlineTitle] = useState("");
  const showToast = useToast();
  const [openPopover, setOpenPopover] = useState<{
    rowIndex: number;
    type: "status" | "estimate" | "labels" | "priority";
  } | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    issueId: string;
    x: number;
    y: number;
  } | null>(null);
  const [cycleMap, setCycleMap] = useState<Map<string, number>>(new Map());
  const [showStoryPoints, setShowStoryPoints] = useState(
    () => localStorage.getItem("exponential-backlog-show-points") === "true",
  );
  const [showGhosts, setShowGhosts] = useState<boolean>(() => {
    try {
      const stored = localStorage.getItem("exponential-backlog-show-done-ghosts");
      if (stored !== null) return stored === "true";
      return false;
    } catch {
      return false;
    }
  });
  const toggleGhosts = useCallback(() => {
    setShowGhosts((prev) => {
      const next = !prev;
      try {
        localStorage.setItem("exponential-backlog-show-done-ghosts", String(next));
      } catch { /* localStorage unavailable */ }
      return next;
    });
  }, []);
  const [showFilterMenu, setShowFilterMenu] = useState(false);
  const [hierarchyMode, setHierarchyMode] = useState<HierarchyMode>(() => {
    try {
      return (
        (localStorage.getItem(
          `exponential-backlog-hierarchy-${activeTab}`,
        ) as HierarchyMode) || "nested"
      );
    } catch {
      return "nested";
    }
  });
  const toggleHierarchy = useCallback(() => {
    setHierarchyMode((prev) => {
      const next = prev === "nested" ? "flat" : "nested";
      try {
        localStorage.setItem(
          `exponential-backlog-hierarchy-${activeTab}`,
          next,
        );
      } catch { /* localStorage unavailable */ }
      return next;
    });
  }, [activeTab]);
  const [showEmptyGroups, setShowEmptyGroups] = useState<boolean>(() => {
    try {
      const stored = localStorage.getItem(
        `exponential-backlog-empty-groups-${activeTab}`,
      );
      if (stored !== null) return stored === "true";
      return activeTab === "all" || activeTab === "done";
    } catch {
      return activeTab === "all" || activeTab === "done";
    }
  });
  const toggleEmptyGroups = useCallback(() => {
    setShowEmptyGroups((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(
          `exponential-backlog-empty-groups-${activeTab}`,
          String(next),
        );
      } catch { /* localStorage unavailable */ }
      return next;
    });
  }, [activeTab]);
  const showFilterMenuRef = useRef(showFilterMenu);
  showFilterMenuRef.current = showFilterMenu;
  const filterBtnRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const inlineRef = useRef<HTMLInputElement>(null);
  const defaultLabels = useContext(DefaultLabelsContext);

  useEffect(() => {
    fetchCycles()
      .then((data) => {
        if (data.enabled && data.cycles) {
          setCycleMap(new Map(data.cycles.map((c) => [c.id, c.number])));
        }
      })
      .catch(() => {});
  }, []);

  const handleInlineCreate = useCallback(
    async (status: string, title: string) => {
      if (!title.trim()) return;
      const sortOrder = computeAppendKey(issues);
      const issueId = await createIssue({
        title: title.trim(),
        labels: ["feature"],
        sort_order: sortOrder,
      });
      if (status !== "BACKLOG") await addDraft(issueId, "UPDATE", { status });
      setInlineTitle("");
      onRefresh();
      showToast("Issue created");
    },
    [onRefresh, issues, showToast],
  );

  const STATUS_LABELS: Record<string, string> = {
    BACKLOG: "Backlog",
    PLANNED: "Planned",
    DOING: "In Progress",
    BLOCKED: "Blocked",
    DONE: "Done",
    CANCELED: "Canceled",
    DUPLICATE: "Duplicate",
  };

  const [moveChildrenPrompt, setMoveChildrenPrompt] = useState<{
    issueId: string;
    title: string;
    status: string;
    doneCount: number;
    children: { id: string; title: string; status: string }[];
    pendingUpdate?: Record<string, unknown>;
  } | null>(null);

  const applyStatusChange = useCallback(
    async (
      issueId: string,
      status: string,
      includeChildren: boolean,
      extraFields?: Record<string, unknown>,
    ) => {
      await addDraft(issueId, "UPDATE", { status, ...extraFields });
      if (includeChildren) {
        const children = issues.filter(
          (i) =>
            i.parent_id === issueId &&
            i.status !== status &&
            !isTerminal(i.status),
        );
        for (const child of children) {
          await addDraft(child.id, "UPDATE", { status });
        }
      }
      onRefresh();
    },
    [issues, onRefresh],
  );

  const handleQuickStatus = useCallback(
    async (issueId: string, status: string) => {
      const children = issues.filter(
        (i) =>
          i.parent_id === issueId && i.status !== status && !isTerminal(i.status),
      );
      setOpenPopover(null);
      if (children.length > 0) {
        const doneCount = issues.filter(
          (i) => i.parent_id === issueId && isTerminal(i.status),
        ).length;
        const issue = issues.find((i) => i.id === issueId);
        setMoveChildrenPrompt({
          issueId,
          title: issue?.title || issueId,
          status,
          doneCount,
          children: children.map((c) => ({
            id: c.id,
            title: c.title,
            status: c.status,
          })),
        });
        return;
      }
      await addDraft(issueId, "UPDATE", { status });
      onRefresh();
      showToast(`Status changed to ${status}`);
    },
    [issues, onRefresh, showToast],
  );
  const handleQuickEstimate = useCallback(
    async (issueId: string, estimate: number) => {
      await addDraft(issueId, "UPDATE", { estimate });
      setOpenPopover(null);
      onRefresh();
      showToast(`Estimate set to ${estimate || "none"}`);
    },
    [onRefresh, showToast],
  );
  const handleQuickPriority = useCallback(
    async (issueId: string, priority: number) => {
      await addDraft(issueId, "UPDATE", { priority });
      setOpenPopover(null);
      onRefresh();
    },
    [onRefresh],
  );
  const handleQuickLabelToggle = useCallback(
    async (issue: Issue, label: string) => {
      const current = issue.labels || [];
      const next = toggleLabel(current, label);
      const removed = next.length < current.length;
      await addDraft(issue.id, "UPDATE", { labels: next });
      onRefresh();
      showToast(
        removed ? `Removed label "${label}"` : `Added label "${label}"`,
      );
    },
    [onRefresh, showToast],
  );

  const allKnownLabels = useAllLabels(issues);

  const startInlineCreate = useCallback(
    (status: string) => {
      setInlineCreateStatus(status);
      setInlineTitle("");
      if (!expandedGroups.has(status))
        setExpandedGroups((prev) => new Set(prev).add(status));
      setTimeout(() => inlineRef.current?.focus(), 0);
    },
    [expandedGroups],
  );


  const visibleStatuses = TAB_CONFIGS[activeTab].statuses;
  const query = search.toLowerCase();
  const filteredIssues = issues.filter((i) => {
    if (!visibleStatuses.includes(i.status)) return false;
    if (
      query &&
      !(
        i.title.toLowerCase().includes(query) ||
        i.id.toLowerCase().includes(query) ||
        i.labels?.some((l) => l.toLowerCase().includes(query))
      )
    )
      return false;
    return matchesFilters(i, filters);
  });

  const childrenByParent = useMemo(() => buildChildrenByParent(issues), [issues]);

  const expandedNodes = useMemo(() => {
    void nodeToggleCount;
    const stored = localStorage.getItem(`exponential-backlog-nodes-collapsed`);
    const collapsed: Set<string> = stored
      ? new Set(JSON.parse(stored) as string[])
      : new Set();
    const expanded = new Set(Array.from(childrenByParent.keys()));
    for (const id of collapsed) expanded.delete(id);
    return expanded;
  }, [childrenByParent, nodeToggleCount]);

  useEffect(() => {
    const storageKey = `exponential-backlog-expanded-${activeTab}`;
    const stored = localStorage.getItem(storageKey);
    let next: Set<string>;
    if (stored) {
      next = new Set(JSON.parse(stored) as string[]);
    } else {
      next = new Set<string>();
      for (const status of visibleStatuses) {
        const count = issues.filter((i) => i.status === status).length;
        if (count > 0 && (!isTerminal(status) || activeTab === "done"))
          next.add(status);
      }
    }
    setExpandedGroups(next);
    try {
      setHierarchyMode(
        (localStorage.getItem(
          `exponential-backlog-hierarchy-${activeTab}`,
        ) as HierarchyMode) || "nested",
      );
    } catch { /* localStorage unavailable */ }
    try {
      const emptyStored = localStorage.getItem(
        `exponential-backlog-empty-groups-${activeTab}`,
      );
      setShowEmptyGroups(
        emptyStored !== null
          ? emptyStored === "true"
          : activeTab === "all" || activeTab === "done",
      );
    } catch { /* localStorage unavailable */ }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  const toggleGroup = useCallback(
    (status: string) => {
      setExpandedGroups((prev) => {
        const next = new Set(prev);
        if (next.has(status)) next.delete(status);
        else next.add(status);
        localStorage.setItem(
          `exponential-backlog-expanded-${activeTab}`,
          JSON.stringify(Array.from(next)),
        );
        return next;
      });
    },
    [activeTab],
  );

  const toggleNode = useCallback((id: string) => {
    const nodesKey = `exponential-backlog-nodes-collapsed`;
    const stored = localStorage.getItem(nodesKey);
    const collapsed: Set<string> = stored
      ? new Set(JSON.parse(stored))
      : new Set();
    if (collapsed.has(id)) collapsed.delete(id);
    else collapsed.add(id);
    localStorage.setItem(nodesKey, JSON.stringify(Array.from(collapsed)));
    setNodeToggleCount((c) => c + 1);
  }, []);

  const expandAllNodes = useCallback(() => {
    localStorage.setItem(`exponential-backlog-nodes-collapsed`, "[]");
    setNodeToggleCount((c) => c + 1);
  }, []);

  const collapseAllNodes = useCallback(() => {
    localStorage.setItem(
      `exponential-backlog-nodes-collapsed`,
      JSON.stringify(Array.from(childrenByParent.keys())),
    );
    setNodeToggleCount((c) => c + 1);
  }, [childrenByParent]);


  const rows = useBacklogRows(
    issues,
    filteredIssues,
    activeTab,
    expandedGroups,
    expandedNodes,
    sortKey,
    hierarchyMode,
    showEmptyGroups,
    showGhosts,
  );

  // Report navigation order to parent
  const prevNavOrder = useRef<string>("");
  useEffect(() => {
    const ids = rows
      .filter((r) => r.kind === "issue")
      .map((r) => (r as { kind: "issue"; issue: Issue }).issue.id);
    const key = ids.join(",");
    if (key !== prevNavOrder.current) {
      prevNavOrder.current = key;
      onNavigationOrderChange?.(ids);
    }
  }, [rows, onNavigationOrderChange]);

  // DnD state
  const isDndEnabled = sortKey === "manual";
  const [activeId, setActiveId] = useState<string | null>(null);
  const dropIndicatorRef = useRef<DropIndicatorValue>(null);
  const dropGroupStatusRef = useRef<string | null>(null);
  const dropNestTargetIdRef = useRef<string | null>(null);
  const dragBatchRef = useRef<string[]>([]);
  const modifiersRef = useRef({ alt: false, meta: false, ctrl: false });
  const pointerYRef = useRef<number>(0);

  const setIndicator = useCallback((value: DropIndicatorValue) => {
    const prev = dropIndicatorRef.current;
    dropIndicatorRef.current = value;
    applyIndicatorDOM(listRef.current, prev, value);
  }, []);

  const setGroup = useCallback((value: string | null) => {
    const prev = dropGroupStatusRef.current;
    dropGroupStatusRef.current = value;
    applyGroupDOM(listRef.current, prev, value);
  }, []);

  const setNest = useCallback((value: string | null) => {
    const prev = dropNestTargetIdRef.current;
    dropNestTargetIdRef.current = value;
    applyNestDOM(listRef.current, prev, value);
  }, []);

  const wasDraggingRef = useRef(false);

  useEffect(() => {
    if (!activeId) return;
    const syncKeys = (e: KeyboardEvent) => {
      modifiersRef.current = {
        alt: e.altKey,
        meta: e.metaKey,
        ctrl: e.ctrlKey,
      };
    };
    const syncPointer = (e: PointerEvent) => {
      pointerYRef.current = e.clientY;
    };
    window.addEventListener("keydown", syncKeys);
    window.addEventListener("keyup", syncKeys);
    window.addEventListener("pointermove", syncPointer);
    return () => {
      window.removeEventListener("keydown", syncKeys);
      window.removeEventListener("keyup", syncKeys);
      window.removeEventListener("pointermove", syncPointer);
    };
  }, [activeId]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 3 } }),
    useSensor(KeyboardSensor),
  );

  const getRowStatusGroup = useCallback(
    (rowIndex: number): string | null => {
      for (let j = rowIndex; j >= 0; j--) {
        const r = rows[j];
        if (r.kind === "group") return r.status;
      }
      return null;
    },
    [rows],
  );

  const getDragGroup = useCallback(
    (issueId: string): string[] => {
      const issue = issues.find((i) => i.id === issueId);
      if (!issue) return [issueId];
      const children = (childrenByParent.get(issueId) || []).filter(
        (c) => c.status === issue.status,
      );
      if (children.length === 0) return [issueId];
      return [issueId, ...children.map((c) => c.id)];
    },
    [issues, childrenByParent],
  );

  const performDrop = useCallback(
    async (
      droppedId: string,
      groupTarget: string | null,
      indicatorTarget: { rowIndex: number; position: "above" | "below" } | null,
      nestTarget: string | null,
      altHeld: boolean,
    ) => {
      if (nestTarget && nestTarget !== droppedId) {
        const draggedIssue = issues.find((i) => i.id === droppedId);
        const targetRowIdx = rows.findIndex(
          (r) => r.kind === "issue" && r.issue.id === nestTarget,
        );
        const targetStatus =
          targetRowIdx >= 0 ? getRowStatusGroup(targetRowIdx) : null;
        const newSiblings = issues
          .filter((i) => i.parent_id === nestTarget)
          .sort((a, b) =>
            (a.sort_order || "") < (b.sort_order || "")
              ? -1
              : (a.sort_order || "") > (b.sort_order || "")
                ? 1
                : 0,
          );
        const lastKey = newSiblings[newSiblings.length - 1]?.sort_order || null;
        const nestUpdate: Record<string, unknown> = {
          parent_id: nestTarget,
          sort_order: generateKeyBetween(lastKey, null),
        };
        if (targetStatus && draggedIssue?.status !== targetStatus) {
          nestUpdate.status = targetStatus;
        }
        await addDraft(droppedId, "UPDATE", nestUpdate);
        onRefresh();
        return;
      }
      if (groupTarget) {
        const draggedIssue = issues.find((i) => i.id === droppedId);
        let groupUpdate: Record<string, unknown>;
        if (draggedIssue?.parent_id) {
          groupUpdate = { status: groupTarget };
        } else {
          const groupIssues = rows
            .filter(
              (r): r is typeof r & { kind: "issue" } =>
                r.kind === "issue" && r.depth === 0,
            )
            .filter((r) => {
              const idx = rows.indexOf(r);
              return getRowStatusGroup(idx) === groupTarget;
            })
            .map((r) => r.issue);
          const last = groupIssues[groupIssues.length - 1];
          const lastKey = last?.sort_order || null;
          groupUpdate = {
            status: groupTarget,
            sort_order: generateKeyBetween(lastKey, null),
          };
        }
        const movableChildren = issues.filter(
          (i) =>
            i.parent_id === droppedId &&
            i.status !== groupTarget &&
            !isTerminal(i.status),
        );
        if (movableChildren.length > 0) {
          const doneCount = issues.filter(
            (i) => i.parent_id === droppedId && isTerminal(i.status),
          ).length;
          setMoveChildrenPrompt({
            issueId: droppedId,
            title: draggedIssue?.title || droppedId,
            status: groupTarget,
            doneCount,
            children: movableChildren.map((c) => ({
              id: c.id,
              title: c.title,
              status: c.status,
            })),
            pendingUpdate: groupUpdate,
          });
          return;
        }
        await addDraft(droppedId, "UPDATE", groupUpdate);
        onRefresh();
        return;
      }
      if (!indicatorTarget) return;
      const targetRow = rows[indicatorTarget.rowIndex];
      if (targetRow.kind !== "issue") return;
      const status = getRowStatusGroup(indicatorTarget.rowIndex);
      const draggedStatus = issues.find((i) => i.id === droppedId)?.status;
      const update: Record<string, unknown> = {};
      if (draggedStatus !== status) update.status = status;
      if (targetRow.depth > 0) {
        if (altHeld) {
          update.parent_id = targetRow.issue.parent_id!;
        }
        const targetParentId = targetRow.issue.parent_id!;
        const siblings = issues
          .filter((i) => i.parent_id === targetParentId)
          .sort((a, b) =>
            (a.sort_order || "") < (b.sort_order || "")
              ? -1
              : (a.sort_order || "") > (b.sort_order || "")
                ? 1
                : 0,
          );
        const withoutDragged = siblings.filter((i) => i.id !== droppedId);
        let insertIdx = withoutDragged.findIndex(
          (i) => i.id === targetRow.issue.id,
        );
        if (insertIdx === -1) insertIdx = withoutDragged.length;
        if (indicatorTarget.position === "below") insertIdx++;
        const prev = withoutDragged[insertIdx - 1];
        const next = withoutDragged[insertIdx];
        const prevKey = prev?.sort_order || null;
        const nextKey = next?.sort_order || null;
        update.sort_order = generateKeyBetween(
          prevKey,
          nextKey && nextKey !== prevKey ? nextKey : null,
        );
      } else {
        const dragged = issues.find((i) => i.id === droppedId);
        const sharedParent =
          dragged?.parent_id &&
          targetRow.issue.parent_id === dragged.parent_id
            ? dragged.parent_id
            : null;
        const adoptParent =
          !sharedParent && altHeld && targetRow.issue.parent_id
            ? targetRow.issue.parent_id
            : null;
        if (sharedParent || adoptParent) {
          const parentId = (sharedParent || adoptParent)!;
          if (adoptParent) update.parent_id = adoptParent;
          const siblings = issues
            .filter((i) => i.parent_id === parentId)
            .sort((a, b) =>
              (a.sort_order || "") < (b.sort_order || "")
                ? -1
                : (a.sort_order || "") > (b.sort_order || "")
                  ? 1
                  : 0,
            );
          const withoutDragged = siblings.filter((i) => i.id !== droppedId);
          let insertIdx = withoutDragged.findIndex(
            (i) => i.id === targetRow.issue.id,
          );
          if (insertIdx === -1) insertIdx = withoutDragged.length;
          if (indicatorTarget.position === "below") insertIdx++;
          const prev = withoutDragged[insertIdx - 1];
          const next = withoutDragged[insertIdx];
          const prevKey = prev?.sort_order || null;
          const nextKey = next?.sort_order || null;
          update.sort_order = generateKeyBetween(
            prevKey,
            nextKey && nextKey !== prevKey ? nextKey : null,
          );
        } else {
          if (altHeld) {
            if (dragged?.parent_id) update.parent_id = "";
          }
          const groupTopLevel = rows
            .filter(
              (r): r is typeof r & { kind: "issue" } =>
                r.kind === "issue" && r.depth === 0,
            )
            .filter((r) => {
              const idx = rows.indexOf(r);
              return getRowStatusGroup(idx) === status;
            })
            .map((r) => r.issue);
          const withoutDragged = groupTopLevel.filter(
            (i) => i.id !== droppedId,
          );
          let insertIdx = withoutDragged.findIndex(
            (i) => i.id === targetRow.issue.id,
          );
          if (insertIdx === -1) insertIdx = withoutDragged.length;
          if (indicatorTarget.position === "below") insertIdx++;
          const prev = withoutDragged[insertIdx - 1];
          const next = withoutDragged[insertIdx];
          update.sort_order = generateKeyBetween(
            prev?.sort_order || null,
            next?.sort_order || null,
          );
        }
      }
      if (update.status) {
        const movableChildren = issues.filter(
          (i) =>
            i.parent_id === droppedId &&
            i.status !== update.status &&
            !isTerminal(i.status),
        );
        if (movableChildren.length > 0) {
          const issue = issues.find((i) => i.id === droppedId);
          const doneCount = issues.filter(
            (i) => i.parent_id === droppedId && isTerminal(i.status),
          ).length;
          setMoveChildrenPrompt({
            issueId: droppedId,
            title: issue?.title || droppedId,
            status: update.status as string,
            doneCount,
            children: movableChildren.map((c) => ({
              id: c.id,
              title: c.title,
              status: c.status,
            })),
            pendingUpdate: update,
          });
          return;
        }
      }
      await addDraft(droppedId, "UPDATE", update);
      onRefresh();
    },
    [rows, getRowStatusGroup, issues, onRefresh],
  );

  const resetDropState = useCallback(() => {
    setActiveId(null);
    setIndicator(null);
    setGroup(null);
    setNest(null);
    modifiersRef.current = { alt: false, meta: false, ctrl: false };
    setTimeout(() => {
      wasDraggingRef.current = false;
    }, 0);
  }, [setIndicator, setGroup, setNest]);

  const handleDndStart = useCallback(
    (event: DragStartEvent) => {
      const id = String(event.active.id);
      const orig = event.activatorEvent as MouseEvent | KeyboardEvent;
      modifiersRef.current = {
        alt: !!orig.altKey,
        meta: !!orig.metaKey,
        ctrl: !!orig.ctrlKey,
      };
      if (typeof (orig as PointerEvent).clientY === "number")
        pointerYRef.current = (orig as PointerEvent).clientY;
      dragBatchRef.current = getDragGroup(id);
      wasDraggingRef.current = true;
      setActiveId(id);
      setIndicator(null);
      setGroup(null);
      setNest(null);
    },
    [getDragGroup, setIndicator, setGroup, setNest],
  );

  const isDescendant = useCallback(
    (parentId: string, childId: string): boolean => {
      for (const i of issues) {
        if (i.parent_id === parentId) {
          if (i.id === childId) return true;
          if (isDescendant(i.id, childId)) return true;
        }
      }
      return false;
    },
    [issues],
  );

  const handleDndOver = useCallback(
    (event: DragOverEvent) => {
      const draggedId = String(event.active.id);
      const overRaw = event.over?.id ? String(event.over.id) : null;
      if (!overRaw) {
        setIndicator(null);
        setGroup(null);
        setNest(null);
        return;
      }
      if (overRaw.startsWith("group-")) {
        const status = overRaw.slice("group-".length);
        const draggedStatus = issues.find((i) => i.id === draggedId)?.status;
        const groupIsEmpty = !rows.some(
          (r, j) => r.kind === "issue" && getRowStatusGroup(j) === status,
        );
        if (groupIsEmpty || draggedStatus !== status) {
          setIndicator(null);
          setGroup(status);
          setNest(null);
        } else {
          const firstIssueIdx = rows.findIndex(
            (r, j) => r.kind === "issue" && getRowStatusGroup(j) === status,
          );
          setIndicator({ rowIndex: firstIssueIdx, position: "above" });
          setGroup(null);
          setNest(null);
        }
        return;
      }
      if (overRaw.startsWith("ghost:")) {
        const ghostIssueId = overRaw.slice("ghost:".length);
        const ghostRowIndex = rows.findIndex(
          (r) =>
            r.kind === "issue" &&
            r.issue.id === ghostIssueId &&
            (!!r.isGhostParent || !!r.isGhostChild),
        );
        if (ghostRowIndex >= 0) {
          const status = getRowStatusGroup(ghostRowIndex);
          if (status) {
            setIndicator(null);
            setGroup(status);
            setNest(null);
          }
        }
        return;
      }
      const overRowIndex = rows.findIndex(
        (r) =>
          r.kind === "issue" &&
          r.issue.id === overRaw &&
          !r.isGhostParent &&
          !r.isGhostChild,
      );
      if (overRowIndex < 0) return;
      const overRow = rows[overRowIndex];
      if (overRow.kind !== "issue") return;
      if (overRow.issue.id === draggedId) {
        setIndicator(null);
        setGroup(null);
        setNest(null);
        return;
      }
      const draggedStatus = issues.find((i) => i.id === draggedId)?.status;
      const targetStatus = getRowStatusGroup(overRowIndex);
      const isCrossGroup = draggedStatus !== targetStatus;
      if (
        isCrossGroup &&
        !(
          modifiersRef.current.meta ||
          modifiersRef.current.ctrl ||
          modifiersRef.current.alt
        )
      ) {
        setIndicator(null);
        setGroup(targetStatus);
        return;
      }
      setGroup(null);
    },
    [rows, issues, getRowStatusGroup, setIndicator, setGroup, setNest],
  );

  const findAfterTree = useCallback(
    (
      parentIdx: number,
    ): { rowIndex: number; position: "above" | "below" } | null => {
      const nextTop = rows.findIndex(
        (r, j) => j > parentIdx && r.kind === "issue" && r.depth === 0,
      );
      if (nextTop !== -1) return { rowIndex: nextTop, position: "above" };
      for (let j = rows.length - 1; j > parentIdx; j--) {
        const r = rows[j];
        if (r.kind === "issue" && r.depth > 0)
          return { rowIndex: j, position: "below" };
      }
      return null;
    },
    [rows],
  );

  const isSiblingPosition = useCallback(
    (
      draggedParentId: string | undefined,
      targetRow: RowItem & { kind: "issue" },
      position: "above" | "below",
    ): boolean => {
      if (draggedParentId) {
        if (targetRow.depth === 0) {
          if (targetRow.issue.id === draggedParentId)
            return position === "below";
          if (targetRow.issue.parent_id === draggedParentId) return true;
          return false;
        }
        return targetRow.issue.parent_id === draggedParentId;
      }
      return targetRow.depth === 0 && !targetRow.issue.parent_id;
    },
    [],
  );

  const handleDndMove = useCallback(
    (event: DragMoveEvent) => {
      const draggedId = String(event.active.id);
      const overRaw = event.over?.id ? String(event.over.id) : null;
      if (!overRaw) return;

      if (overRaw.startsWith("ghost:")) return;

      if (overRaw.startsWith("group-")) {
        const status = overRaw.slice("group-".length);
        const draggedStatus = issues.find((i) => i.id === draggedId)?.status;
        const firstIssueIdx = rows.findIndex(
          (r, j) => r.kind === "issue" && getRowStatusGroup(j) === status,
        );
        if (draggedStatus === status && firstIssueIdx !== -1) {
          setIndicator({ rowIndex: firstIssueIdx, position: "above" });
        }
        return;
      }

      const overRowIndex = rows.findIndex(
        (r) =>
          r.kind === "issue" &&
          r.issue.id === overRaw &&
          !r.isGhostParent &&
          !r.isGhostChild,
      );
      if (overRowIndex < 0) return;
      const overRow = rows[overRowIndex];
      if (overRow.kind !== "issue") return;
      if (overRow.issue.id === draggedId) return;

      const dragged = issues.find((i) => i.id === draggedId);
      const draggedParentId = dragged?.parent_id;
      const draggedStatus = dragged?.status;
      const targetStatus = getRowStatusGroup(overRowIndex);

      if (
        modifiersRef.current.alt &&
        overRow.depth === 0 &&
        !overRow.issue.parent_id
      ) {
        const alreadyChild = draggedParentId === overRaw;
        if (!alreadyChild && !isDescendant(draggedId, overRaw)) {
          setNest(overRaw);
          setIndicator(null);
          setGroup(null);
          return;
        }
      }
      setNest(null);

      if (
        draggedStatus !== targetStatus &&
        !(
          modifiersRef.current.meta ||
          modifiersRef.current.ctrl ||
          modifiersRef.current.alt
        )
      ) {
        setIndicator(null);
        return;
      }

      const overEl = document.querySelector(`[data-row="${overRowIndex}"]`);
      const overRect = overEl?.getBoundingClientRect();
      if (!overRect) return;
      const position: "above" | "below" =
        pointerYRef.current < overRect.top + overRect.height / 2
          ? "above"
          : "below";

      const draggedRowIndex = rows.findIndex(
        (r) =>
          r.kind === "issue" &&
          r.issue.id === draggedId &&
          !r.isGhostParent &&
          !r.isGhostChild,
      );
      if (draggedRowIndex >= 0) {
        if (position === "below" && overRowIndex === draggedRowIndex - 1)
          return;
        if (position === "above" && overRowIndex === draggedRowIndex + 1)
          return;
      }

      if (
        position === "below" &&
        overRow.hasVisibleChildren &&
        (expandedNodes.has(overRow.issue.id) || overRow.isGhostParent)
      ) {
        const firstChildIdx = rows.findIndex(
          (r, j) =>
            j > overRowIndex && r.kind === "issue" && r.depth > overRow.depth,
        );
        if (firstChildIdx !== -1) {
          if (
            modifiersRef.current.alt ||
            draggedParentId === overRow.issue.id
          ) {
            setIndicator({ rowIndex: firstChildIdx, position: "above" });
          } else {
            setIndicator(findAfterTree(overRowIndex));
          }
          return;
        }
      }

      if (modifiersRef.current.alt) {
        setGroup(null);
        setIndicator({ rowIndex: overRowIndex, position });
        return;
      }

      if (isSiblingPosition(draggedParentId, overRow, position)) {
        setIndicator({ rowIndex: overRowIndex, position });
      } else if (overRow.depth > 0 && overRow.issue.parent_id) {
        const parentIdx = rows.findIndex(
          (r) => r.kind === "issue" && r.issue.id === overRow.issue.parent_id,
        );
        setIndicator(parentIdx !== -1 ? findAfterTree(parentIdx) : null);
      } else {
        setIndicator(null);
      }
    },
    [
      rows,
      issues,
      expandedNodes,
      getRowStatusGroup,
      findAfterTree,
      isSiblingPosition,
      isDescendant,
      setIndicator,
      setGroup,
      setNest,
    ],
  );

  const handleDndEnd = useCallback(
    async (event: DragEndEvent) => {
      const droppedId = String(event.active.id);
      const groupTarget = dropGroupStatusRef.current;
      const indicatorTarget = dropIndicatorRef.current;
      const nestTarget = dropNestTargetIdRef.current;
      const altHeld = modifiersRef.current.alt;
      resetDropState();
      if (!groupTarget && !indicatorTarget && !nestTarget) return;
      await performDrop(
        droppedId,
        groupTarget,
        indicatorTarget,
        nestTarget,
        altHeld,
      );
    },
    [performDrop, resetDropState],
  );

  const handleRowMouseEnter = useCallback((index: number) => {
    setKeyboardNav(false);
    setFocusedIndex(index);
  }, []);

  const handleOpenPopover = useCallback(
    (rowIndex: number, type: "status" | "labels" | "estimate" | "priority") => {
      setOpenPopover({ rowIndex, type });
    },
    [],
  );

  const handleClosePopover = useCallback(() => {
    setOpenPopover(null);
  }, []);

  const handleToggleStoryPoints = useCallback(() => {
    setShowStoryPoints((v) => {
      const next = !v;
      localStorage.setItem("exponential-backlog-show-points", String(next));
      return next;
    });
  }, []);

  const draggedIssue = useMemo(
    () => (activeId ? issues.find((i) => i.id === activeId) : undefined),
    [activeId, issues],
  );

  const [showViewMenu, setShowViewMenu] = useState(false);
  const viewBtnRef = useRef<HTMLButtonElement>(null);
  const viewMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!showViewMenu) return;
    const handler = (e: MouseEvent) => {
      if (
        viewMenuRef.current &&
        !viewMenuRef.current.contains(e.target as Node) &&
        viewBtnRef.current &&
        !viewBtnRef.current.contains(e.target as Node)
      )
        setShowViewMenu(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showViewMenu]);

  useEffect(() => {
    if (!showViewMenu) return;
    const btn = viewBtnRef.current;
    const menu = viewMenuRef.current;
    if (!btn || !menu) return;
    const aRect = btn.getBoundingClientRect();
    const mRect = menu.getBoundingClientRect();
    let top = aRect.bottom + 4;
    let left = aRect.left + aRect.width / 2 - mRect.width / 2;
    if (top + mRect.height > window.innerHeight - 8) top = aRect.top - mRect.height - 4;
    if (left < 8) left = 8;
    if (left + mRect.width > window.innerWidth - 8) left = window.innerWidth - mRect.width - 8;
    menu.style.top = `${top}px`;
    menu.style.left = `${left}px`;
    menu.style.visibility = "visible";
  }, [showViewMenu]);

  // Stable refs for keyboard handler
  const rowsRef = useRef(rows);
  rowsRef.current = rows;
  const focusedIndexRef = useRef(focusedIndex);
  focusedIndexRef.current = focusedIndex;
  const expandedGroupsRef = useRef(expandedGroups);
  expandedGroupsRef.current = expandedGroups;
  const expandedNodesRef = useRef(expandedNodes);
  expandedNodesRef.current = expandedNodes;
  const openPopoverRef = useRef(openPopover);
  openPopoverRef.current = openPopover;
  const onIssueClickRef = useRef(onIssueClick);
  onIssueClickRef.current = onIssueClick;
  const activeTabRef = useRef(activeTab);
  activeTabRef.current = activeTab;
  const onTabChangeRef = useRef(onTabChange);
  onTabChangeRef.current = onTabChange;
  const toggleGroupRef = useRef(toggleGroup);
  toggleGroupRef.current = toggleGroup;
  const toggleNodeRef = useRef(toggleNode);
  toggleNodeRef.current = toggleNode;
  const expandAllNodesRef = useRef(expandAllNodes);
  expandAllNodesRef.current = expandAllNodes;
  const collapseAllNodesRef = useRef(collapseAllNodes);
  collapseAllNodesRef.current = collapseAllNodes;

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (openPopoverRef.current) return;
      if (isEditableTarget(e)) return;
      if (e.metaKey || e.ctrlKey) return;
      if (e.key === "}") {
        e.preventDefault();
        expandAllNodesRef.current();
        return;
      }
      if (e.key === "{") {
        e.preventDefault();
        collapseAllNodesRef.current();
        return;
      }
      if (e.key === "[" || e.key === "]") {
        e.preventDefault();
        const tabs = Object.keys(TAB_CONFIGS) as Tab[];
        const idx = tabs.indexOf(activeTabRef.current);
        const next = e.key === "]"
          ? (idx + 1) % tabs.length
          : (idx - 1 + tabs.length) % tabs.length;
        onTabChangeRef.current(tabs[next]);
        return;
      }
      if (e.key === "f") {
        e.preventDefault();
        setShowFilterMenu((v) => !v);
        return;
      }
      if (e.key === "/") {
        e.preventDefault();
        searchRef.current?.focus();
        return;
      }
      if (showFilterMenuRef.current) {
        if (e.key === "Escape") {
          e.preventDefault();
          setShowFilterMenu(false);
        }
        return;
      }
      if (e.key === "ArrowDown" || e.key === "j") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex((i) => Math.min(i + 1, rowsRef.current.length - 1));
        return;
      }
      if (e.key === "ArrowUp" || e.key === "k") {
        e.preventDefault();
        setKeyboardNav(true);
        setFocusedIndex((i) => Math.max(i - 1, 0));
        return;
      }
      const row = rowsRef.current[focusedIndexRef.current];
      if (!row) return;
      if (e.key === "Enter") {
        e.preventDefault();
        if (row.kind === "group" && !row.isEmpty)
          toggleGroupRef.current(row.status);
        else if (row.kind === "issue") onIssueClickRef.current?.(row.issue);
        return;
      }
      if (e.key === "ArrowRight") {
        e.preventDefault();
        if (
          row.kind === "group" &&
          !row.isEmpty &&
          !expandedGroupsRef.current.has(row.status)
        )
          toggleGroupRef.current(row.status);
        else if (
          row.kind === "issue" &&
          row.hasVisibleChildren &&
          !expandedNodesRef.current.has(row.issue.id)
        )
          toggleNodeRef.current(row.issue.id);
        return;
      }
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        if (row.kind === "group" && expandedGroupsRef.current.has(row.status))
          toggleGroupRef.current(row.status);
        else if (
          row.kind === "issue" &&
          row.hasVisibleChildren &&
          expandedNodesRef.current.has(row.issue.id)
        )
          toggleNodeRef.current(row.issue.id);
        return;
      }
      if (e.key === "." && row.kind === "issue") {
        navigator.clipboard.writeText(row.issue.id);
        showToast("Copied issue ID");
        return;
      }
      if (row.kind === "issue") {
        const ri = focusedIndexRef.current;
        if (e.key === "s") {
          e.preventDefault();
          setOpenPopover({ rowIndex: ri, type: "status" });
          return;
        }
        if (e.key === "l") {
          e.preventDefault();
          setOpenPopover({ rowIndex: ri, type: "labels" });
          return;
        }
        if (e.key === "e") {
          e.preventDefault();
          setOpenPopover({ rowIndex: ri, type: "estimate" });
          return;
        }
        if (e.key === "p") {
          e.preventDefault();
          setOpenPopover({ rowIndex: ri, type: "priority" });
          return;
        }
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [showToast]);

  const issuesRef = useRef(issues);
  issuesRef.current = issues;
  useEffect(() => {
    const container = listRef.current;
    if (!container) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const row = target.closest<HTMLElement>("[data-context-issue]");
      if (!row) return;
      const issueId = row.getAttribute("data-context-issue");
      if (!issueId) return;
      e.preventDefault();
      setOpenPopover(null);
      setContextMenu({ issueId, x: e.clientX, y: e.clientY });
    };
    container.addEventListener("contextmenu", handler);
    return () => container.removeEventListener("contextmenu", handler);
  }, []);

  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-row="${focusedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  if (issues.length === 0) {
    return (
      <EmptyState
        title="No issues yet"
        description="Your backlog is empty. Create an issue to start tracking work for your project."
        icon={
          <svg width="160" height="120" viewBox="0 0 160 120" fill="none">
            <rect
              x="30"
              y="20"
              width="100"
              height="14"
              rx="4"
              stroke="var(--color-text-muted)"
              strokeWidth="1.5"
              strokeDasharray="4 3"
            />
            <rect
              x="30"
              y="42"
              width="100"
              height="14"
              rx="4"
              stroke="var(--color-text-muted)"
              strokeWidth="1.5"
              strokeDasharray="4 3"
            />
            <rect
              x="30"
              y="64"
              width="100"
              height="14"
              rx="4"
              stroke="var(--color-text-muted)"
              strokeWidth="1.5"
              strokeDasharray="4 3"
            />
            <circle cx="80" cy="100" r="2" fill="var(--color-text-muted)" />
          </svg>
        }
        actionLabel="Create an issue"
        onAction={onNewIssue}
      />
    );
  }

  return (
    <div className="h-full flex flex-col relative">
      {/* Tab bar */}
      <TopBar
        left={
          <div className="flex items-center gap-4 h-full">
            {(Object.entries(TAB_CONFIGS) as [Tab, { label: string }][]).map(
              ([id, config]) => (
                <button
                  key={id}
                  onClick={() => onTabChange(id)}
                  className={`text-sm font-medium h-full border-b-2 -mb-px transition-colors duration-[var(--duration-fast)] ${activeTab === id ? "text-[var(--color-text-primary)] border-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"}`}
                >
                  {config.label}
                </button>
              ),
            )}
          </div>
        }
        center={
          <div className="relative w-full max-w-md">
            <input
              ref={searchRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  setSearch("");
                  searchRef.current?.blur();
                }
              }}
              placeholder="Filter issues..."
              className="text-sm h-8 pl-8 pr-10 w-full rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-0)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]"
            />
            <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
            </svg>
            <kbd className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-[var(--color-text-muted)] border border-[var(--color-border-subtle)] rounded px-1 py-0.5 leading-none pointer-events-none">
              {search ? 'Esc' : '/'}
            </kbd>
          </div>
        }
        right={
          <div className="flex items-center gap-2">
            <div className="relative">
              <Tooltip content="Filter">
                <button
                  ref={filterBtnRef}
                  onClick={() => setShowFilterMenu((v) => !v)}
                  className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors relative"
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
                      d="M12 3c2.755 0 5.455.232 8.083.678.533.09.917.556.917 1.096v1.044a2.25 2.25 0 01-.659 1.591l-5.432 5.432a2.25 2.25 0 00-.659 1.591v2.927a2.25 2.25 0 01-1.244 2.013L9.75 21v-6.568a2.25 2.25 0 00-.659-1.591L3.659 7.409A2.25 2.25 0 013 5.818V4.774c0-.54.384-1.006.917-1.096A48.32 48.32 0 0112 3z"
                    />
                  </svg>
                  {hasActiveFilters(filters) && (
                    <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-[var(--color-accent-primary)]" />
                  )}
                </button>
              </Tooltip>
              {showFilterMenu && (
                <FilterMenu
                  issues={filteredIssues}
                  filters={filters}
                  onChange={onFiltersChange}
                  anchorRef={filterBtnRef}
                  onClose={() => setShowFilterMenu(false)}
                />
              )}
            </div>
            <div>
              <Tooltip content="View options">
                <button
                  ref={viewBtnRef}
                  onClick={() => setShowViewMenu((v) => !v)}
                  className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
                >
                  <Settings2 size={14} />
                </button>
              </Tooltip>
              {showViewMenu && createPortal(
                <div
                  ref={viewMenuRef}
                  style={{ position: "fixed", visibility: "hidden" }}
                  className="z-50 min-w-44 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1"
                >
                  {childrenByParent.size > 0 && (
                    <>
                      <div className="px-3 py-1.5 text-xs text-[var(--color-text-muted)] font-medium uppercase tracking-wider">
                        Layout
                      </div>

                      <button
                        onClick={() => {
                          if (hierarchyMode !== "flat") toggleHierarchy();
                          setShowViewMenu(false);
                        }}
                        className={`flex items-center gap-2 w-full h-7 px-3 text-sm transition-colors ${hierarchyMode === "flat" ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"}`}
                      >
                        <svg
                          className="w-3.5 h-3.5 shrink-0"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          strokeWidth={1.5}
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            d="M3.75 6h16.5M3.75 12h16.5M3.75 18h16.5"
                          />
                        </svg>
                        Flat
                        {hierarchyMode === "flat" && (
                          <svg
                            className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
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
                      <button
                        onClick={() => {
                          if (hierarchyMode !== "nested") toggleHierarchy();
                          setShowViewMenu(false);
                        }}
                        className={`flex items-center gap-2 w-full h-7 px-3 text-sm transition-colors ${hierarchyMode === "nested" ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"}`}
                      >
                        <ListTree size={14} className="w-3.5 shrink-0" />
                        Nested
                        {hierarchyMode === "nested" && (
                          <svg
                            className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
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
                      {hierarchyMode === "nested" && (
                        <>
                          <div className="my-1 border-t border-[var(--color-border-subtle)]" />
                          <button
                            onClick={() => {
                              expandAllNodes();
                              setShowViewMenu(false);
                            }}
                            className="flex items-center gap-2 w-full h-7 px-3 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                          >
                            <ChevronsUpDown
                              size={14}
                              className="w-3.5 shrink-0"
                            />
                            Expand all
                          </button>
                          <button
                            onClick={() => {
                              collapseAllNodes();
                              setShowViewMenu(false);
                            }}
                            className="flex items-center gap-2 w-full h-7 px-3 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                          >
                            <ChevronsDownUp
                              size={14}
                              className="w-3.5 shrink-0"
                            />
                            Collapse all
                          </button>
                        </>
                      )}
                      <div className="my-1 border-t border-[var(--color-border-subtle)]" />
                    </>
                  )}
                  {!childrenByParent.size && (
                    <div className="px-3 py-1.5 text-xs text-[var(--color-text-muted)] font-medium uppercase tracking-wider">
                      Layout
                    </div>
                  )}
                  <button
                    onClick={() => {
                      toggleEmptyGroups();
                      setShowViewMenu(false);
                    }}
                    className="flex items-center gap-2 w-full h-7 px-3 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                  >
                    Show empty groups
                    {showEmptyGroups && (
                      <svg
                        className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
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
                  {hierarchyMode === "nested" && (
                    <button
                      onClick={() => {
                        toggleGhosts();
                        setShowViewMenu(false);
                      }}
                      className="flex items-center gap-2 w-full h-7 px-3 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                    >
                      Show done ghosts
                      {showGhosts && (
                        <svg
                          className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
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
                  )}
                  <div className="my-1 border-t border-[var(--color-border-subtle)]" />
                  <div className="px-3 py-1.5 text-xs text-[var(--color-text-muted)] font-medium uppercase tracking-wider">
                    Sort by
                  </div>
                  {SORT_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      onClick={() => {
                        onSortChange(opt.value);
                        setShowViewMenu(false);
                      }}
                      className={`flex items-center gap-2 w-full h-7 px-3 text-sm transition-colors ${opt.value === sortKey ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)]"}`}
                    >
                      {opt.label}
                      {opt.value === sortKey && (
                        <svg
                          className="w-3 h-3 ml-auto text-[var(--color-accent-primary)]"
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
                  ))}
                </div>,
                document.body,
              )}
            </div>
            <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
              {filteredIssues.length} issue
              {filteredIssues.length !== 1 ? "s" : ""}
            </span>
          </div>
        }
      />

      {filteredIssues.length === 0 ? (
        <EmptyState
          title={
            hasActiveFilters(filters) || search
              ? "No matching issues"
              : activeTab === "backlog"
                ? "No issues in the backlog"
                : activeTab === "active"
                  ? "No active issues"
                  : activeTab === "done"
                    ? "No completed issues"
                    : "No issues"
          }
          description={
            hasActiveFilters(filters) || search
              ? "Try adjusting your filters or search."
              : activeTab === "backlog"
                ? "Issues with Backlog status will appear here."
                : activeTab === "active"
                  ? "Issues that are Planned, In Progress, or Blocked will appear here."
                  : activeTab === "done"
                    ? "Completed, canceled, and duplicate issues will appear here."
                    : "No issues to display."
          }
          icon={
            <svg width="160" height="120" viewBox="0 0 160 120" fill="none">
              <rect x="30" y="20" width="100" height="14" rx="4" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <rect x="30" y="42" width="100" height="14" rx="4" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <rect x="30" y="64" width="100" height="14" rx="4" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
              <circle cx="80" cy="100" r="2" fill="var(--color-text-muted)" />
            </svg>
          }
        />
      ) : (
      <>
      {/* Rows */}
      <DndContext
        sensors={sensors}
        collisionDetection={backlogCollision}
        onDragStart={handleDndStart}
        onDragOver={handleDndOver}
        onDragMove={handleDndMove}
        onDragEnd={handleDndEnd}
        onDragCancel={resetDropState}
      >
        <div
          ref={listRef}
          className="flex-1 overflow-y-auto relative"
          onClickCapture={(e) => {
            if (wasDraggingRef.current) {
              e.stopPropagation();
              e.preventDefault();
            }
          }}
          onMouseLeave={() => {
            if (!keyboardNav) setFocusedIndex(-1);
          }}
        >
          {(() => {
            const sections: {
              groupRow: RowItem & { kind: "group" };
              groupIndex: number;
              issueRows: { row: RowItem & { kind: "issue" }; index: number }[];
            }[] = [];
            for (let i = 0; i < rows.length; i++) {
              const row = rows[i];
              if (row.kind === "group")
                sections.push({ groupRow: row, groupIndex: i, issueRows: [] });
              else if (sections.length > 0)
                sections[sections.length - 1].issueRows.push({ row, index: i });
            }
            return sections.map(({ groupRow, groupIndex, issueRows }) => {
              const isExpanded = expandedGroups.has(groupRow.status);
              const isInlineActive = inlineCreateStatus === groupRow.status;
              return (
                <div
                  key={`g-${groupRow.status}`}
                  data-group-status={groupRow.status}
                >
                  <BacklogGroupHeader
                    status={groupRow.status}
                    label={groupRow.label}
                    isEmpty={groupRow.isEmpty}
                    isExpanded={isExpanded}
                    count={groupRow.count}
                    storyPoints={groupRow.storyPoints}
                    groupIndex={groupIndex}
                    isFocused={groupIndex === focusedIndex}
                    keyboardNav={keyboardNav}
                    isDndEnabled={isDndEnabled}
                    hasActiveId={activeId !== null}
                    showStoryPoints={showStoryPoints}
                    onToggle={toggleGroup}
                    onToggleStoryPoints={handleToggleStoryPoints}
                    onStartInlineCreate={startInlineCreate}
                    onMouseEnter={handleRowMouseEnter}
                  />
                  {isInlineActive && (
                    <div className="flex items-center gap-3 px-5 h-10 border-b border-[var(--color-border-subtle)] bg-[var(--color-surface-1)]">
                      <span className="w-4 shrink-0" />
                      <div className="shrink-0">
                        <div className="w-6 h-6 -m-1 flex items-center justify-center">
                          <PriorityIcon priority={0} size={16} />
                        </div>
                      </div>
                      <span className="font-mono text-xs text-left shrink-0 tabular-nums text-[var(--color-text-muted)] opacity-40">
                        xpo-······
                      </span>
                      <div className="shrink-0">
                        <div className="w-6 h-6 -m-1 flex items-center justify-center">
                          <StatusIcon
                            status={groupRow.status}
                            size={14}
                          />
                        </div>
                      </div>
                      <input
                        ref={inlineRef}
                        value={inlineTitle}
                        onChange={(e) => setInlineTitle(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter" && inlineTitle.trim())
                            handleInlineCreate(groupRow.status, inlineTitle);
                          if (e.key === "Escape") {
                            setInlineCreateStatus(null);
                            setInlineTitle("");
                          }
                        }}
                        onBlur={() => {
                          if (!inlineTitle.trim()) {
                            setInlineCreateStatus(null);
                            setInlineTitle("");
                          }
                        }}
                        placeholder="New issue title... (Enter to create, Esc to cancel)"
                        className="flex-1 bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
                      />
                    </div>
                  )}
                  {(() => {
                    const groupShowAll = showAllGroups.has(groupRow.status);
                    const shouldCap =
                      issueRows.length > GROUP_VISIBLE_COUNT && !groupShowAll;
                    const visibleRows = shouldCap
                      ? issueRows.slice(0, GROUP_VISIBLE_COUNT)
                      : issueRows;
                    const hiddenCount = issueRows.length - GROUP_VISIBLE_COUNT;
                    return (
                      <>
                        {visibleRows.map(({ row, index: i }) => {
                          const { issue, depth, hasChildren, hasVisibleChildren, childDone, childTotal, childPointsDone, childPointsTotal, parentBreadcrumb, isGhostParent, treeGuides } = row;
                          const isGhostRow = !!isGhostParent || !!row.isGhostChild;
                          return (
                            <BacklogIssueRow
                              key={issue.id}
                              issue={issue}
                              depth={depth}
                              hasChildren={hasChildren}
                              hasVisibleChildren={hasVisibleChildren}
                              childDone={childDone}
                              childTotal={childTotal}
                              childPointsDone={childPointsDone}
                              childPointsTotal={childPointsTotal}
                              parentBreadcrumb={parentBreadcrumb}
                              isGhostParent={isGhostParent}
                              isGhostChild={row.isGhostChild}
                              treeGuides={treeGuides}
                              rowIndex={i}
                              hierarchyMode={hierarchyMode}
                              isFocused={i === focusedIndex || contextMenu?.issueId === issue.id}
                              keyboardNav={keyboardNav}
                              isNodeExpanded={expandedNodes.has(issue.id)}
                              canDrag={isDndEnabled && !isGhostRow}
                              isDropTarget={isDndEnabled && activeId !== null}
                              dndId={isGhostRow ? `ghost:${issue.id}` : issue.id}
                              isDraggedOrBatch={activeId !== null && dragBatchRef.current.includes(issue.id)}
                              dragBatchCount={activeId === issue.id ? dragBatchRef.current.length : 0}
                              popoverType={openPopover?.rowIndex === i ? openPopover.type : null}
                              allKnownLabels={allKnownLabels}
                              cycleNumber={cycleMap.get(issue.cycle_id || "")}
                              onIssueClick={onIssueClick}
                              onToggleNode={toggleNode}
                              onMouseEnter={handleRowMouseEnter}
                              onOpenPopover={handleOpenPopover}
                              onClosePopover={handleClosePopover}
                              onQuickStatus={handleQuickStatus}
                              onQuickPriority={handleQuickPriority}
                              onQuickEstimate={handleQuickEstimate}
                              onQuickLabelToggle={handleQuickLabelToggle}
                              onConfigLabelsChange={onConfigLabelsChange}
                            />
                          );
                        })}
                        {shouldCap && (
                          <button
                            onClick={() =>
                              setShowAllGroups((prev) =>
                                new Set(prev).add(groupRow.status),
                              )
                            }
                            className="w-full py-2 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors border-b border-[var(--color-border-subtle)]"
                          >
                            + {hiddenCount} more
                          </button>
                        )}
                      </>
                    );
                  })()}
                </div>
              );
            });
          })()}
        </div>
        <DragOverlay dropAnimation={null}>
          {draggedIssue ? (
            <DragOverlayCard
              issue={draggedIssue}
              batchCount={dragBatchRef.current.length}
            />
          ) : null}
        </DragOverlay>
      </DndContext>
      {contextMenu &&
        (() => {
          const ctxIssue = issues.find((i) => i.id === contextMenu.issueId);
          if (!ctxIssue) return null;
          return (
            <ContextMenu
              issue={ctxIssue}
              issues={issues}
              x={contextMenu.x}
              y={contextMenu.y}
              onClose={() => setContextMenu(null)}
              onRefresh={onRefresh}
              allLabels={allKnownLabels}
              contributors={contributors}
              onConfigLabelsChange={onConfigLabelsChange}
              patchIssue={patchIssue}
            />
          );
        })()}
      <Modal
        isOpen={!!moveChildrenPrompt}
        onClose={() => setMoveChildrenPrompt(null)}
        title="Update sub-issues?"
        size="xl"
        showCloseButton={false}
      >
        {moveChildrenPrompt &&
          (() => {
            const targetLabel =
              STATUS_LABELS[moveChildrenPrompt.status] ||
              moveChildrenPrompt.status;
            const fullChildren = moveChildrenPrompt.children.map((c) => {
              const full = issues.find((i) => i.id === c.id);
              return full || c;
            });
            return (
              <>
                <p className="text-sm text-[var(--color-text-secondary)] mb-5 leading-6">
                  <span className="text-text-primary">
                    {moveChildrenPrompt.title}
                  </span>{" "}
                  has {moveChildrenPrompt.children.length}{" "}
                  {moveChildrenPrompt.children.length === 1
                    ? "sub-issue "
                    : "sub-issues "}
                  in a different status. Do you want to change their status to{" "}
                  <StatusIcon
                    status={moveChildrenPrompt.status}
                    size={12}
                    className="inline-block align-[-1px] mx-0.5"
                  />
                  <strong className="text-[var(--color-text-primary)]">
                    {targetLabel}
                  </strong>{" "}
                  at the same time?
                  {moveChildrenPrompt.doneCount > 0 && (
                    <>
                      {" "}
                      {moveChildrenPrompt.doneCount} completed{" "}
                      {moveChildrenPrompt.doneCount === 1 ? "issue" : "issues"}{" "}
                      will not be updated.
                    </>
                  )}
                </p>
                <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden mb-8 max-h-64 overflow-y-auto">
                  {fullChildren.map((child, i) => {
                    const rawLabels =
                      "labels" in child ? (child as Issue).labels || [] : [];
                    const { primary: pl, metadata: ml } = splitLabels(
                      rawLabels,
                      defaultLabels,
                    );
                    const allLabels = [...pl, ...ml];
                    const extraCount = Math.max(0, allLabels.length - 2);
                    return (
                      <div
                        key={child.id}
                        className={`flex items-center gap-3 px-4 py-2.5 text-sm ${i > 0 ? "border-t border-[var(--color-border-subtle)]" : ""}`}
                      >
                        <StatusIcon status={child.status} size={14} />
                        <span className="text-[var(--color-text-primary)] truncate min-w-0 flex-1">
                          {child.title}
                        </span>
                        {"priority" in child &&
                          (child as Issue).priority > 0 && (
                            <PriorityIcon
                              priority={(child as Issue).priority}
                              size={14}
                            />
                          )}
                        {allLabels.slice(0, 2).map((label: string) => (
                          <LabelBadge key={label} label={label} />
                        ))}
                        {extraCount > 0 && (
                          <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                            +{extraCount}
                          </span>
                        )}
                        {"estimate" in child &&
                          (child as Issue).estimate > 0 && (
                            <EstimateBadge value={(child as Issue).estimate} />
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
                      const { issueId, status, pendingUpdate } =
                        moveChildrenPrompt;
                      setMoveChildrenPrompt(null);
                      const extra = Object.fromEntries(Object.entries(pendingUpdate || {}).filter(([k]) => k !== "status"));
                      await applyStatusChange(
                        issueId,
                        status,
                        false,
                        Object.keys(extra).length > 0 ? extra : undefined,
                      );
                      showToast("Status changed");
                    }}
                  >
                    Just this issue
                  </button>

                  <button
                    className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-colors"
                    onClick={async () => {
                      const { issueId, status, children, pendingUpdate } =
                        moveChildrenPrompt;
                      setMoveChildrenPrompt(null);
                      const extra = Object.fromEntries(Object.entries(pendingUpdate || {}).filter(([k]) => k !== "status"));
                      await applyStatusChange(
                        issueId,
                        status,
                        true,
                        Object.keys(extra).length > 0 ? extra : undefined,
                      );
                      showToast(
                        `Updated ${children.length + 1} issues to ${targetLabel}`,
                      );
                    }}
                  >
                    Update all
                  </button>
                </div>
              </>
            );
          })()}
      </Modal>
      </>
      )}
    </div>
  );
}
