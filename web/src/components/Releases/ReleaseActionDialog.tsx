import { useState } from "react";
import type { Release, ReleasesResponse } from "../../api/types";
import { useCompleteRelease, useDeleteRelease } from "../../api/mutations";
import { Button, ConfirmDialog, InlineDropdown, useToast } from "../ui";
import { moveHelp, releaseDialogCopy, releaseIntro, type HelpPart, type ReleaseAction } from "./release-action-utils";
import { moveOptions, moveRequest, type MoveOption } from "./releases-utils";

/** Help text with its bold runs. */
function HelpText({ parts }: { parts: HelpPart[] }) {
  return <>{parts.map((part, i) => typeof part === "string"
    ? part
    : <strong key={i} className="font-medium text-[var(--color-text-primary)]">{part.strong}</strong>)}</>;
}

/** Where the issues go: a labelled picker of releases and No release. */
export function MovePicker({ label, options, value, onChange }: {
  label: string;
  options: MoveOption[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      <span className="block text-xs font-medium text-[var(--color-text-secondary)]">{label}</span>
      <InlineDropdown placeholder="Pick a release" options={options} value={value} onChange={onChange} fullWidth />
    </div>
  );
}

/**
 * Confirms completing or deleting a saved open release, and asks where its
 * issues go when any would move. A success toasts the server's messages.
 */
export default function ReleaseActionDialog({ action, board, release, count, onClose }: {
  action: ReleaseAction;
  board: ReleasesResponse;
  release: Release;
  count: number;
  onClose: () => void;
}) {
  const [picker] = useState(() => moveOptions(board, release));
  const [value, setValue] = useState(picker.value);
  const complete = useCompleteRelease();
  const remove = useDeleteRelease();
  const mutation = action === "complete" ? complete : remove;
  const showToast = useToast();
  const copy = releaseDialogCopy(action, count);
  const help = moveHelp(action, release, count, picker.options.find((o) => o.value === value));

  const confirm = () => {
    mutation.mutate(
      { id: release.id!, ...moveRequest(value) },
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
      scope={`release.confirm-${action}`}
      title={copy.title}
      description={<HelpText parts={help} />}
      body={
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-secondary)]"><HelpText parts={releaseIntro(action, release, count)} /></p>
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
