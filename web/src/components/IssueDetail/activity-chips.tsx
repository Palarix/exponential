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

// Chips are flex items in the system row, which centers them on the row's
// text; the chip in turn centers its icon on its label. lift adds an optical
// correction for icons whose ink is not centered in the box (see EstimateChip).
const CHIP = `inline-flex items-center ${HOVER}`;

function ChipIcon({ children, lift }: { children: React.ReactNode; lift?: string }) {
  return <span className={`mr-1 [&>svg]:block ${lift ?? ""}`}>{children}</span>;
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
    <span className={CHIP}>
      <ChipIcon>
        <StatusIcon status={status} size={12} />
      </ChipIcon>
      {STATUS_LABELS[status] || status}
    </span>
  );
}

export function EstimateChip({ points }: { points: number }) {
  return (
    <span className={CHIP}>
      {/* Lucide's triangle sits 0.25px low when centered on the label. */}
      <ChipIcon lift="relative -top-[0.25px]">
        <Triangle className="w-3 h-3" />
      </ChipIcon>
      {points} {points === 1 ? "Point" : "Points"}
    </span>
  );
}

export function PriorityChip({ priority }: { priority: number }) {
  return (
    <span className={CHIP}>
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
