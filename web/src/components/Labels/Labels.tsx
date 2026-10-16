import { useState, useMemo, useCallback, useContext, useRef, useEffect, type ReactNode } from "react";
import { LabelBadge } from "../ui/Badge";
import { LabelColorsContext, DefaultLabelsContext } from "../ui/BadgeContexts";
import { TopBar, Heading, Text, SearchInput, ViewContainer } from "../ui";
import { LABEL_PRESET_COLORS } from "../../constants";
import { labelColor, canonicalLabel, filterLabelsByName, partitionBuiltinLabels, recentIssuesWithLabel } from "../../utils/labels";
import { useIssueList } from "../../api/queries";
import { useAddLabel, useDeleteLabel, useUpdateLabel } from "../../api/mutations";
import { useAppNav } from "../../app/hooks";
import EditLabelDialog, { type EditableLabel } from "./EditLabelDialog";

type LabelInfo = EditableLabel;

function issueCountText(count: number): string {
  return `${count} ${count === 1 ? "issue" : "issues"}`;
}

export default function Labels() {
  const issues = useIssueList();
  const { mutate: addLabel } = useAddLabel();
  const { mutate: updateLabel } = useUpdateLabel();
  const { mutate: deleteLabel } = useDeleteLabel();
  const { navigate } = useAppNav();
  const onLabelClick = (label: string) => navigate("backlog", { filters: { labels: [label] } });
  const configLabels = useContext(LabelColorsContext);
  const defaultLabels = useContext(DefaultLabelsContext);
  const [editing, setEditing] = useState<LabelInfo | null>(null);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [newColor, setNewColor] = useState(LABEL_PRESET_COLORS[0]);
  const [search, setSearch] = useState("");
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

  const { builtin: builtinLabels, custom: customLabels } = useMemo(
    () => partitionBuiltinLabels(labels, defaultLabels),
    [labels, defaultLabels],
  );
  const visibleBuiltin = useMemo(() => filterLabelsByName(builtinLabels, search), [builtinLabels, search]);
  const visibleCustom = useMemo(() => filterLabelsByName(customLabels, search), [customLabels, search]);

  useEffect(() => {
    if (creating) {
      requestAnimationFrame(() => createInputRef.current?.focus());
    }
  }, [creating]);

  const saveEdit = useCallback((name: string, color: string) => {
    if (!editing) return;
    updateLabel(
      { oldName: editing.name, newName: name, color },
      { onSuccess: () => setEditing(null) },
    );
  }, [editing, updateLabel]);

  const handleDelete = useCallback(() => {
    if (!editing) return;
    deleteLabel({ name: editing.name }, { onSuccess: () => setEditing(null) });
  }, [editing, deleteLabel]);

  const handleCreate = useCallback(() => {
    if (!newName.trim()) return;
    addLabel({ name: newName.trim(), color: newColor }, {
      onSuccess: () => {
        setNewName("");
        setNewColor(LABEL_PRESET_COLORS[0]);
        setCreating(false);
      },
    });
  }, [newName, newColor, addLabel]);

  const countLink = (label: LabelInfo) => label.count > 0 ? (
    <button
      onClick={(e) => { e.stopPropagation(); onLabelClick(label.name); }}
      onKeyDown={(e) => e.stopPropagation()}
      className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:underline transition-colors"
      title={`Show issues labeled ${label.name}`}
    >
      {issueCountText(label.count)}
    </button>
  ) : (
    <span className="text-sm text-[var(--color-text-muted)]">{issueCountText(0)}</span>
  );

  const rowContent = (label: LabelInfo) => (
    <>
      <span className="w-3 h-3 rounded-full shrink-0" style={{ background: label.color }} />
      <span className="text-sm text-[var(--color-text-primary)] flex-1 min-w-0">
        {label.name.charAt(0).toUpperCase() + label.name.slice(1)}
      </span>
      {countLink(label)}
    </>
  );

  const renderBuiltinRow = (label: LabelInfo) => (
    <div key={label.name} className="flex items-center gap-3 px-4 py-3">
      {rowContent(label)}
    </div>
  );

  const renderCustomRow = (label: LabelInfo) => (
    <div
      key={label.name}
      role="button"
      tabIndex={0}
      onClick={() => setEditing(label)}
      onKeyDown={(e) => {
        if (e.target !== e.currentTarget) return;
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          setEditing(label);
        }
      }}
      className="flex items-center gap-3 px-4 py-3 hover:bg-[var(--color-hover-surface)] transition-colors cursor-pointer outline-none focus-visible:bg-[var(--color-hover-surface)]"
    >
      {rowContent(label)}
    </div>
  );

  return (
    <ViewContainer
      maxWidth="7xl"
      innerClassName="py-4"
      topBar={
        <TopBar
          left={<Heading title="Labels" as="h1" />}
          center={<SearchInput value={search} onChange={setSearch} placeholder="Filter labels..." />}
          right={
            <button
              onClick={() => { setCreating(true); setEditing(null); }}
              className="flex items-center gap-2 h-7 px-3 text-xs font-medium rounded-[var(--radius-md)] bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] transition-colors"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
              </svg>
              New Label
            </button>
          }
        />
      }
    >
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
            <Text size="xs">Color</Text>
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

      {/* Label lists */}
      {visibleBuiltin.length === 0 && visibleCustom.length === 0 && search.trim() ? (
        <div className="flex justify-center py-16">
          <Text as="p">No labels match “{search.trim()}”</Text>
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          {visibleBuiltin.length > 0 && (
            <LabelSection title="Built-in labels">
              {visibleBuiltin.map(renderBuiltinRow)}
            </LabelSection>
          )}
          {(visibleCustom.length > 0 || !search.trim()) && (
            <LabelSection title="Custom labels">
              {customLabels.length === 0 && !creating ? (
                <div className="flex flex-col items-center justify-center py-10 gap-3">
                  <svg className="w-10 h-10 text-[var(--color-text-muted)] opacity-40" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
                  </svg>
                  <Text as="p">No custom labels yet</Text>
                  <button
                    onClick={() => setCreating(true)}
                    className="text-xs text-[var(--color-accent-primary)] hover:underline"
                  >
                    Create your first label
                  </button>
                </div>
              ) : (
                visibleCustom.map(renderCustomRow)
              )}
            </LabelSection>
          )}
        </div>
      )}
      {editing && (
        <EditLabelDialog
          key={editing.name}
          label={editing}
          sampleIssues={recentIssuesWithLabel(issues, editing.name, 2)}
          onSave={saveEdit}
          onDelete={handleDelete}
          onClose={() => setEditing(null)}
        />
      )}
    </ViewContainer>
  );
}

function LabelSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="px-4 pb-2 text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{title}</h2>
      <div className="divide-y divide-[var(--color-border-subtle)] border-y border-[var(--color-border-subtle)]">{children}</div>
    </section>
  );
}
