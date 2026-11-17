import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { ChevronRight, Triangle, TriangleAlert } from "lucide-react";
import type { Milestone, MilestoneChangeKind, MilestoneProgressResponse } from "../../api/types";
import { useMilestoneProgress } from "../../api/queries";
import { formatShortDate } from "../../utils/format";
import { cn } from "../../utils/cn";
import { Popover, PopoverPanel, raisedIndicatorPill, StatusIcon, SubProgress } from "../ui";
import Tooltip from "../ui/Tooltip";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { dropHighlightClass, headerClickToggles, milestoneNameClass, timeBar, type DropHighlight } from "./milestones-utils";
import { burnUpPoints, type BurnUpPoint } from "./burnup-utils";
import { useBurnUpAxis, useProgressMode, type BurnUpAxis, type ProgressMode } from "./milestone-settings";

export type { ProgressMode } from "./milestone-settings";

const STAT_ORDER = ["DOING", "BLOCKED", "PLANNED", "BACKLOG", "DONE"];
const STATUS_NAMES: Record<string, string> = {
  DOING: "In progress", BLOCKED: "Blocked", PLANNED: "Planned", BACKLOG: "Backlog", DONE: "Done",
};

/**
 * The columns every milestone row shares, so the stack reads as a table:
 * handle, title (with the progress), warning, target date, Current/Next
 * badge, actions menu, and the expand chevron always rightmost. Below a
 * stack width of 40rem the warning and date columns drop out (their cells
 * use NARROW_HIDDEN). Cells align to the first line: an expanded title can
 * take a second line.
 */
export const MILESTONE_ROW_GRID =
  "grid grid-cols-[16px_minmax(0,1fr)_4rem_20px_20px] @[40rem]:grid-cols-[16px_minmax(0,1fr)_16px_4.5rem_4rem_20px_20px] items-start gap-x-3";

/** A row cell: one line high, its content centred on that line. */
export const ROW_CELL = "flex items-center min-h-6";

/** A cell that only shows when the stack is at least 40rem wide. */
export const NARROW_HIDDEN = "hidden @[40rem]:flex";

/** The header of a stack row: MILESTONE_ROW_GRID, with room above and below the first line. */
export const ROW_HEADER =
  "grid grid-cols-[16px_minmax(0,1fr)_4rem_20px_20px] @[40rem]:grid-cols-[16px_minmax(0,1fr)_16px_4.5rem_4rem_20px_20px] items-start gap-x-3 px-3 py-2.5 cursor-pointer";

/**
 * A small icon button in a row: 24×24 to hit, but it takes only its icon's
 * room in the layout.
 */
export const ROW_ICON_BUTTON =
  "p-[5px] -mx-[5px] rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors";

/**
 * Where an expanded row shows its content: one surface below the card, so it
 * reads as nested. Rows inside inset their content to the title column.
 */
export const ROW_BODY = "mx-2 mb-2 p-1 rounded-[var(--radius-md)] bg-[var(--color-surface)]";

/**
 * The second line of an expanded row's title: the goal, or a hint, muted and
 * wrapping.
 */
export function TitleNote({ children }: { children: ReactNode }) {
  return <p className="mt-0.5 mb-0.5 text-xs leading-snug text-[var(--color-text-muted)]">{children}</p>;
}

/** The scope of a milestone by status, one line per status. Counts are leaf issues only (server). */
export function MilestoneStats({ milestone }: { milestone: Milestone }) {
  return (
    <div className="space-y-1.5">
      {STAT_ORDER.filter((s) => milestone.counts[s]).map((s) => (
        <div key={s} className="flex items-center gap-2 text-sm">
          <StatusIcon status={s} size={14} />
          <span className="flex-1 text-[var(--color-text-secondary)]">{STATUS_NAMES[s]}</span>
          <span className="text-[var(--color-text-muted)] tabular-nums">{milestone.counts[s]}</span>
        </div>
      ))}
      {milestone.total === 0 && <p className="text-sm text-[var(--color-text-muted)]">No issues yet.</p>}
    </div>
  );
}

