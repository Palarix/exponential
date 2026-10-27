import { useId, useState, type ReactNode } from "react";
import type { Milestone } from "../../api/types";
import { useCreateMilestone, useUpdateMilestone } from "../../api/mutations";
import { Button, DialogOverlay, Dialog, useToast } from "../ui";

const INPUT_CLASS =
  "w-full h-8 px-3 text-sm bg-[var(--color-surface-2)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]";

/**
 * Creates a milestone at the end of the open milestones, or with `milestone` edits
 * that one. Every field is optional; on an edit, an emptied field clears it.
 */
export default function NewMilestoneDialog({ milestone, onClose }: { milestone?: Milestone; onClose: () => void }) {
  const id = useId();
  const [label, setLabel] = useState(milestone?.label ?? "");
  const [goal, setGoal] = useState(milestone?.goal ?? "");
  const [targetDate, setTargetDate] = useState(milestone?.target_date ?? "");
  const create = useCreateMilestone();
  const update = useUpdateMilestone();
  const pending = create.isPending || update.isPending;
  const showToast = useToast();

  const submit = () => {
    if (milestone) {
      update.mutate(
        { id: milestone.id!, label: label.trim(), goal: goal.trim(), target_date: targetDate },
        {
          onSuccess: (saved) => {
            showToast(`Saved milestone ${saved.name}`);
            onClose();
          },
        },
      );
      return;
    }
    create.mutate(
      { label: label.trim() || undefined, goal: goal.trim() || undefined, target_date: targetDate || undefined },
      {
        onSuccess: (milestone) => {
          showToast(`Created milestone ${milestone.name}`);
          onClose();
        },
      },
    );
  };
  const onEnter = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !pending) submit();
  };

  return (
    <DialogOverlay scope={milestone ? "milestone.edit" : "milestone.new"} onClose={onClose}>
      <Dialog
        id={id}
        title={milestone ? "Edit milestone" : "New milestone"}
        footer={<>
          <Button variant="secondary" size="sm" onClick={onClose}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={submit} loading={pending}>{milestone ? "Save" : "Create milestone"}</Button>
        </>}
      >
        <div className="grid grid-cols-[80px_1fr] items-center gap-x-3 gap-y-3">
          <FieldName>Label</FieldName>
          <input
            value={label}
            autoFocus
            onChange={(e) => setLabel(e.target.value)}
            onKeyDown={onEnter}
            placeholder="1.6.0 or Team Mode (optional)"
            aria-label="Label"
            className={INPUT_CLASS}
          />
          <FieldName>Goal</FieldName>
          <input
            value={goal}
            onChange={(e) => setGoal(e.target.value)}
            onKeyDown={onEnter}
            placeholder="One line (optional)"
            aria-label="Goal"
            className={INPUT_CLASS}
          />
          <FieldName>Target date</FieldName>
          <input
            type="date"
            value={targetDate}
            onChange={(e) => setTargetDate(e.target.value)}
            onKeyDown={onEnter}
            aria-label="Target date"
            className={INPUT_CLASS}
          />
        </div>
      </Dialog>
    </DialogOverlay>
  );
}

function FieldName({ children }: { children: ReactNode }) {
  return <span className="text-xs font-medium text-[var(--color-text-secondary)]">{children}</span>;
}
