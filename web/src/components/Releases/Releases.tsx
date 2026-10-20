import { useCallback, useEffect, useMemo, useRef, useState, type HTMLAttributes } from "react";
import {
  closestCenter, DndContext, DragOverlay, PointerSensor, pointerWithin, useDraggable, useDroppable, useSensor, useSensors,
  type CollisionDetection, type DragEndEvent, type DragOverEvent, type DragStartEvent,
} from "@dnd-kit/core";
import { ChevronRight, CircleCheck, GripVertical, Inbox, Plus } from "lucide-react";
import type { Issue, Release, ReleasesResponse } from "../../api/types";
import { useIssueList, useReleases } from "../../api/queries";
import { useReorderRelease } from "../../api/mutations";
import { useReleaseAssign } from "./release-assign-context";
import { useAppNav, useIssueClick, useViewState } from "../../app/hooks";
import { StatusBarSlot } from "../../app/StatusBarSlot";
import { EMPTY_FILTERS, FILTERS_CODEC, hasActiveFilters, matchesFilters, type BacklogFilters } from "../Backlog/filters";
import FilterChips from "../Backlog/FilterChips";
import { isTerminal } from "../../constants";
import { cn } from "../../utils/cn";
import { CopyableId, FilterButton, Heading, indicatorPill, SearchInput, StatusIcon, SubProgress, TopBar, ViewContainer } from "../ui";
import Tooltip from "../ui/Tooltip";
import { DragOverlayCard } from "../Backlog/DndComponents";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { IssuePreview } from "../IssueRef/IssuePreview";
import NewReleaseDialog from "./NewReleaseDialog";
import ReleaseDetail from "./ReleaseDetail";
import { ExpandToggle, RELEASE_ROW_GRID, ReleaseRowBody } from "./ReleaseRow";
import {
  acceptsIssueDrop, childProgress, dropHighlightClass, findRelease, parentContext, issuesInRelease, openReleases, orderWarnings,
  reorderMove, savedOrder, noReleaseIssues, type DropHighlight,
} from "./releases-utils";

/** Past releases shown before "Show N older". */
const PAST_SHOWN = 3;
/** At most this many No release issues render; the search narrows the rest. */
const NO_RELEASE_LIMIT = 100;
/** How long the pointer rests on an issue row before its hover card opens (as on issue chips). */
const PREVIEW_DELAY = 250;

// Drag data. Releases drag by their handle and drop on other releases'
// targets; issues drag by their row and drop on any open release or on No release.
type DragData = { kind: "release"; release: Release } | { kind: "issue"; issue: Issue };
type DropData = { kind: "target"; release: Release } | { kind: "no-release" };

const dragData = (d: unknown) => d as DragData | undefined;
const dropData = (d: unknown) => d as DropData | undefined;

/** Each drag only sees its own targets: releases reorder among saved releases, issues drop on releases or No release. */
const releaseCollision: CollisionDetection = (args) => {
  const kind = dragData(args.active.data.current)?.kind;
  const droppableContainers = args.droppableContainers.filter((c) => {
    const d = dropData(c.data.current);
    if (kind === "release") return d?.kind === "target" && !d.release.placeholder;
    return !!d;
  });
  const hits = pointerWithin({ ...args, droppableContainers });
  if (hits.length > 0 || kind !== "release") return hits;
  return closestCenter({ ...args, droppableContainers });
};

function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export default function Releases() {
  const { data: board, isPending, isError } = useReleases();
  const { releaseNumber, setReleaseNumber } = useAppNav();

  if (isPending) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="w-5 h-5 border-2 border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }
  if (isError || !board) {
    return <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">Couldn't load releases.</div>;
  }
  if (releaseNumber != null) {
    return <ReleaseDetail number={releaseNumber} release={findRelease(board, releaseNumber)} today={todayISO()} onBack={() => setReleaseNumber(null)} />;
  }
  return <ReleasesBoard board={board} onOpen={(r) => setReleaseNumber(r.number)} />;
}