const CHANGE_NAMES: Record<MilestoneChangeKind, string> = {
  done: "Done", reopened: "Reopened", added: "Added", removed: "Removed", canceled: "Canceled", estimated: "Estimated",
};

/** At most this many changes show in a step's hover; the rest are counted. */
const HOVER_CHANGES = 10;

const formatStepTime = (at: string) =>
  new Date(at).toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });

/** What the hover shows for one point: the step's time and changes, or the day. */
function PointDetail({ point, mode }: { point: BurnUpPoint; mode: ProgressMode }) {
  const unit = mode === "points" ? " pts" : "";
  const changes = point.step?.changes ?? [];
  return (
    <>
      <div className="flex justify-between gap-3 text-[var(--color-text-muted)] tabular-nums">
        <span>{point.step ? formatStepTime(point.step.at) : formatShortDate(point.day!.date)}</span>
        <span>Done {point.done} of {point.scope}{unit}</span>
      </div>
      {point.step && changes.length === 0 && <div className="mt-1 text-[var(--color-text-muted)]">Start</div>}
      {changes.length > 0 && (
        <ul className="mt-1 space-y-0.5">
          {changes.slice(0, HOVER_CHANGES).map((c) => (
            <li key={c.issue_id} className="flex gap-1.5 min-w-0">
              <span className="shrink-0 text-[var(--color-text-muted)]">{CHANGE_NAMES[c.kind] ?? c.kind}</span>
              <span className="truncate text-[var(--color-text-secondary)]">{c.title || c.issue_id}</span>
            </li>
          ))}
          {changes.length > HOVER_CHANGES && <li className="text-[var(--color-text-muted)]">and {changes.length - HOVER_CHANGES} more</li>}
        </ul>
      )}
    </>
  );
}

/**
 * A burn-up: done work rising toward the scope, a dashed line; the gap
 * between them is the open work. Adding scope moves only the scope line, so
 * a milestone that grows while it is worked on still reads as progress. One point
 * per step (Order) or per day (Time), evenly spaced. A single point draws as a
 * flat line across the chart. With `hover`, resting the pointer on a point
 * shows its time and changes.
 */
export function BurnUp({ points, height = 40, mode = "issues", hover = false }: {
  points: BurnUpPoint[];
  height?: number;
  mode?: ProgressMode;
  hover?: boolean;
}) {
  const [active, setActive] = useState<number | null>(null);
  if (points.length === 0) return null;
  const drawn = points.length === 1 ? [points[0], points[0]] : points;
  const w = 200;
  const h = height;
  const py = 3;
  const max = Math.max(1, ...drawn.map((d) => d.scope));
  const toX = (i: number) => (i / (drawn.length - 1)) * w;
  const toY = (v: number) => py + (h - py * 2) * (1 - v / max);
  const line = (pick: (d: BurnUpPoint) => number) => drawn.map((d, i) => `${i === 0 ? "M" : "L"}${toX(i)},${toY(pick(d))}`).join(" ");
  const done = line((d) => d.done);
  // Each point owns the band halfway to its neighbours.
  const band = points.length === 1 ? w : w / (points.length - 1);
  const pointX = (i: number) => (points.length === 1 ? w / 2 : toX(i));
  const fraction = active === null ? 0 : pointX(active) / w;
  return (
    <div className="relative" onMouseLeave={() => setActive(null)}>
      <svg role="img" aria-label="Burn-up" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" className="w-full block" style={{ height }}>
        <path d={line((d) => d.scope)} fill="none" stroke="var(--color-text-muted)" strokeWidth="1" strokeDasharray="2 2" opacity="0.5" vectorEffect="non-scaling-stroke" />
        <path d={`${done} L${w},${h} L0,${h} Z`} fill="var(--color-accent-primary)" opacity="0.12" />
        <path d={done} fill="none" stroke="var(--color-accent-primary)" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
        {active !== null && (
          <line x1={pointX(active)} x2={pointX(active)} y1={0} y2={h} stroke="var(--color-text-muted)" strokeWidth="1" opacity="0.6" vectorEffect="non-scaling-stroke" />
        )}
        {hover && points.map((_, i) => (
          <rect
            key={i}
            data-point={i}
            x={Math.max(0, pointX(i) - band / 2)}
            width={Math.min(w, pointX(i) + band / 2) - Math.max(0, pointX(i) - band / 2)}
            y={0}
            height={h}
            fill="transparent"
            onMouseEnter={() => setActive(i)}
          />
        ))}
      </svg>
      {active !== null && (
        <div
          role="tooltip"
          className="absolute bottom-full mb-1 z-10 w-64 px-2.5 py-2 text-xs rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-2)] shadow-lg pointer-events-none"
          // Left at the point, shifted by the same fraction: it stays inside the chart at both ends.
          style={{ left: `${fraction * 100}%`, transform: `translateX(-${fraction * 100}%)` }}
        >
          <PointDetail point={points[active]} mode={mode} />
        </div>
      )}
    </div>
  );
}

