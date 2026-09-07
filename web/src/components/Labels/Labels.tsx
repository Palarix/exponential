import { useState, useMemo, useCallback, useContext, useRef, useEffect } from "react";
import type { Issue } from "../../api/client";
import { addConfigLabel, updateConfigLabel, deleteConfigLabel } from "../../api/client";
import { LabelBadge } from "../ui/Badge";
import { LabelColorsContext } from "../ui/BadgeContexts";
import { Trash2 } from "lucide-react";
import { LABEL_PRESET_COLORS } from "../../constants";
import { labelColor, canonicalLabel } from "../../utils/labels";

interface LabelsProps {
  issues: Issue[];
  onConfigLabelsChange: (labels: Record<string, string>) => void;
  onRefresh: () => void;
}

interface LabelInfo {
  name: string;
  color: string;
  count: number;
}

export default function Labels({ issues, onConfigLabelsChange, onRefresh }: LabelsProps) {
  const configLabels = useContext(LabelColorsContext);
  const [editing, setEditing] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editColor, setEditColor] = useState("");
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState(LABEL_PRESET_COLORS[0]);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const editInputRef = useRef<HTMLInputElement>(null);
  const createInputRef = useRef<HTMLInputElement>(null);

  const labelCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const issue of issues) {
      for (const label of issue.labels || []) {
        counts[label] = (counts[label] || 0) + 1;
      }
    }
    return counts;
  }, [issues]);

  const labels: LabelInfo[] = useMemo(() => {
    const seen = new Map<string, string>();
    for (const name of Object.keys(configLabels)) seen.set(name.toLowerCase(), name);
    for (const name of Object.keys(labelCounts)) {
      if (!seen.has(name.toLowerCase())) seen.set(name.toLowerCase(), canonicalLabel(name, configLabels));
    }

    return Array.from(seen.values())
      .sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }))
      .map((name) => {
        const lk = name.toLowerCase();
        let count = 0;
        for (const [k, v] of Object.entries(labelCounts)) {
          if (k.toLowerCase() === lk) count += v;
        }
        return { name, color: labelColor(name, configLabels), count };
      });
  }, [configLabels, labelCounts]);

  useEffect(() => {
    if (editing) {
      requestAnimationFrame(() => editInputRef.current?.focus());
    }
  }, [editing]);

  useEffect(() => {
    if (creating) {
      requestAnimationFrame(() => createInputRef.current?.focus());
    }
  }, [creating]);

  const startEditing = useCallback((label: LabelInfo) => {
    setEditing(label.name);
    setEditName(label.name);
    setEditColor(label.color);
    setConfirmDelete(null);
  }, []);

  const cancelEditing = useCallback(() => {
    setEditing(null);
    setEditName("");
    setEditColor("");
  }, []);

  const saveEdit = useCallback(async () => {
    if (!editing || !editName.trim()) return;
    await updateConfigLabel(editing, editName.trim(), editColor);
    const updated = { ...configLabels };
    if (editing !== editName.trim()) delete updated[editing];
    updated[editName.trim()] = editColor;
    onConfigLabelsChange(updated);
    onRefresh();
    setEditing(null);
  }, [editing, editName, editColor, configLabels, onConfigLabelsChange, onRefresh]);

  const handleDelete = useCallback(async (name: string) => {
    await deleteConfigLabel(name);
    const updated = { ...configLabels };
    delete updated[name];
    onConfigLabelsChange(updated);
    onRefresh();
    setConfirmDelete(null);
  }, [configLabels, onConfigLabelsChange, onRefresh]);

  const handleCreate = useCallback(async () => {
    if (!newName.trim()) return;
    await addConfigLabel(newName.trim(), newColor);
    onConfigLabelsChange({ ...configLabels, [newName.trim()]: newColor });
    setNewName("");
    setNewColor(LABEL_PRESET_COLORS[0]);
    setCreating(false);
  }, [newName, newColor, configLabels, onConfigLabelsChange]);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-6 h-13 shrink-0 border-b border-[var(--color-border-subtle)]">
        <h1 className="text-sm font-medium text-[var(--color-text-primary)]">Labels</h1>
        <button
          onClick={() => { setCreating(true); setEditing(null); setConfirmDelete(null); }}
          className="flex items-center gap-2 h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] transition-colors"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          New Label
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-7xl mx-auto py-4">
          {/* Create form */}
          {creating && (
            <div className="mx-4 mb-4 p-3 rounded-[var(--radius-md)] border border-[var(--color-border-default)] bg-[var(--color-surface-1)]">
              <div className="flex items-center gap-3 mb-3">
                <input
                  ref={createInputRef}
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate();
                    if (e.key === "Escape") setCreating(false);
                  }}
                  placeholder="Label name"
                  className="flex-1 h-8 px-3 text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)]"
                />
              </div>
              <div className="flex items-center gap-2 mb-3">
                <span className="text-xs text-[var(--color-text-muted)]">Color</span>
                {LABEL_PRESET_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setNewColor(c)}
                    className={`w-6 h-6 rounded-full transition-transform ${newColor === c ? "ring-2 ring-[var(--color-text-primary)] ring-offset-2 ring-offset-[var(--color-surface-1)] scale-110" : "hover:scale-110"}`}
                    style={{ background: c }}
                  />
                ))}
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-[var(--color-text-muted)] mr-auto">
                  {newName.trim() && <LabelBadge label={newName.trim()} />}
                </span>
                <button
                  onClick={() => setCreating(false)}
                  className="h-7 px-3 text-xs rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreate}
                  disabled={!newName.trim()}
                  className="h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] disabled:opacity-40 transition-colors"
                >
                  Create
                </button>
              </div>
            </div>
          )}

          {/* Label list */}
          {labels.length === 0 && !creating ? (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <svg className="w-10 h-10 text-[var(--color-text-muted)] opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
              </svg>
              <p className="text-sm text-[var(--color-text-muted)]">No labels yet</p>
              <button
                onClick={() => setCreating(true)}
                className="text-xs text-[var(--color-accent-primary)] hover:underline"
              >
                Create your first label
              </button>
            </div>
          ) : (
            <div className="divide-y divide-[var(--color-border-subtle)]">
              {labels.map((label) => (
                <div key={label.name}>
                  {editing === label.name ? (
                    <div className="px-4 py-3 bg-[var(--color-surface-1)]">
                      <div className="flex items-center gap-3 mb-3">
                        <input
                          ref={editInputRef}
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") saveEdit();
                            if (e.key === "Escape") cancelEditing();
                          }}
                          className="flex-1 h-8 px-3 text-sm bg-[var(--color-surface)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] outline-none focus:border-[var(--color-border-focus)]"
                        />
                      </div>
                      <div className="flex items-center gap-2 mb-3">
                        <span className="text-xs text-[var(--color-text-muted)]">Color</span>
                        {LABEL_PRESET_COLORS.map((c) => (
                          <button
                            key={c}
                            onClick={() => setEditColor(c)}
                            className={`w-6 h-6 rounded-full transition-transform ${editColor === c ? "ring-2 ring-[var(--color-text-primary)] ring-offset-2 ring-offset-[var(--color-surface-1)] scale-110" : "hover:scale-110"}`}
                            style={{ background: c }}
                          />
                        ))}
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-[var(--color-text-muted)] mr-auto">
                          {editName.trim() && (
                            <span className="inline-flex items-center gap-2 text-xs font-medium text-text-secondary border rounded-2xl h-6 px-2 border-bg-elevated">
                              <span className="w-2 h-2 rounded-full shrink-0" style={{ background: editColor }} />
                              {editName.trim().charAt(0).toUpperCase() + editName.trim().slice(1)}
                            </span>
                          )}
                        </span>
                        <button
                          onClick={cancelEditing}
                          className="h-7 px-3 text-xs rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                        >
                          Cancel
                        </button>
                        <button
                          onClick={saveEdit}
                          disabled={!editName.trim()}
                          className="h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] disabled:opacity-40 transition-colors"
                        >
                          Save
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex items-center gap-3 px-4 py-3 group hover:bg-[var(--color-hover-surface)] transition-colors">
                      <span className="w-3 h-3 rounded-full shrink-0" style={{ background: label.color }} />
                      <span className="text-sm text-[var(--color-text-primary)] flex-1 min-w-0">
                        {label.name.charAt(0).toUpperCase() + label.name.slice(1)}
                      </span>
                      <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
                        {label.count} {label.count === 1 ? "issue" : "issues"}
                      </span>
                      {confirmDelete === label.name ? (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-[var(--color-text-muted)]">Delete?</span>
                          <button
                            onClick={() => handleDelete(label.name)}
                            className="h-6 px-2 text-xs font-medium rounded-[var(--radius-sm)] bg-[var(--color-error)] text-white hover:opacity-90 transition-opacity"
                          >
                            Yes
                          </button>
                          <button
                            onClick={() => setConfirmDelete(null)}
                            className="h-6 px-2 text-xs rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
                          >
                            No
                          </button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            onClick={() => startEditing(label)}
                            className="h-6 w-6 flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-1)] transition-colors"
                            title="Edit label"
                          >
                            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                              <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L6.832 19.82a4.5 4.5 0 01-1.897 1.13l-2.685.8.8-2.685a4.5 4.5 0 011.13-1.897L16.863 4.487z" />
                            </svg>
                          </button>
                          <button
                            onClick={() => setConfirmDelete(label.name)}
                            className="h-6 w-6 flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-error)] hover:bg-[var(--color-surface-1)] transition-colors"
                            title="Delete label"
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
