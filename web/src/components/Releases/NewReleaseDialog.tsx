import { useId, useState, type ReactNode } from "react";
import { useCreateRelease } from "../../api/mutations";
import { Button, DialogOverlay, DialogPanel, useToast } from "../ui";

const INPUT_CLASS =
  "w-full h-8 px-3 text-sm bg-[var(--color-surface-2)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]";

/** Creates a release at the end of the open releases. Every field is optional. */
export default function NewReleaseDialog({ onClose }: { onClose: () => void }) {
  const id = useId();
  const [label, setLabel] = useState("");
  const [goal, setGoal] = useState("");
  const [targetDate, setTargetDate] = useState("");
  const create = useCreateRelease();
  const showToast = useToast();

  const submit = () => {
    create.mutate(
      { label: label.trim() || undefined, goal: goal.trim() || undefined, target_date: targetDate || undefined },
      {
        onSuccess: (release) => {
          showToast(`Created release ${release.name}`);
          onClose();
        },
      },
    );
  };
  const onEnter = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !create.isPending) submit();
  };

  return (
    <DialogOverlay scope="release.new" onClose={onClose}>
      <DialogPanel
        id={id}
        title="New release"
        footer={<>
          <Button variant="secondary" size="sm" onClick={onClose}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={submit} loading={create.isPending}>Create release</Button>
        </>}
      >
        <div className="mt-4 grid grid-cols-[80px_1fr] items-center gap-x-3 gap-y-3">
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
      </DialogPanel>
    </DialogOverlay>
  );
}

function FieldName({ children }: { children: ReactNode }) {
  return <span className="text-xs font-medium text-[var(--color-text-secondary)]">{children}</span>;
}