function ReleasesBoard({ board, onOpen }: { board: ReleasesResponse; onOpen: (release: Release) => void }) {
  const issues = useIssueList();
  const { assign: assignRelease } = useReleaseAssign();
  const reorder = useReorderRelease();
  const onIssueClick = useIssueClick();
  const [showNew, setShowNew] = useState(false);
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useViewState("exponential-releases-filters", FILTERS_CODEC);

  return (
    <>
      {hasActiveFilters(filters) && (
        <StatusBarSlot>
          <FilterChips filters={filters} onChange={setFilters} />
        </StatusBarSlot>
      )}
      <ViewContainer
        maxWidth="7xl"
        innerClassName="px-6 pt-5 pb-10"
        topBar={
          <TopBar
            left={<Heading title="Releases" />}
            center={<SearchInput value={query} onChange={setQuery} placeholder="Find issues" />}
            right={
              <div className="flex items-center gap-2">
                <FilterButton issues={issues} filters={filters} onFiltersChange={setFilters} />
                <button
                  onClick={() => setShowNew(true)}
                  className="flex items-center gap-2 h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  New Release
                </button>
              </div>
            }
          />
        }
      >
        <ReleaseStack
          board={board}
          issues={issues}
          today={todayISO()}
          query={query}
          filters={filters}
          onOpen={onOpen}
          onIssueClick={onIssueClick}
          onReorder={(move) => reorder.mutate(move)}
          onAssign={(issue, target) => assignRelease([issue.id], target)}
        />
        {showNew && <NewReleaseDialog onClose={() => setShowNew(false)} />}
      </ViewContainer>
    </>
  );
}

export interface ReleaseStackProps {
  board: ReleasesResponse;
  issues: Issue[];
  today: string;
  /** The top bar search: narrows every row's issues and opens the rows with matches. */
  query: string;
  /** The top bar filters: narrow the issue lists like the search. */
  filters?: BacklogFilters;
  onOpen: (release: Release) => void;
  onIssueClick: (issue: Issue) => void;
  onReorder: (move: { id: string; before?: string; after?: string }) => void;
  /** Moves a dropped issue to `target` (null is No release). */
  onAssign: (issue: Issue, target: Release | null) => void;
  /** Releases (by number) expanded at first. */
  initialExpanded?: number[];
}

/**
 * One column with the future on top: the saved open releases (the last
 * Upcoming release first, then Next, then Current), the open issues in no
 * release (to drag up into a release), then the past releases.
 */