/** Time from the start (or creation) to the target date, filled to today. */
export function TimeBar({ milestone, today }: { milestone: Milestone; today: string }) {
  const bar = timeBar(milestone.started_at ?? milestone.created_at, milestone.target_date, today, milestone.done < milestone.total);
  if (!bar) return null;
  return (
    <div role="img" aria-label="Time to target date" className="h-1 rounded-full bg-[var(--color-hover-surface)] overflow-hidden">
      <div
        className={cn("h-full", bar.overdue ? "bg-[var(--color-warning)]" : "bg-[var(--color-text-muted)] opacity-60")}
        style={{ width: `${bar.fraction * 100}%` }}
      />
    </div>
  );
}

export function OrderWarning({ names }: { names: string[] | undefined }) {
  if (!names?.length) return null;
  const text = `Target date conflicts with ${names.join(", ")}`;
  return (
    <Tooltip content={text}>
      <span aria-label={text} className="inline-flex items-center text-[var(--color-warning)]">
        <TriangleAlert size={14} />
      </span>
    </Tooltip>
  );
}

export function ExpandToggle({ expanded, onToggle }: { expanded: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      aria-label={expanded ? "Hide issues" : "Show issues"}
      className={ROW_ICON_BUTTON}
    >
      <ChevronRight size={14} className={cn("transition-transform", expanded && "rotate-90")} />
    </button>
  );
}

/** Current is the loudest: a solid accent pill. Next is quieter: the accent tint. */
export function SlotBadge({ slot }: { slot?: string }) {
  if (slot !== "current" && slot !== "next") return null;
  return (
    <span
      className={cn(
        "shrink-0 px-2 h-5 inline-flex items-center rounded-full text-xs font-medium",
        slot === "current"
          ? "bg-[var(--color-accent-primary)] text-white"
          : "bg-[var(--color-accent-primary)]/15 text-[var(--color-accent-primary)]",
      )}
    >
      {slot === "current" ? "Current" : "Next"}
    </span>
  );
}

/**
 * The milestone as one title: its name in medium weight (the label, or `#4`
 * until it has one), then " – " and the goal in a muted tone, then the
 * Current/Next badge. Only the text truncates; the badge stays visible.
 * `name` replaces the plain name, so a row can make just the name a button;
 * `extra` (the row's progress) sits between the text and the badge.
 */
export function MilestoneTitle({ milestone, badge = true, name, extra }: { milestone: Milestone; badge?: boolean; name?: ReactNode; extra?: ReactNode }) {
  return (
    <span className="flex items-center gap-2 min-w-0">
      <span className="text-sm truncate">
        {name ?? <MilestoneName milestone={milestone} />}
        {milestone.goal && <span className="text-[var(--color-text-muted)]"> – {milestone.goal}</span>}
      </span>
      {extra}
      {badge && <SlotBadge slot={milestone.slot} />}
    </span>
  );
}

function MilestoneName({ milestone }: { milestone: Milestone }) {
  return <span className={milestoneNameClass(milestone)}>{milestone.name}</span>;
}

/**
 * The date column: the completion date of a past milestone, otherwise the target
 * date, in the warning colour once it has passed with open scope.
 */
