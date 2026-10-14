import { PriorityIcon, StatusIcon } from "../ui";
import { Triangle } from "lucide-react";
import { useCycles } from "../../api/queries";
import { PRIORITY_LABELS } from "../../utils/sort";

// System rows stay muted; an actor or value turns primary on hover, with a
// pointer cursor, to hint that it is interactive.
const HOVER = "cursor-pointer transition-colors hover:text-[var(--color-text-primary)]";

export function ActivityValue({ children }: { children: React.ReactNode }) {
  return <span className={HOVER}>{children}</span>;
}

// Chips are inline text, not flex boxes, so the label sits on the row's
// baseline. Only the icon is centered on the text: an inline-flex chip with
// align-middle centers the whole chip and drops its label below the line.
// lift adds an optical correction for icons whose ink is not centered in the
// box (see EstimateChip).
function ChipIcon({ children, lift }: { children: React.ReactNode; lift?: string }) {
  return <span className={`mr-1 inline-block align-middle [&>svg]:block ${lift ?? ""}`}>{children}</span>;
}

const STATUS_LABELS: Record<string, string> = {
  BACKLOG: "Backlog",
  PLANNED: "Planned",
  DOING: "In Progress",
  BLOCKED: "Blocked",
  DONE: "Done",
};

export function StatusChip({ status }: { status: string }) {
  return (
    <span className={HOVER}>
      <ChipIcon>
        <StatusIcon status={status} size={12} />
      </ChipIcon>
      {STATUS_LABELS[status] || status}
    </span>
  );
}

export function EstimateChip({ points }: { points: number }) {
  return (
    <span className={HOVER}>
      {/* Lucide's triangle base hangs 1.5px below the baseline when centered. */}
      <ChipIcon lift="relative -top-[1.5px]">
        <Triangle className="w-3 h-3" />
      </ChipIcon>
      {points} {points === 1 ? "Point" : "Points"}
    </span>
  );
}

export function PriorityChip({ priority }: { priority: number }) {
  return (
    <span className={HOVER}>
      <ChipIcon>
        <PriorityIcon priority={priority} size={12} />
      </ChipIcon>
      {PRIORITY_LABELS[priority] ?? priority}
    </span>
  );
}

// Shows "Cycle N" for a known cycle. Falls back to the raw cycle ID (its start
// date) while cycles load, when cycles are disabled, or for an unknown cycle.
export function CycleName({ id }: { id: string }) {
  const { cycles } = useCycles();
  const cycle = cycles.find((c) => c.id === id);
  return <ActivityValue>{cycle ? `Cycle ${cycle.number}` : id}</ActivityValue>;
}
