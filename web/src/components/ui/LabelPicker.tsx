import { useState, useMemo, useContext, useCallback, type ReactNode } from "react";
import { LabelBadge } from "./Badge";
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext } from "./BadgeContexts";
import { LABEL_PRESET_COLORS } from "../../constants";
import { addConfigLabel } from "../../api/client";
import { labelColor } from "../../utils/labels";
import { Menu, MenuItem, MenuFilter, MenuDivider, MenuLabel } from "./Menu";

interface LabelPickerProps {
  allLabels: string[];
  selected: string[];
  onToggle: (label: string) => void;
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  onClose?: () => void;
  singleSelect?: boolean;
  exclude?: string[];
  borderlessBadges?: boolean;
}

function displayLabel(label: string): string {
  return label === label.toLowerCase()
    ? label.charAt(0).toUpperCase() + label.slice(1)
    : label;
}

export default function LabelPicker({
  allLabels,
  selected,
  onToggle,
  onConfigLabelsChange,
  onClose,
  // singleSelect — accepted for API compatibility, no longer affects rendering
  exclude,
  borderlessBadges,
}: LabelPickerProps) {
  const configLabels = useContext(LabelColorsContext);
  const hideDefaultLabels = useContext(HideDefaultLabelsContext);
  const defaultLabels = useContext(DefaultLabelsContext);
  const [search, setSearch] = useState("");
  const [creatingLabel, setCreatingLabel] = useState<string | null>(null);

  const excludeSet = useMemo(
    () => new Set((exclude || []).map((l) => l.toLowerCase())),
    [exclude],
  );

  const orderedLabels = useMemo(() => {
    const keep = (l: string) => !excludeSet.has(l.toLowerCase());
    if (hideDefaultLabels) {
      return [...allLabels]
        .filter(keep)
        .sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));
    }
    const defaultNames = defaultLabels.map((d) => d.name).filter(keep);
    const defaultSet = new Set(defaultNames.map((n) => n.toLowerCase()));
    const others = allLabels
      .filter((l) => !defaultSet.has(l.toLowerCase()) && keep(l))
      .sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()));
    return [...defaultNames, ...others];
  }, [allLabels, hideDefaultLabels, excludeSet, defaultLabels]);

  const defaultLabelSet = useMemo(
    () => new Set(defaultLabels.map((d) => d.name.toLowerCase())),
    [defaultLabels],
  );

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return orderedLabels;
    return orderedLabels.filter((l) => l.toLowerCase().includes(q));
  }, [orderedLabels, search]);

  const canCreate =
    search.trim().length > 0 &&
    !orderedLabels.some((l) => l.toLowerCase() === search.trim().toLowerCase());

  const handleSelect = useCallback(
    (name: string) => {
      const hasColor = labelColor(name, configLabels) !== "var(--color-text-muted)";
      if (!hasColor && onConfigLabelsChange) {
        setCreatingLabel(name);
      } else {
        onToggle(name);
        setSearch("");
      }
    },
    [configLabels, onConfigLabelsChange, onToggle],
  );

  const handleCreateLabel = useCallback(
    async (name: string, color: string) => {
      await addConfigLabel(name, color);
      onConfigLabelsChange?.({ ...configLabels, [name]: color });
      setCreatingLabel(null);
      setSearch("");
      onToggle(name);
    },
    [configLabels, onConfigLabelsChange, onToggle],
  );

  if (creatingLabel) {
    return (
      <>
        <div className="px-3 py-2 flex items-center gap-2">
          <button
            onClick={() => setCreatingLabel(null)}
            className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 19l-7-7 7-7"
              />
            </svg>
          </button>
          <span className="text-xs text-[var(--color-text-muted)]">
            Pick a color for
          </span>
          <LabelBadge borderless={borderlessBadges} label={creatingLabel} />
        </div>
        <div className="border-t border-[var(--color-border-subtle)]" />
        <div className="flex items-center gap-2 px-3 py-3">
          {LABEL_PRESET_COLORS.map((color) => (
            <button
              key={color}
              onClick={() => handleCreateLabel(creatingLabel, color)}
              className="w-6 h-6 rounded-full border-2 border-transparent hover:border-[var(--color-text-primary)] transition-colors hover:scale-110"
              style={{ background: color }}
            />
          ))}
        </div>
      </>
    );
  }

  const items: ReactNode[] = [];
  for (let i = 0; i < filtered.length; i++) {
    const label = filtered[i];
    const isActive = selected.some(s => s.toLowerCase() === label.toLowerCase());
    const isDefault = !hideDefaultLabels && defaultLabelSet.has(label.toLowerCase());
    const prev = i > 0 ? filtered[i - 1] : null;
    const prevIsDefault =
      !hideDefaultLabels && prev !== null && defaultLabelSet.has(prev.toLowerCase());
    if (!isDefault && prevIsDefault) {
      items.push(<MenuDivider key={`div-${label}`} />);
    }
    items.push(
      <MenuItem
        key={label}
        label={displayLabel(label)}
        icon={
          <span
            className="w-2 h-2 rounded-full shrink-0"
            style={{ background: labelColor(label, configLabels) }}
          />
        }
        checked={isActive}
        onClick={() => onToggle(label)}
      />,
    );
  }

  return (
    <Menu onClose={onClose} bare autoFocus={false}>
      <MenuFilter
        value={search}
        onChange={setSearch}
        placeholder="Filter or create label..."
      />
      {items}
      {canCreate && onConfigLabelsChange && (
        <MenuItem
          label={`Create "${search.trim()}"`}
          onClick={() => handleSelect(search.trim())}
        />
      )}
      {filtered.length === 0 && !canCreate && (
        <MenuLabel>No matching labels</MenuLabel>
      )}
    </Menu>
  );
}
