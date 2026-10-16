import { useId, useState, type ReactNode } from "react";
import type { Issue } from "../../api/client";
import { Button, ConfirmDialog, DialogOverlay, DialogPanel, LabelBadge, StatusIcon } from "../ui";
import { LabelColorsContext } from "../ui/BadgeContexts";
import { LABEL_PRESET_COLORS } from "../../constants";
import { cn } from "../../utils/cn";

const HEX_COLOR_RE = /^#?[0-9a-f]{6}$/i;

export interface EditableLabel {
  name: string;
  color: string;
  count: number;
}

type SampleIssue = Pick<Issue, "id" | "title" | "status">;

interface EditLabelDialogProps {
  label: EditableLabel;
  /** Up to two issues to preview the label on. */
  sampleIssues: SampleIssue[];
  onSave: (name: string, color: string) => void;
  onDelete: () => void;
  onClose: () => void;
}

/** Edits a custom label's name and color, and deletes it after a confirmation. */
export default function EditLabelDialog({ label, sampleIssues, onSave, onDelete, onClose }: EditLabelDialogProps) {
  const id = useId();
  const [name, setName] = useState(label.name);
  // An uncolored label has the muted CSS var as its color, which the server rejects.
  const [color, setColor] = useState(HEX_COLOR_RE.test(label.color) ? label.color : "");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const one = label.count === 1;

  return (
    <>
      <DialogOverlay scope="label.edit" onClose={onClose} enabled={!confirmDelete}>
        <EditLabelPanel
          id={id}
          name={name}
          color={color}
          sampleIssues={sampleIssues}
          onNameChange={setName}
          onColorChange={setColor}
          onSave={() => onSave(name.trim(), color)}
          onCancel={onClose}
          onDelete={() => setConfirmDelete(true)}
        />
      </DialogOverlay>
      {confirmDelete && (
        <ConfirmDialog
          scope="label.confirm-delete"
          title="Delete label?"
          description={label.count > 0
            ? `It will be removed from ${label.count} ${one ? "issue" : "issues"}. This can't be undone.`
            : "This can't be undone."}
          body={<LabelBadge label={label.name} />}
          onCancel={() => setConfirmDelete(false)}
        >
          <Button variant="secondary" size="sm" onClick={() => setConfirmDelete(false)}>Cancel</Button>
          <Button variant="danger" size="sm" onClick={onDelete}>Delete</Button>
        </ConfirmDialog>
      )}
    </>
  );
}

interface EditLabelPanelProps {
  id: string;
  name: string;
  color: string;
  sampleIssues: SampleIssue[];
  onNameChange: (name: string) => void;
  onColorChange: (color: string) => void;
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
}

/** The dialog without its overlay: the name and color, then a preview of the label on issue rows. */
export function EditLabelPanel({
  id, name, color, sampleIssues, onNameChange, onColorChange, onSave, onCancel, onDelete,
}: EditLabelPanelProps) {
  const canSave = name.trim().length > 0 && color !== "";
  return (
    <DialogPanel
      id={id}
      title="Edit label"
      footerLeft={<Button variant="ghost" tone="danger" size="sm" className="-ml-3" onClick={onDelete}>Delete label</Button>}
      footer={<>
        <Button variant="secondary" size="sm" onClick={onCancel}>Cancel</Button>
        <Button variant="primary" size="sm" onClick={onSave} disabled={!canSave}>Save</Button>
      </>}
    >
      <div className="mt-4 grid grid-cols-[56px_1fr] items-center gap-x-3 gap-y-3">
        <FieldName>Name</FieldName>
        <input
          value={name}
          autoFocus
          onChange={(e) => onNameChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && canSave) onSave();
          }}
          placeholder="Label name"
          aria-label="Label name"
          className="w-full h-8 px-3 text-sm bg-[var(--color-surface-2)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)]"
        />
        <FieldName>Color</FieldName>
        <div className="flex items-center justify-between" role="group" aria-label="Color">
          {LABEL_PRESET_COLORS.map((c) => (
            <button
              key={c}
              type="button"
              aria-label={`Color ${c}`}
              aria-pressed={color === c}
              onClick={() => onColorChange(c)}
              className={cn(
                "w-5 h-5 rounded-full transition-transform",
                color === c
                  ? "ring-2 ring-[var(--color-text-primary)] ring-offset-2 ring-offset-[var(--color-surface-3)]"
                  : "hover:scale-110",
              )}
              style={{ background: c }}
            />
          ))}
        </div>
      </div>
      <LabelPreview name={name} color={color} issues={sampleIssues} />
    </DialogPanel>
  );
}

function FieldName({ className, children }: { className?: string; children: ReactNode }) {
  return <span className={cn("text-xs font-medium text-[var(--color-text-secondary)]", className)}>{children}</span>;
}

/** The label as it will look on issue rows, with the unsaved name and color. */
function LabelPreview({ name, color, issues }: { name: string; color: string; issues: SampleIssue[] }) {
  const label = name.trim() || "Label";
  return (
    <LabelColorsContext.Provider value={{ [label]: color || "var(--color-text-muted)" }}>
      <div className="mt-5 p-3 pt-2.5 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] shadow-[inset_0_1px_3px_rgba(0,0,0,0.35)]">
        <div className="mb-2 text-[10px] font-medium uppercase tracking-wider text-[var(--color-text-muted)]">Preview</div>
        <div className="rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] overflow-hidden divide-y divide-[var(--color-border-subtle)]">
          {issues.length === 0 ? (
            <div className="flex items-center gap-2.5 px-3 h-10">
              <StatusIcon status="PLANNED" size={14} />
              <span className="text-sm text-[var(--color-text-muted)] truncate flex-1">Issues with this label look like this</span>
              <LabelBadge label={label} />
            </div>
          ) : issues.map((issue, i) => (
            <div key={issue.id} className={cn("flex items-center gap-2.5 px-3 h-10", i > 0 && "opacity-50")}>
              <StatusIcon status={issue.status} size={14} />
              <span className="font-mono text-xs text-[var(--color-text-muted)] shrink-0">{issue.id}</span>
              <span className="text-sm text-[var(--color-text-primary)] truncate flex-1">{issue.title}</span>
              <LabelBadge label={label} />
            </div>
          ))}
        </div>
      </div>
    </LabelColorsContext.Provider>
  );
}
