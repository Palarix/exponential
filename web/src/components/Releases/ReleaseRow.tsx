import { useRef, useState, type ReactNode } from "react";
import { ChevronRight, TriangleAlert } from "lucide-react";
import type { Release, ReleaseDay } from "../../api/types";
import { useReleaseProgress } from "../../api/queries";
import { formatShortDate } from "../../utils/format";
import { cn } from "../../utils/cn";
import { indicatorPill, Popover, PopoverPanel, StatusIcon, SubProgress } from "../ui";
import Tooltip from "../ui/Tooltip";
import { dropHighlightClass, timeBar, type DropHighlight } from "./releases-utils";

const STAT_ORDER = ["DOING", "BLOCKED", "PLANNED", "BACKLOG", "DONE"];
const STATUS_NAMES: Record<string, string> = {
  DOING: "In progress", BLOCKED: "Blocked", PLANNED: "Planned", BACKLOG: "Backlog", DONE: "Done",
};

/**
 * The columns every release row shares, so the stack reads as a table:
 * handle, title, warning, target date, progress, expand.
 */
export const RELEASE_ROW_GRID =
  "grid grid-cols-[16px_minmax(0,1fr)_16px_4.5rem_5rem_20px] items-center gap-x-3";

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
 */
export function ReleaseTitle({ release, badge = true }: { release: Release; badge?: boolean }) {
  return (
    <span className="flex items-center gap-2 min-w-0">
      <span className="text-sm truncate">
        <span className={cn("font-medium", release.label ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] tabular-nums")}>
          {release.name}
        </span>
        {release.goal && <span className="text-[var(--color-text-muted)]"> – {release.goal}</span>}
      </span>
      {badge && <SlotBadge slot={release.slot} />}
    </span>
  );
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

/** Done of total as the epic progress ring; a click opens the full breakdown. */
export function ReleaseProgress({ release, today }: { release: Release; today: string }) {
  const [open, setOpen] = useState(false);
  const anchorRef = useRef<HTMLButtonElement>(null);
  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen(!open)}
        aria-label={`${release.done} of ${release.total} done. Show breakdown`}
        aria-expanded={open}
        className={cn(indicatorPill, "justify-self-end hover:border-[var(--color-border-default)] hover:text-[var(--color-text-primary)] transition-colors")}
      >
        <SubProgress done={release.done} total={release.total} size={12} />
        {release.done}/{release.total}
      </button>
      {open && (
        <Popover anchorRef={anchorRef} onClose={() => setOpen(false)} placement="bottom-end">
          <PopoverPanel className="w-72 py-0">
            <ReleaseSummaryCard release={release} today={today} />
          </PopoverPanel>
        </Popover>
      )}
    </>
  );
}

/** The full breakdown of a release: status counts, target date and burndown. */
export function ReleaseSummaryCard({ release, today }: { release: Release; today: string }) {
  const started = !!release.started_at && !release.placeholder;
  const days = useReleaseProgress(`#${release.number}`, started)?.days ?? [];
  const section = "px-3 py-2.5 border-t border-[var(--color-border-subtle)] first:border-t-0";
  return (
    <div className="text-sm">
      <div className={cn(section, "flex items-center gap-3")}>
        <span className="flex-1 min-w-0"><ReleaseTitle release={release} badge={false} /></span>
        <span className="text-[var(--color-text-muted)] tabular-nums shrink-0">{release.done}/{release.total} done</span>
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

interface ReleaseRowBodyProps {
  release: Release;
  today: string;
  /** The drag handle; placeholders have none. */
  handle: ReactNode;
  warning?: string[];
  expanded: boolean;
  onToggle: () => void;
  onOpen: () => void;
  highlight?: DropHighlight;
  /** The release's issues, shown while expanded. */
  children?: ReactNode;
}

/**
 * One saved release in the stack; a past release is a faded row with a
 * checkmark instead of the handle. Placeholders don't get a row; the top
 * bar's "New Release" button creates releases.
 */
export function ReleaseRowBody({
  release, today, handle, warning, expanded, onToggle, onOpen, highlight = null, children,
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
      <div className={cn(RELEASE_ROW_GRID, "px-3 h-11")}>
        <span className="flex">{handle}</span>
        <button type="button" onClick={onOpen} className="min-w-0 text-left hover:[&_span.truncate]:underline underline-offset-2">
          <ReleaseTitle release={release} />
        </button>
        <span className="flex"><OrderWarning names={warning} /></span>
        <ReleaseDate release={release} today={today} />
        <ReleaseProgress release={release} today={today} />
        <ExpandToggle expanded={expanded} onToggle={onToggle} />
      </div>
      {expanded && <div className="px-3 pb-3">{children}</div>}
    </div>
  );
}
