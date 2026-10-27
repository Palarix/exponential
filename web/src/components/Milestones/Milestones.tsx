import { useMemo, useRef, useState, type HTMLAttributes } from "react";
import {
  closestCenter, DndContext, DragOverlay, PointerSensor, pointerWithin, useDraggable, useDroppable, useSensor, useSensors,
  type CollisionDetection, type DragEndEvent, type DragOverEvent, type DragStartEvent,
} from "@dnd-kit/core";
import { ChevronRight, CircleCheck, CircleCheckBig, GripVertical, Inbox, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";
import type { Issue, Milestone, MilestonesResponse } from "../../api/types";
import { useIssueList, useMilestones } from "../../api/queries";
import { useReorderMilestone } from "../../api/mutations";
import { useMilestoneAssign } from "./milestone-assign-context";
import { useAppNav, useIssueClick, useViewState } from "../../app/hooks";
import { StatusBarSlot } from "../../app/StatusBarSlot";
import { EMPTY_FILTERS, FILTERS_CODEC, hasActiveFilters, matchesFilters, type BacklogFilters } from "../Backlog/filters";
import FilterChips from "../Backlog/FilterChips";
import { cn } from "../../utils/cn";
import { FilterButton, Heading, indicatorPill, Menu, MenuItem, Popover, SearchInput, TopBar, ViewContainer } from "../ui";
import { DragOverlayCard } from "../Backlog/DndComponents";
import NewMilestoneDialog from "./NewMilestoneDialog";
import MilestoneActionDialog from "./MilestoneActionDialog";
import { milestoneMenuItems, type MilestoneAction } from "./milestone-action-utils";
import { ExpandToggle, PastBurndown, MILESTONE_ROW_GRID, MilestoneRowBody, type ProgressMode } from "./MilestoneRow";
import { IssueRow, MilestoneIssueList } from "./MilestoneIssues";
import {
  acceptsIssueDrop, dropHighlightClass, headerClickToggles, milestoneFilter, issuesInMilestone, openMilestones, orderWarnings,
  milestoneMemberCounts, reorderMove, savedOrder, noMilestoneIssues, type DropHighlight,
} from "./milestones-utils";

/** Where the progress pills' issues/points choice is saved. */
const PROGRESS_MODE_KEY = "exponential-milestones-show-points";

/** Past milestones shown before "Show N older". */
const PAST_SHOWN = 3;
/** At most this many No milestone issues render; the search narrows the rest. */
const NO_MILESTONE_LIMIT = 100;

// Drag data. Milestones drag by their handle and drop on other milestones'
// targets; issues drag by their row and drop on any open milestone or on No milestone.
type DragData = { kind: "milestone"; milestone: Milestone } | { kind: "issue"; issue: Issue };
type DropData = { kind: "target"; milestone: Milestone } | { kind: "no-milestone" };

const dragData = (d: unknown) => d as DragData | undefined;
const dropData = (d: unknown) => d as DropData | undefined;

/** Each drag only sees its own targets: milestones reorder among saved milestones, issues drop on milestones or No milestone. */
const milestoneCollision: CollisionDetection = (args) => {
  const kind = dragData(args.active.data.current)?.kind;
  const droppableContainers = args.droppableContainers.filter((c) => {
    const d = dropData(c.data.current);
    if (kind === "milestone") return d?.kind === "target" && !d.milestone.placeholder;
    return !!d;
  });
  const hits = pointerWithin({ ...args, droppableContainers });
  if (hits.length > 0 || kind !== "milestone") return hits;
  return closestCenter({ ...args, droppableContainers });
};

function todayISO(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export default function Milestones() {
  const { data: board, isPending, isError } = useMilestones();
  const { navigate } = useAppNav();

  if (isPending) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="w-5 h-5 border-2 border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }
  if (isError || !board) {
    return <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">Couldn't load milestones.</div>;
  }
  const onOpen = (milestone: Milestone) => {
    const filter = milestoneFilter(milestone);
    if (filter) navigate("backlog", { tab: "all", filters: { milestones: [filter] } });
  };
  return <MilestonesBoard board={board} onOpen={onOpen} />;
}

function MilestonesBoard({ board, onOpen }: { board: MilestonesResponse; onOpen: (milestone: Milestone) => void }) {
  const issues = useIssueList();
  const { assign: assignMilestone } = useMilestoneAssign();
  const reorder = useReorderMilestone();
  const onIssueClick = useIssueClick();
  const [showNew, setShowNew] = useState(false);
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useViewState("exponential-milestones-filters", FILTERS_CODEC);
  const [progressMode, setProgressMode] = useState<ProgressMode>(
    () => localStorage.getItem(PROGRESS_MODE_KEY) === "true" ? "points" : "issues",
  );
  const toggleProgressMode = () => setProgressMode((mode) => {
    const next = mode === "points" ? "issues" : "points";
    localStorage.setItem(PROGRESS_MODE_KEY, String(next === "points"));
    return next;
  });

  return (
    <>
      {hasActiveFilters(filters) && (
        <StatusBarSlot>
          <FilterChips filters={filters} onChange={setFilters} />
        </StatusBarSlot>
      )}
      <ViewContainer
        contentClassName="px-6 pt-5 pb-10"
        topBar={
          <TopBar
            left={<Heading title="Milestones" />}
            center={<SearchInput value={query} onChange={setQuery} placeholder="Find issues" />}
            right={
              <div className="flex items-center gap-2">
                <FilterButton issues={issues} filters={filters} onFiltersChange={setFilters} />
                <button
                  onClick={() => setShowNew(true)}
                  className="flex items-center gap-2 h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  New Milestone
                </button>
              </div>
            }
          />
        }
      >
        <MilestoneStack
          board={board}
          issues={issues}
          today={todayISO()}
          query={query}
          filters={filters}
          onOpen={onOpen}
          onIssueClick={onIssueClick}
          onReorder={(move) => reorder.mutate(move)}
          onAssign={(issue, target) => assignMilestone([issue.id], target)}
          progressMode={progressMode}
          onToggleProgressMode={toggleProgressMode}
        />
        {showNew && <NewMilestoneDialog onClose={() => setShowNew(false)} />}
      </ViewContainer>
    </>
  );
}

export interface MilestoneStackProps {
  board: MilestonesResponse;
  issues: Issue[];
  today: string;
  /** The top bar search: narrows every row's issues and opens the rows with matches. */
  query: string;
  /** The top bar filters: narrow the issue lists like the search. */
  filters?: BacklogFilters;
  /** Opens an open milestone in Issues; past milestones toggle in place. */
  onOpen: (milestone: Milestone) => void;
  onIssueClick: (issue: Issue) => void;
  onReorder: (move: { id: string; before?: string; after?: string }) => void;
  /** Moves a dropped issue to `target` (null is No milestone). */
  onAssign: (issue: Issue, target: Milestone | null) => void;
  /** Milestones (by number) expanded at first. */
  initialExpanded?: number[];
  /** What every progress pill counts; a click on one calls onToggleProgressMode. */
  progressMode?: ProgressMode;
  onToggleProgressMode?: () => void;
}

/**
 * One column with the future on top: the saved open milestones (the last
 * Upcoming milestone first, then Next, then Current), the open issues in no
 * milestone (to drag up into a milestone), then the past milestones.
 */
export function MilestoneStack({
  board, issues, today, query, filters = EMPTY_FILTERS, onOpen, onIssueClick, onReorder, onAssign, initialExpanded = [],
  progressMode = "issues", onToggleProgressMode = () => {},
}: MilestoneStackProps) {
  // Search and filters narrow the lists; counts, breadcrumbs and child progress keep using all issues.
  const listed = useMemo(() => hasActiveFilters(filters) ? issues.filter((i) => matchesFilters(i, filters, board)) : issues, [issues, filters, board]);
  const narrowing = query.trim() !== "" || hasActiveFilters(filters);
  const [expanded, setExpanded] = useState<Set<number>>(() => new Set(initialExpanded));
  const [active, setActive] = useState<DragData | null>(null);
  const [over, setOver] = useState<DropData | null>(null);

  const open = useMemo(() => openMilestones(board), [board]);
  const stack = useMemo(() => open.filter((r) => !r.placeholder).reverse(), [open]);
  const orderIds = useMemo(() => savedOrder(board).map((r) => r.id!), [board]);
  const warnings = useMemo(() => orderWarnings(open), [open]);

  const toggle = (number: number) => setExpanded((prev) => {
    const next = new Set(prev);
    if (next.has(number)) next.delete(number); else next.add(number);
    return next;
  });

  // The stack shows the order upside down: a milestone moved later in the
  // order lands above its target.
  const highlightFor = (target: Milestone | null): DropHighlight => {
    if (!active || !over) return null;
    const overMilestone = over.kind === "target" ? over.milestone : null;
    if (over.kind === "no-milestone" ? target !== null : overMilestone?.number !== target?.number) return null;
    if (active.kind === "issue") return acceptsIssueDrop(target, active.issue) ? "drop" : null;
    if (!target?.id || target.id === active.milestone.id) return null;
    const move = reorderMove(orderIds, active.milestone.id!, target.id);
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
    if (drag.kind === "milestone") {
      if (drop.kind !== "target" || !drop.milestone.id) return;
      const move = reorderMove(orderIds, drag.milestone.id!, drop.milestone.id);
      if (move) onReorder(move);
      return;
    }
    const target = drop.kind === "target" ? drop.milestone : null;
    if (acceptsIssueDrop(target, drag.issue)) onAssign(drag.issue, target);
  };

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={milestoneCollision}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragEnd={onDragEnd}
      onDragCancel={onDragCancel}
    >
      <div className="space-y-1.5">
        {stack.map((milestone) => (
          <MilestoneItem
            key={milestone.number}
            board={board}
            milestone={milestone}
            today={today}
            issues={issues}
            listed={listed}
            query={query}
            narrowing={narrowing}
            warning={warnings.get(milestone.number)}
            expanded={expanded.has(milestone.number)}
            onToggle={() => toggle(milestone.number)}
            onOpen={() => onOpen(milestone)}
            onIssueClick={onIssueClick}
            highlight={highlightFor(milestone)}
            progressMode={progressMode}
            onToggleProgressMode={onToggleProgressMode}
          />
        ))}
        <NoMilestoneRow
          issues={issues}
          listed={listed}
          query={query}
          narrowing={narrowing}
          onIssueClick={onIssueClick}
          highlight={highlightFor(null)}
        />
        <PastMilestones
          milestones={board.past}
          issues={issues}
          listed={listed}
          query={query}
          narrowing={narrowing}
          today={today}
          expanded={expanded}
          onToggle={toggle}
          onIssueClick={onIssueClick}
          progressMode={progressMode}
          onToggleProgressMode={onToggleProgressMode}
        />
      </div>

      <DragOverlay dropAnimation={null}>
        {active?.kind === "issue" && <DragOverlayCard issue={active.issue} batchCount={1} />}
        {active?.kind === "milestone" && (
          <div className="inline-flex items-center gap-2 px-3 h-9 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-elevated)] shadow-lg text-sm font-medium text-[var(--color-text-primary)] pointer-events-none">
            <GripVertical size={14} className="text-[var(--color-text-muted)]" />
            {active.milestone.name}
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
}

interface MilestoneItemProps {
  board: MilestonesResponse;
  milestone: Milestone;
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
  progressMode: ProgressMode;
  onToggleProgressMode: () => void;
}

/**
 * While the search or the filters narrow the lists, a row with matches opens
 * and lists only them; one without stays collapsed. Clearing both restores
 * the user's own expansion.
 */
function shownExpanded(expanded: boolean, narrowing: boolean, matches: Issue[]): boolean {
  return narrowing ? matches.length > 0 : expanded;
}

/** A saved open milestone: a drop target, draggable by its handle, with its actions. */
function MilestoneItem({ board, milestone, issues, listed, query, narrowing, expanded, onIssueClick, ...props }: MilestoneItemProps) {
  const [dialog, setDialog] = useState<MilestoneAction | "edit" | null>(null);
  const milestoneIssues = issuesInMilestone(listed, milestone.id, query);
  const { setNodeRef } = useDroppable({ id: `target:${milestone.number}`, data: { kind: "target", milestone } satisfies DropData });
  const { attributes, listeners, isDragging } = useDraggable({
    id: `milestone:${milestone.number}`,
    data: { kind: "milestone", milestone } satisfies DragData,
    disabled: !!milestone.placeholder,
  });
  const handle = milestone.placeholder ? null : (
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
      <MilestoneRowBody
        {...props}
        milestone={milestone}
        handle={handle}
        expanded={shownExpanded(expanded, narrowing, milestoneIssues)}
        actions={<MilestoneMenu milestone={milestone} onPick={setDialog} />}
      >
        <MilestoneIssueList
          issues={milestoneIssues}
          all={issues}
          onIssueClick={onIssueClick}
          empty="No issues yet. Drag issues here from No milestone."
          renderRow={(issue) => <DraggableIssueRow key={issue.id} issue={issue} all={issues} onIssueClick={onIssueClick} />}
        />
      </MilestoneRowBody>
      {dialog === "edit" && <NewMilestoneDialog milestone={milestone} onClose={() => setDialog(null)} />}
      {(dialog === "complete" || dialog === "delete") && (
        <MilestoneActionDialog
          action={dialog}
          board={board}
          milestone={milestone}
          count={milestoneMemberCounts(issues, milestone.id)[dialog === "complete" ? "open" : "all"]}
          onClose={() => setDialog(null)}
        />
      )}
    </div>
  );
}

/** The `⋯` menu of a saved open milestone: Edit, Complete milestone, Delete; on Current, Complete milestone first. */
function MilestoneMenu({ milestone, onPick }: { milestone: Milestone; onPick: (action: MilestoneAction | "edit") => void }) {
  const [open, setOpen] = useState(false);
  const anchorRef = useRef<HTMLButtonElement>(null);
  const pick = (action: MilestoneAction | "edit") => { setOpen(false); onPick(action); };
  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        aria-label={`Actions for ${milestone.name}`}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className="p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
      >
        <MoreHorizontal size={14} />
      </button>
      {open && (
        <Popover anchorRef={anchorRef} onClose={() => setOpen(false)} placement="bottom-end">
          <Menu onClose={() => setOpen(false)} aria-label={`Actions for ${milestone.name}`}>
            {milestoneMenuItems(milestone).map((item) => {
              switch (item) {
                case "complete": return <MenuItem key={item} label="Complete milestone" icon={<CircleCheckBig size={14} />} onClick={() => pick(item)} />;
                case "edit": return <MenuItem key={item} label="Edit" icon={<Pencil size={14} />} onClick={() => pick(item)} />;
                case "delete": return <MenuItem key={item} label="Delete" icon={<Trash2 size={14} />} destructive onClick={() => pick(item)} />;
              }
            })}
          </Menu>
        </Popover>
      )}
    </>
  );
}

/** An issue to drag into (or out of) a milestone. */
function DraggableIssueRow(props: { issue: Issue; all: Issue[]; onIssueClick: (issue: Issue) => void }) {
  const { setNodeRef, attributes, listeners, isDragging } = useDraggable({
    id: `issue:${props.issue.id}`,
    data: { kind: "issue", issue: props.issue } satisfies DragData,
  });
  return <IssueRow {...props} setNodeRef={setNodeRef} dragProps={{ ...attributes, ...listeners } as HTMLAttributes<HTMLDivElement>} dragging={isDragging} />;
}

/**
 * Open issues in no milestone, as the row under Current: an inbox icon
 * where milestones have their handle, the count where they have progress. It
 * starts expanded, toggles on a header click like the milestones, and lists the issues in the same rows; under
 * a search it opens only when something matches. The whole row is the drop
 * target that takes an issue out of its milestone.
 */
function NoMilestoneRow({ issues, listed, query, narrowing, onIssueClick, highlight }: {
  issues: Issue[];
  listed: Issue[];
  query: string;
  narrowing: boolean;
  onIssueClick: (issue: Issue) => void;
  highlight: DropHighlight;
}) {
  const [expanded, setExpanded] = useState(true);
  const { setNodeRef } = useDroppable({ id: "no-milestone", data: { kind: "no-milestone" } satisfies DropData });
  const all = useMemo(() => noMilestoneIssues(issues, ""), [issues]);
  const shown = useMemo(() => noMilestoneIssues(listed, query), [listed, query]);
  const open = shownExpanded(expanded, narrowing, shown);

  return (
    <section
      ref={setNodeRef}
      className={cn(
        "rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-1)] transition-shadow",
        dropHighlightClass(highlight),
      )}
    >
      <div
        className={cn(MILESTONE_ROW_GRID, "px-3 h-11 cursor-pointer")}
        onClick={(e) => { if (headerClickToggles(e.target as Element, e.currentTarget)) setExpanded(!expanded); }}
      >
        <Inbox size={14} className="text-[var(--color-text-muted)]" />
        <span className="flex items-center gap-2 min-w-0">
          <span className="min-w-0 text-sm truncate">
            <span className="font-medium text-[var(--color-text-primary)]">No milestone</span>
            <span className="text-[var(--color-text-muted)]"> – Open issues in no milestone</span>
          </span>
          <span title={`${all.length} open issues in no milestone`} className={cn(indicatorPill, "shrink-0")}>{all.length}</span>
        </span>
        <span />
        <span />
        <span />
        <span />
        <ExpandToggle expanded={open} onToggle={() => setExpanded(!expanded)} />
      </div>
      {open && (
        <div className="px-3 pb-3">
          {shown.length === 0 ? (
            <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">Every open issue is in a milestone.</p>
          ) : (
            <div className="border-t border-[var(--color-border-subtle)] pt-2 space-y-px">
              {shown.slice(0, NO_MILESTONE_LIMIT).map((issue) => (
                <DraggableIssueRow key={issue.id} issue={issue} all={issues} onIssueClick={onIssueClick} />
              ))}
              {shown.length > NO_MILESTONE_LIMIT && (
                <p className="px-2 py-2 text-xs text-[var(--color-text-muted)]">
                  {shown.length - NO_MILESTONE_LIMIT} more. Search to find them.
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
 * Completed milestones, newest first, as faded rows. They are neither dragged
 * nor drop targets, but expand (by the chevron or the title) to their record:
 * the burndown (hidden while the search or filters narrow the lists), then
 * what they delivered. Only the most
 * recent few show until "Show N older".
 */
function PastMilestones({
  milestones, issues, listed, query, narrowing, today, expanded, onToggle, onIssueClick, progressMode, onToggleProgressMode,
}: {
  milestones: Milestone[];
  issues: Issue[];
  listed: Issue[];
  query: string;
  narrowing: boolean;
  today: string;
  expanded: Set<number>;
  onToggle: (number: number) => void;
  onIssueClick: (issue: Issue) => void;
  progressMode: ProgressMode;
  onToggleProgressMode: () => void;
}) {
  const [showAll, setShowAll] = useState(false);
  const older = milestones.length - PAST_SHOWN;
  const shown = showAll ? milestones : milestones.slice(0, PAST_SHOWN);
  return (
    <>
      {shown.map((r) => {
        const delivered = issuesInMilestone(listed, r.id, query);
        return (
          <MilestoneRowBody
            key={r.number}
            milestone={r}
            today={today}
            handle={<CircleCheck size={14} className="text-[var(--color-text-muted)]" />}
            expanded={shownExpanded(expanded.has(r.number), narrowing, delivered)}
            onToggle={() => onToggle(r.number)}
            onOpen={() => onToggle(r.number)}
            progressMode={progressMode}
            onToggleProgressMode={onToggleProgressMode}
          >
            {!narrowing && <PastBurndown milestone={r} />}
            <MilestoneIssueList issues={delivered} all={issues} onIssueClick={onIssueClick} />
          </MilestoneRowBody>
        );
      })}
      {older > 0 && (
        <button
          type="button"
          onClick={() => setShowAll(!showAll)}
          aria-expanded={showAll}
          className={cn(MILESTONE_ROW_GRID, "w-full px-3 h-9 text-left text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors")}
        >
          <ChevronRight size={14} className={cn("transition-transform", showAll ? "-rotate-90" : "rotate-90")} />
          <span>{showAll ? "Show fewer" : `Show ${older} older ${older === 1 ? "milestone" : "milestones"}`}</span>
        </button>
      )}
    </>
  );
}
