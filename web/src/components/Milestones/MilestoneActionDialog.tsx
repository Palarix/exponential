import { useState } from "react";
import type { Milestone, MilestonesResponse } from "../../api/types";
import { useCompleteMilestone, useDeleteMilestone } from "../../api/mutations";
import { Button, ConfirmDialog, InlineDropdown, useToast } from "../ui";
import { moveHelp, milestoneDialogCopy, milestoneIntro, type HelpPart, type MilestoneAction } from "./milestone-action-utils";
import { moveOptions, moveRequest, type MoveOption } from "./milestones-utils";

/** Help text with its bold runs. */
function HelpText({ parts }: { parts: HelpPart[] }) {
  return <>{parts.map((part, i) => typeof part === "string"
    ? part
    : <strong key={i} className="font-medium text-[var(--color-text-primary)]">{part.strong}</strong>)}</>;
}

/** Where the issues go: a labelled picker of milestones and No milestone. */
export function MovePicker({ label, options, value, onChange }: {
  label: string;
  options: MoveOption[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <span className="block text-xs font-medium text-[var(--color-text-secondary)]">{label}</span>
      <InlineDropdown placeholder="Pick a milestone" options={options} value={value} onChange={onChange} fullWidth />
    </div>
  );
}

/**
 * Confirms completing or deleting a saved open milestone, and asks where its
 * issues go when any would move. A success toasts the server's messages.
 */
export default function MilestoneActionDialog({ action, board, milestone, count, onClose }: {
  action: MilestoneAction;
  board: MilestonesResponse;
  milestone: Milestone;
  count: number;
  onClose: () => void;
}) {
  const [picker] = useState(() => moveOptions(board, milestone));
  const [value, setValue] = useState(picker.value);
  const complete = useCompleteMilestone();
  const remove = useDeleteMilestone();
  const mutation = action === "complete" ? complete : remove;
  const showToast = useToast();
  const copy = milestoneDialogCopy(action, count);
  const help = moveHelp(action, milestone, count, picker.options.find((o) => o.value === value));

  const confirm = () => {
    mutation.mutate(
      { id: milestone.id!, ...moveRequest(value) },
      {
        onSuccess: ({ messages }) => {
          showToast(messages.join(". "));
          onClose();
        },
      },
    );
  };

  return (
    <ConfirmDialog
      scope={`milestone.confirm-${action}`}
      title={copy.title}
      description={<HelpText parts={help} />}
      body={
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-secondary)]"><HelpText parts={milestoneIntro(action, milestone, count)} /></p>
          {copy.pickerLabel && <MovePicker label={copy.pickerLabel} options={picker.options} value={value} onChange={setValue} />}
        </div>
      }
      onCancel={onClose}
    >
      <Button variant="secondary" size="sm" onClick={onClose}>Cancel</Button>
      <Button variant={action === "delete" ? "danger" : "primary"} size="sm" onClick={confirm} loading={mutation.isPending}>
        {copy.confirm}
      </Button>
    </ConfirmDialog>
  );
}