export function MilestoneDate({ milestone, today }: { milestone: Milestone; today: string }) {
  if (milestone.completed_at) {
    return (
      <span title="Completed" className="text-sm tabular-nums text-right text-[var(--color-text-muted)]">
        {formatShortDate(milestone.completed_at)}
      </span>
    );
  }
  if (!milestone.target_date) return <span />;
  const overdue = timeBar(milestone.started_at ?? milestone.created_at, milestone.target_date, today, milestone.done < milestone.total)?.overdue;
  return (
    <span className={cn("text-sm tabular-nums text-right", overdue ? "text-[var(--color-warning)]" : "text-[var(--color-text-muted)]")}>
      {formatShortDate(milestone.target_date)}
    </span>
  );
}

const BREAKDOWN_DELAY = 250;

/**
 * Done of total: issues with the epic progress ring, or story points with
 * the estimate triangle; a click switches every row between the two. Resting
 * the pointer on it opens the full breakdown. In points mode a past milestone
 * without stored points shows nothing.
 */
export function MilestoneProgress({ milestone, today, mode = "issues", onToggleMode }: {
  milestone: Milestone;
  today: string;
  mode?: ProgressMode;
  onToggleMode?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [intent] = useState(() => createHoverIntent(BREAKDOWN_DELAY, setOpen));
  useEffect(() => () => intent.cancel(), [intent]);
  const anchorRef = useRef<HTMLButtonElement>(null);
  const points = mode === "points";
  // A milestone without issues has nothing to break down.
  const breakdown = milestone.total > 0;
  if (points && !milestone.points) return null;
  const { done, total } = points ? milestone.points! : milestone;
  const label = points
    ? `${done} of ${total} story points done. Click for issue count`
    : `${done} of ${total} issues done. Click for story points`;
  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => { intent.leave(); onToggleMode?.(); }}
        onMouseEnter={() => { if (breakdown) intent.enter(); }}
        onMouseLeave={() => intent.leave()}
        aria-label={label}
        className={raisedIndicatorPill}
      >
        {points ? <Triangle size={10} fill="currentColor" strokeWidth={0} aria-hidden /> : <SubProgress done={done} total={total} size={12} />}
        {done}/{total}
      </button>
      {open && (
        <Popover anchorRef={anchorRef} onClose={() => setOpen(false)} placement="bottom-end" className="pointer-events-none">
          <PopoverPanel className="w-72 py-0">
            <div role="tooltip"><MilestoneSummaryCard milestone={milestone} today={today} mode={mode} /></div>
          </PopoverPanel>
        </Popover>
      )}
    </>
  );
}

/**
 * The full breakdown of a milestone: status counts, target date and, for an
 * open milestone, the burn-up. A past milestone's counts are its final snapshot;
 * its burn-up is in its expanded row. The header counts what the pill counts
 * (`mode`): issues done, or story points.
 */
export function MilestoneSummaryCard({ milestone, today, mode = "issues" }: { milestone: Milestone; today: string; mode?: ProgressMode }) {
  const started = !!milestone.started_at && !milestone.placeholder;
  const progress = useMilestoneProgress(`#${milestone.number}`, started);
  const [axis] = useBurnUpAxis();
  const points = burnUpPoints(progress, axis, mode);
  const section = "px-3 py-2.5 border-t border-[var(--color-border-subtle)] first:border-t-0";
  return (
    <div className="text-sm">
      <div className={cn(section, "flex items-center gap-3")}>
        <span className="flex-1 min-w-0"><MilestoneTitle milestone={milestone} badge={false} /></span>
        <span className="text-[var(--color-text-muted)] tabular-nums shrink-0">
          {mode === "points" && milestone.points ? `${milestone.points.done}/${milestone.points.total} pts` : `${milestone.done}/${milestone.total} done`}
        </span>
      </div>
      <div className={section}><MilestoneStats milestone={milestone} /></div>
      {milestone.target_date && (
        <div className={cn(section, "space-y-1.5")}>
          <div className="flex justify-between text-[var(--color-text-muted)]">
            <span>Target date</span>
            <span className="tabular-nums">{formatShortDate(milestone.target_date)}</span>
          </div>
          <TimeBar milestone={milestone} today={today} />
        </div>
      )}
      {milestone.state === "OPEN" && (
        <div className={cn(section, "space-y-1.5")}>
          <div className="flex justify-between text-[var(--color-text-muted)]">
            <span>Burn-up</span>
            {started && <span className="tabular-nums">since {formatShortDate(milestone.started_at!)}</span>}
          </div>
          {!started ? (
            <p className="text-[var(--color-text-muted)]">Starts when this milestone becomes Current.</p>
          ) : <BurnUp points={points} height={48} mode={mode} />}
        </div>
      )}
    </div>
  );
}

