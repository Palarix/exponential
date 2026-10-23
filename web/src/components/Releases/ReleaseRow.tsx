import { useEffect, useRef, useState, type ReactNode } from "react";
import { ChevronRight, Triangle, TriangleAlert } from "lucide-react";
import type { Release, ReleaseDay } from "../../api/types";
import { useReleaseProgress } from "../../api/queries";
import { formatShortDate } from "../../utils/format";
import { cn } from "../../utils/cn";
import { Popover, PopoverPanel, raisedIndicatorPill, StatusIcon, SubProgress } from "../ui";
import Tooltip from "../ui/Tooltip";
import { createHoverIntent } from "../IssueDetail/hover-intent";
import { dropHighlightClass, headerClickToggles, timeBar, type DropHighlight } from "./releases-utils";

const STAT_ORDER = ["DOING", "BLOCKED", "PLANNED", "BACKLOG", "DONE"];
const STATUS_NAMES: Record<string, string> = {
  DOING: "In progress", BLOCKED: "Blocked", PLANNED: "Planned", BACKLOG: "Backlog", DONE: "Done",
};

/**
 * The columns every release row shares, so the stack reads as a table:
 * handle, title (with the progress), warning, target date, Current/Next
 * badge, actions menu, and the expand chevron always rightmost.
 */
export const RELEASE_ROW_GRID =
  "grid grid-cols-[16px_minmax(0,1fr)_16px_4.5rem_4rem_20px_20px] items-center gap-x-3";

/** The scope of a release by status, one line per status. Counts are leaf issues only (server). */
export function ReleaseStats({ release }: { release: Release }) {
  return (
    <div className="space-y-1.5">
      {STAT_ORDER.filter((s) => release.counts[s]).map((s) => (
        <div key={s} className="flex items-center gap-2 text-sm">
          <StatusIcon status={s} size={14} />
          <span className="flex-1 text-[var(--color-text-secondary)]">{STATUS_NAMES[s]}</span>
          <span className="text-[var(--color-text-muted)] tabular-nums">{release.counts[s]}</span>
        </div>
      ))}
      {release.total === 0 && <p className="text-sm text-[var(--color-text-muted)]">No issues yet.</p>}
    </div>
  );
}

