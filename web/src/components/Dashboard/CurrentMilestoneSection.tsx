import { Milestone as MilestoneIcon } from "lucide-react";
import type { Milestone } from "../../api/types";
import { useMilestoneProgress, useMilestones } from "../../api/queries";
import { useAppNav } from "../../app/hooks";
import { cn } from "../../utils/cn";
import { BurndownWithLegend, MilestoneDate, MilestoneTitle, TimeBar } from "../Milestones/MilestoneRow";
import { milestoneFilter, milestoneNameClass, showsOnDashboard, todayISO } from "../Milestones/milestones-utils";
import { Card, SubProgress } from "../ui";
import { Section } from "./Section";

/** The Current milestone: name and goal, progress, target date and burndown. */
function CurrentMilestoneCard({ milestone, onOpen }: { milestone: Milestone; onOpen: () => void }) {
  const started = !!milestone.started_at;
  const days = useMilestoneProgress(`#${milestone.number}`, started)?.days ?? [];
  const today = todayISO();
  return (
    <Card variant="elevated" className="px-4 py-3 space-y-3">
      <div className="flex items-center gap-3">
        <span className="flex-1 min-w-0">
          <MilestoneTitle
            milestone={milestone}
            badge={false}
            name={
              <button
                type="button"
                onClick={onOpen}
                title="Show in Issues"
                className={cn(milestoneNameClass(milestone), "hover:underline underline-offset-2")}
              >
                {milestone.name}
              </button>
            }
          />
        </span>
        <span className="shrink-0 flex items-center gap-1.5 text-sm tabular-nums text-[var(--color-text-muted)]">
          <SubProgress done={milestone.done} total={milestone.total} size={12} />
          {milestone.done}/{milestone.total} done{milestone.points && ` · ${milestone.points.done}/${milestone.points.total} pts`}
        </span>
      </div>
      {milestone.target_date && (
        <div className="space-y-1.5">
          <div className="flex justify-between text-sm text-[var(--color-text-muted)]">
            <span>Target date</span>
            <MilestoneDate milestone={milestone} today={today} />
          </div>
          <TimeBar milestone={milestone} today={today} />
        </div>
      )}
      {started && <BurndownWithLegend days={days} />}
    </Card>
  );
}

/** The Dashboard's section for the Current milestone. Hidden until Current is saved and has scope. */
export default function CurrentMilestoneSection() {
  const { data: board } = useMilestones();
  const { navigate } = useAppNav();
  const current = board?.current;
  if (!current || !showsOnDashboard(current)) return null;
  const onOpen = () => {
    const filter = milestoneFilter(current);
    if (filter) navigate("backlog", { tab: "all", filters: { milestones: [filter] } });
  };
  return (
    <Section title="Current Milestone" icon={<MilestoneIcon size={16} />} collapsible storageKey="exponential-dashboard-milestone-open">
      <div className="px-5 py-3">
        <CurrentMilestoneCard milestone={current} onOpen={onOpen} />
      </div>
    </Section>
  );
}
