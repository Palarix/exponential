import { useId, useState, type ReactNode } from "react";
import type { Release } from "../../api/types";
import { useCreateRelease, useUpdateRelease } from "../../api/mutations";
import { Button, DialogOverlay, Dialog, useToast } from "../ui";

const INPUT_CLASS =
  "w-full h-8 px-3 text-sm bg-[var(--color-surface-2)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]";

/**
 * Creates a release at the end of the open releases, or with `release` edits
 * that one. Every field is optional; on an edit, an emptied field clears it.
 */
export default function NewReleaseDialog({ release, onClose }: { release?: Release; onClose: () => void }) {
  const id = useId();
  const [label, setLabel] = useState(release?.label ?? "");
  const [goal, setGoal] = useState(release?.goal ?? "");
  const [targetDate, setTargetDate] = useState(release?.target_date ?? "");
  const create = useCreateRelease();
  const update = useUpdateRelease();
  const pending = create.isPending || update.isPending;
  const showToast = useToast();

  const submit = () => {
    if (release) {
      update.mutate(
        { id: release.id!, label: label.trim(), goal: goal.trim(), target_date: targetDate },
        {
          onSuccess: (saved) => {
            showToast(`Saved release ${saved.name}`);
            onClose();
          },
        },
      );
      return;
    }
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
    if (e.key === "Enter" && !pending) submit();
  };

  return (
    <DialogOverlay scope={release ? "release.edit" : "release.new"} onClose={onClose}>
      <Dialog
        id={id}
        title={release ? "Edit release" : "New release"}
        footer={<>
          <Button variant="secondary" size="sm" onClick={onClose}>Cancel</Button>
          <Button variant="primary" size="sm" onClick={submit} loading={pending}>{release ? "Save" : "Create release"}</Button>
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