export function ReleaseStack({
  board, issues, today, query, filters = EMPTY_FILTERS, onOpen, onIssueClick, onReorder, onAssign, initialExpanded = [],
}: ReleaseStackProps) {
  // Search and filters narrow the lists; counts, breadcrumbs and child progress keep using all issues.
  const listed = useMemo(() => hasActiveFilters(filters) ? issues.filter((i) => matchesFilters(i, filters)) : issues, [issues, filters]);
  const narrowing = query.trim() !== "" || hasActiveFilters(filters);
  const [expanded, setExpanded] = useState<Set<number>>(() => new Set(initialExpanded));
  const [active, setActive] = useState<DragData | null>(null);
  const [over, setOver] = useState<DropData | null>(null);

  const open = useMemo(() => openReleases(board), [board]);
  const stack = useMemo(() => open.filter((r) => !r.placeholder).reverse(), [open]);
  const orderIds = useMemo(() => savedOrder(board).map((r) => r.id!), [board]);
  const warnings = useMemo(() => orderWarnings(open), [open]);

  const toggle = (number: number) => setExpanded((prev) => {
    const next = new Set(prev);
    if (next.has(number)) next.delete(number); else next.add(number);
    return next;
  });

  // The stack shows the order upside down: a release moved later in the
  // order lands above its target.
  const highlightFor = (target: Release | null): DropHighlight => {
    if (!active || !over) return null;
    const overRelease = over.kind === "target" ? over.release : null;
    if (over.kind === "no-release" ? target !== null : overRelease?.number !== target?.number) return null;
    if (active.kind === "issue") return acceptsIssueDrop(target, active.issue) ? "drop" : null;
    if (!target?.id || target.id === active.release.id) return null;
    const move = reorderMove(orderIds, active.release.id!, target.id);
    return move?.after ? "before" : move?.before ? "after" : null;
  };

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 3 } }));

  const onDragStart = (e: DragStartEvent) => setActive(dragData(e.active.data.current) ?? null);
  const onDragOver = (e: DragOverEvent) => setOver(dropData(e.over?.data.current) ?? null);
  const onDragCancel = () => { setActive(null); setOver(null); };
  const onDragEnd = (e: DragEndEvent) => {
    const drag = dragData(e.active.data.current);
    const drop = dropData(e.over?.data.current);
    onDragCancel();
    if (!drag || !drop) return;
    if (drag.kind === "release") {
      if (drop.kind !== "target" || !drop.release.id) return;
      const move = reorderMove(orderIds, drag.release.id!, drop.release.id);
      if (move) onReorder(move);
      return;
    }
    const target = drop.kind === "target" ? drop.release : null;
    if (acceptsIssueDrop(target, drag.issue)) onAssign(drag.issue, target);
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={releaseCollision}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragEnd={onDragEnd}
      onDragCancel={onDragCancel}
    >
      <div className="space-y-1.5">
        {stack.map((release) => (
          <ReleaseItem
            key={release.number}
            release={release}
            today={today}
            issues={issues}
            listed={listed}
            query={query}
            narrowing={narrowing}
            warning={warnings.get(release.number)}
            expanded={expanded.has(release.number)}
            onToggle={() => toggle(release.number)}
            onOpen={() => onOpen(release)}
            onIssueClick={onIssueClick}
            highlight={highlightFor(release)}
          />
        ))}
        <NoReleaseRow
          issues={issues}
          listed={listed}
          query={query}
          narrowing={narrowing}
          onIssueClick={onIssueClick}
          highlight={highlightFor(null)}
        />
        <PastReleases
          releases={board.past}
          issues={issues}
          listed={listed}
          query={query}
          narrowing={narrowing}
          today={today}
          expanded={expanded}
          onToggle={toggle}
          onOpen={onOpen}
          onIssueClick={onIssueClick}
        />
      </div>

      <DragOverlay dropAnimation={null}>
        {active?.kind === "issue" && <DragOverlayCard issue={active.issue} batchCount={1} />}
        {active?.kind === "release" && (
          <div className="inline-flex items-center gap-2 px-3 h-9 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-elevated)] shadow-lg text-sm font-medium text-[var(--color-text-primary)] pointer-events-none">
            <GripVertical size={14} className="text-[var(--color-text-muted)]" />
            {active.release.name}
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
}

interface ReleaseItemProps {
  release: Release;
  today: string;
  issues: Issue[];
  /** The issues the filters let through. */
  listed: Issue[];
  query: string;
  /** Search or filters are active. */
  narrowing: boolean;
  warning?: string[];
  expanded: boolean;
  onToggle: () => void;
  onOpen: () => void;
  onIssueClick: (issue: Issue) => void;
  highlight: DropHighlight;
}

/** An open release: a drop target and, unless it is a placeholder, draggable by its handle. */
/**
 * While the search or the filters narrow the lists, a row with matches opens
 * and lists only them; one without stays collapsed. Clearing both restores
 * the user's own expansion.
 */
function shownExpanded(expanded: boolean, narrowing: boolean, matches: Issue[]): boolean {
  return narrowing ? matches.length > 0 : expanded;
}

function ReleaseItem({ release, issues, listed, query, narrowing, expanded, onIssueClick, ...props }: ReleaseItemProps) {
  const releaseIssues = issuesInRelease(listed, release.id, query);
  const { setNodeRef } = useDroppable({ id: `target:${release.number}`, data: { kind: "target", release } satisfies DropData });
  const { attributes, listeners, isDragging } = useDraggable({
    id: `release:${release.number}`,
    data: { kind: "release", release } satisfies DragData,
    disabled: !!release.placeholder,
  });
  const handle = release.placeholder ? null : (
    <button
      type="button"
      aria-label="Drag to reorder"
      className="-ml-1 p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] cursor-grab active:cursor-grabbing touch-none"
      {...attributes}
      {...listeners}
    >
      <GripVertical size={14} />
    </button>
  );
  return (
    <div ref={setNodeRef} className={cn(isDragging && "opacity-40")}>
      <ReleaseRowBody {...props} release={release} handle={handle} expanded={shownExpanded(expanded, narrowing, releaseIssues)}>
        <ReleaseIssueList issues={releaseIssues} all={issues} onIssueClick={onIssueClick} />
      </ReleaseRowBody>
    </div>
  );
}

/** A release's issues, flat. A past release's list is read-only: what it delivered, not to be moved. */
function ReleaseIssueList({ issues, all, onIssueClick, readOnly = false }: {
  issues: Issue[];
  all: Issue[];
  onIssueClick: (issue: Issue) => void;
  readOnly?: boolean;
}) {
  if (issues.length === 0) {
    return (
      <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">
        {readOnly ? "No issues yet." : "No issues yet. Drag issues here from No release."}
      </p>
    );
  }
  return (
    <div className="border-t border-[var(--color-border-subtle)] pt-2 space-y-px">
      {issues.map((issue) => readOnly
        ? <IssueRow key={issue.id} issue={issue} all={all} onIssueClick={onIssueClick} />
        : <DraggableIssueRow key={issue.id} issue={issue} all={all} onIssueClick={onIssueClick} />)}
    </div>
  );
}

/** An issue to drag into (or out of) a release. */
function DraggableIssueRow(props: { issue: Issue; all: Issue[]; onIssueClick: (issue: Issue) => void }) {
  const { setNodeRef, attributes, listeners, isDragging } = useDraggable({
    id: `issue:${props.issue.id}`,
    data: { kind: "issue", issue: props.issue } satisfies DragData,
  });
  return <IssueRow {...props} setNodeRef={setNodeRef} dragProps={{ ...attributes, ...listeners } as HTMLAttributes<HTMLDivElement>} dragging={isDragging} />;
}

/**
 * One issue, compact for the half-width lists but in the backlog's order and
 * styles: ID, status, the parent as a "Parent ›" breadcrumb, the title, and
 * the sub-issue progress pill. Priority, labels and the work indicators stay
 * in the backlog; the hover card has the details. The breadcrumb gives way
 * before the title does. A click opens the issue. The row is a div, not a
 * button: it holds CopyableId, which is a button itself. Resting on the row
 * shows the issue hover card beside it, which closes when a drag starts.
 */
function IssueRow({ issue, all, onIssueClick, setNodeRef, dragProps, dragging = false }: {
  issue: Issue;
  /** Every issue, for the parent breadcrumb and the child progress. */
  all: Issue[];
  onIssueClick: (issue: Issue) => void;
  setNodeRef?: (el: HTMLElement | null) => void;
  dragProps?: HTMLAttributes<HTMLDivElement>;
  dragging?: boolean;
}) {
  const parent = parentContext(issue, all);
  const children = childProgress(issue, all);
  const rowRef = useRef<HTMLDivElement | null>(null);
  const [preview, setPreview] = useState(false);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);
  const setRefs = useCallback((el: HTMLDivElement | null) => {
    rowRef.current = el;
    setNodeRef?.(el);
  }, [setNodeRef]);
  const open = () => { intent.leave(); onIssueClick(issue); };
  return (
    <>
      <div
        ref={setRefs}
        role="button"
        tabIndex={0}
        onClick={open}
        onKeyDown={(e) => { if (e.key === "Enter") open(); }}
        onMouseEnter={() => intent.enter()}
        onMouseLeave={() => intent.leave()}
        onPointerDownCapture={() => intent.leave()}
        className={cn(
          "w-full flex items-center gap-3 px-3 h-10 rounded-[var(--radius-md)] cursor-pointer select-none transition-colors hover:bg-[var(--color-hover-surface-1)]",
          dragProps && "touch-none",
          isTerminal(issue.status) && "opacity-60",
          dragging && "opacity-30",
        )}
        {...dragProps}
      >
        <CopyableId id={issue.id} className="text-xs text-left shrink-0 tabular-nums" />
        <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />
        <span className="flex items-center gap-2 flex-1 min-w-0">
          {parent && <ParentCrumb parent={parent.parent} where={parent.where} />}
          <span className="text-sm text-[var(--color-text-primary)] truncate min-w-0 flex-1">{issue.title}</span>
        </span>
        {children && (
          <span className={indicatorPill}>
            <SubProgress done={children.done} total={children.total} size={12} />
            {children.done}/{children.total}
          </span>
        )}
      </div>
      {preview && !dragging && <IssuePreview anchorRef={rowRef} issue={issue} />}
    </>
  );
}

/**
 * A sub-issue's parent as the backlog shows it, "Parent ›" before the title;
 * the tooltip names the parent and the release it is in.
 */
function ParentCrumb({ parent, where }: { parent: Issue; where: string }) {
  const text = `Parent: ${parent.id} ${parent.title} · ${where}`;
  return (
    <Tooltip content={text}>
      <span aria-label={text} className="flex items-center gap-2 min-w-0 max-w-[40%] shrink-[3]">
        <span className="text-sm text-[var(--color-text-muted)] truncate">{parent.title}</span>
        <ChevronRight className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
      </span>
    </Tooltip>
  );
}

/**
 * Open issues in no release, as the row under Current: an inbox icon
 * where releases have their handle, the count where they have progress. It
 * starts expanded and lists the issues in the same rows as the releases; under
 * a search it opens only when something matches. The whole row is the drop
 * target that takes an issue out of its release.
 */
function NoReleaseRow({ issues, listed, query, narrowing, onIssueClick, highlight }: {
  issues: Issue[];
  listed: Issue[];
  query: string;
  narrowing: boolean;
  onIssueClick: (issue: Issue) => void;
  highlight: DropHighlight;
}) {
  const [expanded, setExpanded] = useState(true);
  const { setNodeRef } = useDroppable({ id: "no-release", data: { kind: "no-release" } satisfies DropData });
  const all = useMemo(() => noReleaseIssues(issues, ""), [issues]);
  const shown = useMemo(() => noReleaseIssues(listed, query), [listed, query]);
  const open = shownExpanded(expanded, narrowing, shown);

  return (
    <section
      ref={setNodeRef}
      className={cn(
        "rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-1)] transition-shadow",
        dropHighlightClass(highlight),
      )}
    >
      <div className={cn(RELEASE_ROW_GRID, "px-3 h-11")}>
        <Inbox size={14} className="text-[var(--color-text-muted)]" />
        <button type="button" onClick={() => setExpanded(!expanded)} className="min-w-0 text-left text-sm truncate">
          <span className="font-medium text-[var(--color-text-primary)]">No release</span>
          <span className="text-[var(--color-text-muted)]"> – Open issues in no release</span>
        </button>
        <span />
        <span />
        <span title={`${all.length} open issues in no release`} className={cn(indicatorPill, "justify-self-end")}>{all.length}</span>
        <ExpandToggle expanded={open} onToggle={() => setExpanded(!expanded)} />
      </div>
      {open && (
        <div className="px-3 pb-3">
          {shown.length === 0 ? (
            <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">Every open issue is in a release.</p>
          ) : (
            <div className="border-t border-[var(--color-border-subtle)] pt-2 space-y-px">
              {shown.slice(0, NO_RELEASE_LIMIT).map((issue) => (
                <DraggableIssueRow key={issue.id} issue={issue} all={issues} onIssueClick={onIssueClick} />
              ))}
              {shown.length > NO_RELEASE_LIMIT && (
                <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">
                  {shown.length - NO_RELEASE_LIMIT} more. Search to find them.
                </p>
              )}
            </div>
          )}
        </div>
      )}
    </section>
  );
}

/**
 * Released releases, newest first, as faded rows. They are neither dragged
 * nor drop targets, but expand to list what they delivered. Only the most
 * recent few show until "Show N older".
 */
function PastReleases({ releases, issues, listed, query, narrowing, today, expanded, onToggle, onOpen, onIssueClick }: {
  releases: Release[];
  issues: Issue[];
  listed: Issue[];
  query: string;
  narrowing: boolean;
  today: string;
  expanded: Set<number>;
  onToggle: (number: number) => void;
  onOpen: (release: Release) => void;
  onIssueClick: (issue: Issue) => void;
}) {
  const [showAll, setShowAll] = useState(false);
  const older = releases.length - PAST_SHOWN;
  const shown = showAll ? releases : releases.slice(0, PAST_SHOWN);
  return (
    <>
      {shown.map((r) => {
        const delivered = issuesInRelease(listed, r.id, query);
        return (
          <ReleaseRowBody
            key={r.number}
            release={r}
            today={today}
            handle={<CircleCheck size={14} className="text-[var(--color-text-muted)]" />}
            expanded={shownExpanded(expanded.has(r.number), narrowing, delivered)}
            onToggle={() => onToggle(r.number)}
            onOpen={() => onOpen(r)}
          >
            <ReleaseIssueList issues={delivered} all={issues} onIssueClick={onIssueClick} readOnly />
          </ReleaseRowBody>
        );
      })}
      {older > 0 && (
        <button
          type="button"
          onClick={() => setShowAll(!showAll)}
          aria-expanded={showAll}
          className={cn(RELEASE_ROW_GRID, "w-full px-3 h-9 text-left text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors")}
        >
          <ChevronRight size={14} className={cn("transition-transform", showAll ? "-rotate-90" : "rotate-90")} />
          <span>{showAll ? "Show fewer" : `Show ${older} older ${older === 1 ? "release" : "releases"}`}</span>
        </button>
      )}
    </>
  );
}