const AXIS_OPTIONS: { value: BurnUpAxis; label: string }[] = [
  { value: "order", label: "Order" },
  { value: "time", label: "Time" },
];

/** The Order | Time switch of a burn-up, built from native radios so arrow keys move between them. */
function AxisToggle({ axis, onChange }: { axis: BurnUpAxis; onChange: (axis: BurnUpAxis) => void }) {
  const name = useId();
  return (
    <span role="radiogroup" aria-label="Burn-up axis" className="inline-flex p-px rounded-[var(--radius-sm)] border border-[var(--color-border-subtle)]">
      {AXIS_OPTIONS.map((opt) => {
        const checked = axis === opt.value;
        return (
          <label
            key={opt.value}
            className={cn(
              "px-1.5 rounded-[3px] cursor-pointer transition-colors",
              "has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-[var(--color-border-focus)]",
              checked ? "bg-[var(--color-surface-3)] text-[var(--color-text-primary)]" : "hover:text-[var(--color-text-primary)]",
            )}
          >
            <input type="radio" name={name} value={opt.value} checked={checked} onChange={() => onChange(opt.value)} className="sr-only" />
            {opt.label}
          </label>
        );
      })}
    </span>
  );
}

const pointDate = (p: BurnUpPoint) => (p.step ? p.step.at : p.day!.date);

/**
 * The burn-up with its legend and the Order | Time switch, between the first
 * date and, from the second point, the last. Order counts the steps it draws.
 * It counts what the progress pill counts. With fewer than `minPoints`
 * points it shows the `empty` note instead of the chart, and keeps the switch
 * so the other axis stays one click away; without a note it shows nothing.
 */
export function BurnUpWithLegend({ progress, minPoints = 1, empty }: {
  progress: Pick<MilestoneProgressResponse, "days" | "steps"> | undefined;
  minPoints?: number;
  empty?: ReactNode;
}) {
  const [axis, setAxis] = useBurnUpAxis();
  const [mode] = useProgressMode();
  const points = burnUpPoints(progress, axis, mode);
  const shown = points.length >= Math.max(1, minPoints);
  if (!shown && !empty) return null;
  return (
    <div>
      {shown && <BurnUp points={points} height={96} mode={mode} hover />}
      <div className="mt-1 flex items-center gap-3 text-xs tabular-nums text-[var(--color-text-muted)]">
        {shown && <span>{formatShortDate(pointDate(points[0]))}</span>}
        <span className="flex-1 flex items-center justify-center gap-3">
          {shown ? (
            <>
              <span className="flex items-center gap-1">
                <span className="w-3 h-0.5 bg-[var(--color-accent-primary)]" />
                Done
              </span>
              <span className="flex items-center gap-1">
                <span className="w-3 border-t border-dashed border-[var(--color-text-muted)] opacity-50" />
                Scope
              </span>
              {axis === "order" && <span>{points.length} {points.length === 1 ? "step" : "steps"}</span>}
            </>
          ) : (
            <span>{empty}</span>
          )}
          <AxisToggle axis={axis} onChange={setAxis} />
        </span>
        {shown && points.length >= 2 && <span>{formatShortDate(pointDate(points[points.length - 1]))}</span>}
      </div>
    </div>
  );
}

/**
 * A past milestone's burn-up from its start to its completion, for the
 * summary slot of its expanded row. Nothing for a milestone that was never
 * Current. It needs two points: two steps in Order mode, two days in Time
 * mode (a milestone completed within a day shows only the switch).
 */