/** Open scope per day since started_at, with the scope as a dashed line. */
export function Burndown({ days, height = 40 }: { days: ReleaseDay[]; height?: number }) {
  if (days.length < 2) return null;
  const w = 200;
  const h = height;
  const py = 3;
  const max = Math.max(1, ...days.map((d) => d.scope));
  const toX = (i: number) => (i / (days.length - 1)) * w;
  const toY = (v: number) => py + (h - py * 2) * (1 - v / max);
  const line = (pick: (d: ReleaseDay) => number) => days.map((d, i) => `${i === 0 ? "M" : "L"}${toX(i)},${toY(pick(d))}`).join(" ");
  const open = line((d) => d.open);
  return (
    <svg role="img" aria-label="Burndown" viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none" className="w-full" style={{ height }}>
      <path d={line((d) => d.scope)} fill="none" stroke="var(--color-text-muted)" strokeWidth="1" strokeDasharray="2 2" opacity="0.5" vectorEffect="non-scaling-stroke" />
      <path d={`${open} L${w},${h} L0,${h} Z`} fill="var(--color-accent-primary)" opacity="0.08" />
      <path d={open} fill="none" stroke="var(--color-accent-primary)" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

/** Time from the start (or creation) to the target date, filled to today. */
export function TimeBar({ release, today }: { release: Release; today: string }) {
  const bar = timeBar(release.started_at ?? release.created_at, release.target_date, today, release.done < release.total);
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
      className="p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
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
 * The release as one title: its name in medium weight (the label, or `#4`
 * until it has one), then " – " and the goal in a muted tone, then the
 * Current/Next badge. Only the text truncates; the badge stays visible.
 * `name` replaces the plain name, so a row can make just the name a button;
 * `extra` (the row's progress) sits between the text and the badge.
 */
export function ReleaseTitle({ release, badge = true, name, extra }: { release: Release; badge?: boolean; name?: ReactNode; extra?: ReactNode }) {
  return (
    <span className="flex items-center gap-2 min-w-0">
      <span className="text-sm truncate">
        {name ?? <ReleaseName release={release} />}
        {release.goal && <span className="text-[var(--color-text-muted)]"> – {release.goal}</span>}
      </span>
      {extra}
      {badge && <SlotBadge slot={release.slot} />}
    </span>
  );
}

function releaseNameClass(release: Release): string {
  return cn("font-medium", release.label ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] tabular-nums");
}

function ReleaseName({ release }: { release: Release }) {
  return <span className={releaseNameClass(release)}>{release.name}</span>;
}

/**
 * The date column: the release date of a past release, otherwise the target
 * date, in the warning colour once it has passed with open scope.
 */
function ReleaseDate({ release, today }: { release: Release; today: string }) {
  if (release.released_at) {
    return (
      <span title="Released" className="text-sm tabular-nums text-right text-[var(--color-text-muted)]">
        {formatShortDate(release.released_at)}
      </span>
    );
  }
  if (!release.target_date) return <span />;
  const overdue = timeBar(release.started_at ?? release.created_at, release.target_date, today, release.done < release.total)?.overdue;
  return (
    <span className={cn("text-sm tabular-nums text-right", overdue ? "text-[var(--color-warning)]" : "text-[var(--color-text-muted)]")}>
      {formatShortDate(release.target_date)}
    </span>
  );
}

/** What the progress pill counts: issues, or story points. */
export type ProgressMode = "issues" | "points";

const BREAKDOWN_DELAY = 250;

/**
 * Done of total: issues with the epic progress ring, or story points with
 * the estimate triangle; a click switches every row between the two. Resting
 * the pointer on it opens the full breakdown. In points mode a past release
 * without stored points shows nothing.
 */
export function ReleaseProgress({ release, today, mode = "issues", onToggleMode }: {
  release: Release;
  today: string;
  mode?: ProgressMode;
  onToggleMode?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [intent] = useState(() => createHoverIntent(BREAKDOWN_DELAY, setOpen));
  useEffect(() => () => intent.cancel(), [intent]);
  const anchorRef = useRef<HTMLButtonElement>(null);
  const points = mode === "points";
  if (points && !release.points) return null;
  const { done, total } = points ? release.points! : release;
  const label = points
    ? `${done} of ${total} story points done. Click for issue count`
    : `${done} of ${total} issues done. Click for story points`;
  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => { intent.leave(); onToggleMode?.(); }}
        onMouseEnter={() => intent.enter()}
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
            <div role="tooltip"><ReleaseSummaryCard release={release} today={today} /></div>
          </PopoverPanel>
        </Popover>
      )}
    </>
  );
}

/**
 * The full breakdown of a release: status counts, target date and, for an
 * open release, the burndown. A past release's counts are its final snapshot;
 * its burndown is in its expanded row.
 */
export function ReleaseSummaryCard({ release, today }: { release: Release; today: string }) {
  const started = !!release.started_at && !release.placeholder;
  const days = useReleaseProgress(`#${release.number}`, started)?.days ?? [];
  const section = "px-3 py-2.5 border-t border-[var(--color-border-subtle)] first:border-t-0";
  return (
    <div className="text-sm">
      <div className={cn(section, "flex items-center gap-3")}>
        <span className="flex-1 min-w-0"><ReleaseTitle release={release} badge={false} /></span>
        <span className="text-[var(--color-text-muted)] tabular-nums shrink-0">
          {release.done}/{release.total} done{release.points && ` · ${release.points.done}/${release.points.total} pts`}
        </span>
      </div>
      <div className={section}><ReleaseStats release={release} /></div>
      {release.target_date && (
        <div className={cn(section, "space-y-1.5")}>
          <div className="flex justify-between text-[var(--color-text-muted)]">
            <span>Target date</span>
            <span className="tabular-nums">{formatShortDate(release.target_date)}</span>
          </div>
          <TimeBar release={release} today={today} />
        </div>
      )}
      {release.state === "OPEN" && (
        <div className={cn(section, "space-y-1.5")}>
          <div className="flex justify-between text-[var(--color-text-muted)]">
            <span>Burndown</span>
            {started && <span className="tabular-nums">since {formatShortDate(release.started_at!)}</span>}
          </div>
          {!started ? (
            <p className="text-[var(--color-text-muted)]">Starts when this release becomes Current.</p>
          ) : days.length >= 2 ? <Burndown days={days} height={48} /> : (
            <p className="text-[var(--color-text-muted)]">Not enough days yet.</p>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * A past release's burndown from its start to its release, with both dates
 * and a legend, for the top of its expanded row. Nothing for a release that
 * was never Current or shipped within a day.
 */
export function PastBurndown({ release }: { release: Release }) {
  const days = useReleaseProgress(`#${release.number}`, !!release.started_at)?.days ?? [];
  if (!release.started_at || days.length < 2) return null;
  return (
    <div className="border-t border-[var(--color-border-subtle)] px-3 pt-3 pb-2">
      <Burndown days={days} height={96} />
      <div className="mt-1 flex items-center gap-3 text-xs tabular-nums text-[var(--color-text-muted)]">
        <span>{formatShortDate(days[0].date)}</span>
        <span className="flex-1 flex items-center justify-center gap-3">
          <span className="flex items-center gap-1">
            <span className="w-3 h-0.5 bg-[var(--color-accent-primary)]" />
            Open
          </span>
          <span className="flex items-center gap-1">
            <span className="w-3 border-t border-dashed border-[var(--color-text-muted)] opacity-50" />
            Scope
          </span>
        </span>
        <span>{formatShortDate(days[days.length - 1].date)}</span>
      </div>
    </div>
  );
}

interface ReleaseRowBodyProps {
  release: Release;
  today: string;
  /** The drag handle; placeholders have none. */
  handle: ReactNode;
  warning?: string[];
  expanded: boolean;
  onToggle: () => void;
  /** A click on the name: opens an open release in Issues; on a past release it toggles the issues like the chevron. */
  onOpen: () => void;
  highlight?: DropHighlight;
  /** The last column: the actions menu of an open release. */
  actions?: ReactNode;
  /** What the progress pill counts, and the click that switches it. */
  progressMode?: ProgressMode;
  onToggleProgressMode?: () => void;
  /** The release's issues, shown while expanded. */
  children?: ReactNode;
}

/**
 * One saved release in the stack; a past release is a faded row with a
 * checkmark instead of the handle. Placeholders don't get a row; the top
 * bar's "New Release" button creates releases. A click anywhere on the header
 * toggles the issues; only the name opens the release.
 */
export function ReleaseRowBody({
  release, today, handle, warning, expanded, onToggle, onOpen, highlight = null, actions, progressMode, onToggleProgressMode, children,
}: ReleaseRowBodyProps) {
  const past = release.state === "RELEASED";
  return (
    <div
      className={cn(
        "rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-1)] transition-[box-shadow,opacity]",
        past && "opacity-60 hover:opacity-100",
        dropHighlightClass(highlight),
      )}
    >
      <div
        className={cn(RELEASE_ROW_GRID, "px-3 h-11 cursor-pointer")}
        onClick={(e) => { if (headerClickToggles(e.target as Element, e.currentTarget)) onToggle(); }}
      >
        <span className="flex">{handle}</span>
        <ReleaseTitle
          release={release}
          badge={false}
          name={
            <button
              type="button"
              onClick={onOpen}
              title={past ? undefined : "Show in Issues"}
              aria-expanded={past ? expanded : undefined}
              className={cn(releaseNameClass(release), !past && "hover:underline underline-offset-2")}
            >
              {release.name}
            </button>
          }
          extra={<ReleaseProgress release={release} today={today} mode={progressMode} onToggleMode={onToggleProgressMode} />}
        />
        <span className="flex"><OrderWarning names={warning} /></span>
        <ReleaseDate release={release} today={today} />
        <span className="flex justify-end"><SlotBadge slot={release.slot} /></span>
        <span className="flex">{actions}</span>
        <ExpandToggle expanded={expanded} onToggle={onToggle} />
      </div>
      {expanded && <div className="px-3 pb-3">{children}</div>}
    </div>
  );
}
