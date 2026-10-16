import { useState, useMemo, useContext, type ReactNode } from "react";
import { LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext } from "./BadgeContexts";
import { labelColor } from "../../utils/labels";
import { Menu, MenuItem, MenuFilter, MenuDivider, MenuLabel } from "./Menu";

interface LabelPickerProps {
  allLabels: string[];
  selected: string[];
  onToggle: (label: string) => void;
  /** Offer "Create" for a new label, which applies it uncolored. Colors are set in the Labels view. */
  canCreateLabels?: boolean;
  onClose?: () => void;
  singleSelect?: boolean;
  exclude?: string[];
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
  canCreateLabels = false,
  onClose,
  // singleSelect — accepted for API compatibility, no longer affects rendering
  exclude,
}: LabelPickerProps) {
  const configLabels = useContext(LabelColorsContext);
  const hideDefaultLabels = useContext(HideDefaultLabelsContext);
  const defaultLabels = useContext(DefaultLabelsContext);
  const [search, setSearch] = useState("");

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

  const handleCreate = () => {
    onToggle(search.trim());
    setSearch("");
  };

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
      {canCreate && canCreateLabels && (
        <MenuItem
          label={`Create "${search.trim()}"`}
          onClick={handleCreate}
        />
      )}
      {filtered.length === 0 && !canCreate && (
        <MenuLabel>No matching labels</MenuLabel>
      )}
    </Menu>
  );
}