export function PastBurnUp({ milestone }: { milestone: Milestone }) {
  const progress = useMilestoneProgress(`#${milestone.number}`, !!milestone.started_at);
  const [mode] = useProgressMode();
  if (!milestone.started_at || burnUpPoints(progress, "order", mode).length < 2) return null;
  return (
    // On the card, its left edge under the milestone name.
    <div className="pl-10 pr-3 pt-1 pb-3">
      <BurnUpWithLegend progress={progress} minPoints={2} empty="Completed within one day" />
    </div>
  );
}

interface MilestoneRowBodyProps {
  milestone: Milestone;
  today: string;
  /** The drag handle; placeholders have none. */
  handle: ReactNode;
  warning?: string[];
  expanded: boolean;
  onToggle: () => void;
  /** A click on the name: opens an open milestone in Issues; on a past milestone it toggles the issues like the chevron. */
  onOpen: () => void;
  highlight?: DropHighlight;
  /** The last column: the actions menu of an open milestone. */
  actions?: ReactNode;
  /** What the progress pill counts, and the click that switches it. */
  progressMode?: ProgressMode;
  onToggleProgressMode?: () => void;
  /** Shown on the card while expanded, above the issue list: what sums up the milestone (a past milestone's burn-up). */
  summary?: ReactNode;
  /** The milestone's issues, shown while expanded, one surface below the card. */
  children?: ReactNode;
}

/**
 * One saved milestone in the stack; a past milestone is a faded row with a
 * checkmark instead of the handle while it is collapsed. Placeholders don't
 * get a row; the top bar's "New Milestone" button creates milestones. A click
 * anywhere on the header toggles the issues; only the name opens the
 * milestone. Collapsed, the title is the name and the progress, with the goal
 * as the name's tooltip; expanded, the goal is a second line under them.
 */
export function MilestoneRowBody({
  milestone, today, handle, warning, expanded, onToggle, onOpen, highlight = null, actions, progressMode, onToggleProgressMode, summary, children,
}: MilestoneRowBodyProps) {
  const past = milestone.state === "COMPLETED";
  const goalTooltip = !expanded && !!milestone.goal;
  const name = (
    <button
      type="button"
      onClick={onOpen}
      title={past || goalTooltip ? undefined : "Show in Issues"}
      aria-expanded={past ? expanded : undefined}
      className={cn("min-w-0 truncate text-left text-sm", milestoneNameClass(milestone), !past && "hover:underline underline-offset-2")}
    >
      {milestone.name}
    </button>
  );
  return (
    <div
      className={cn(
        "rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-1)] transition-[box-shadow,opacity]",
        past && !expanded && "opacity-60 hover:opacity-100",
        dropHighlightClass(highlight),
      )}
    >
      <div
        className={ROW_HEADER}
        onClick={(e) => { if (headerClickToggles(e.target as Element, e.currentTarget)) onToggle(); }}
      >
        <span className={ROW_CELL}>{handle}</span>
        <span className="min-w-0">
          <span className={cn(ROW_CELL, "gap-2 min-w-0")}>
            {goalTooltip ? <Tooltip content={<span className="block max-w-sm">{milestone.goal}</span>}>{name}</Tooltip> : name}
            <MilestoneProgress milestone={milestone} today={today} mode={progressMode} onToggleMode={onToggleProgressMode} />
          </span>
          {expanded && milestone.goal && <TitleNote>{milestone.goal}</TitleNote>}
        </span>
        <span className={cn(ROW_CELL, NARROW_HIDDEN)}><OrderWarning names={warning} /></span>
        <span className={cn(ROW_CELL, NARROW_HIDDEN, "justify-end")}><MilestoneDate milestone={milestone} today={today} /></span>
        <span className={cn(ROW_CELL, "justify-end")}><SlotBadge slot={milestone.slot} /></span>
        <span className={ROW_CELL}>{actions}</span>
        <ExpandToggle expanded={expanded} onToggle={onToggle} />
      </div>
      {expanded && summary}
      {expanded && <div className={ROW_BODY}>{children}</div>}
    </div>
  );
}
