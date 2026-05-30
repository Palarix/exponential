import { useState, useMemo, useContext, useCallback, useRef, useEffect } from "react";
import { LabelBadge, LabelColorsContext, HideDefaultLabelsContext, DefaultLabelsContext } from "./Badge";
import { LABEL_PRESET_COLORS } from "../../constants";
import { addConfigLabel } from "../../api/client";

interface LabelPickerProps {
  allLabels: string[];
  selected: string[];
  onToggle: (label: string) => void;
  onConfigLabelsChange?: (labels: Record<string, string>) => void;
  onClose?: () => void;
  singleSelect?: boolean;
  exclude?: string[];
}

export default function LabelPicker({
  allLabels,
  selected,
  onToggle,
  onConfigLabelsChange,
  onClose,
  singleSelect = false,
  exclude,
}: LabelPickerProps) {
  const configLabels = useContext(LabelColorsContext);
  const hideDefaultLabels = useContext(HideDefaultLabelsContext);
  const defaultLabels = useContext(DefaultLabelsContext);
  const [search, setSearch] = useState("");
  const [focusIndex, setFocusIndex] = useState(0);
  const [creatingLabel, setCreatingLabel] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  const excludeSet = useMemo(
    () => new Set((exclude || []).map((l) => l.toLowerCase())),
    [exclude],
  );

  // Defaults first (in canonical order), then the rest sorted alphabetically. Case-insensitive dedup.
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
      const hasColor =
        configLabels[name] || configLabels[name.toLowerCase()];
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

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const total = filtered.length + (canCreate ? 1 : 0);
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFocusIndex((i) => Math.min(i + 1, total - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setFocusIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        e.stopPropagation();
        if (canCreate && focusIndex === filtered.length) {
          handleSelect(search.trim());
        } else if (filtered[focusIndex]) {
          onToggle(filtered[focusIndex]);
        }
      } else if (e.key === " ") {
        const endsWithSpace = search.length > 0 && search[search.length - 1] === " ";
        if (search.length === 0 || endsWithSpace) {
          e.preventDefault();
          if (canCreate && focusIndex === filtered.length) {
            handleSelect(search.trim());
          } else if (filtered[focusIndex]) {
            onToggle(filtered[focusIndex]);
          }
        }
      } else if (e.key === "Escape") {
        onClose?.();
      }
    },
    [filtered, canCreate, focusIndex, search, handleSelect, onToggle, onClose],
  );

  if (creatingLabel) {
    return (
      <>
        <div className="px-3 py-1.5 flex items-center gap-2">
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
          <LabelBadge label={creatingLabel} />
        </div>
        <div className="border-t border-[var(--color-border-subtle)]" />
        <div className="flex items-center gap-2 px-3 py-2.5">
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

  return (
    <>
      <div className="px-3 py-1.5">
        <input
          ref={inputRef}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setFocusIndex(0);
          }}
          onKeyDown={handleKeyDown}
          placeholder="Filter or create label..."
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="border-t border-[var(--color-border-subtle)]" />
      {filtered.map((label, i) => {
        const isActive = selected.includes(label);
        const isFocused = i === focusIndex;
        const isDefault = !hideDefaultLabels && defaultLabelSet.has(label.toLowerCase());
        const prev = i > 0 ? filtered[i - 1] : null;
        const prevIsDefault =
          !hideDefaultLabels && prev !== null && defaultLabelSet.has(prev.toLowerCase());
        const showDivider = !isDefault && prevIsDefault;
        return (
          <div key={label}>
            {showDivider && (
              <div className="my-1 border-t border-[var(--color-border-default)]" />
            )}
          <button
            onClick={() => onToggle(label)}
            onMouseEnter={() => setFocusIndex(i)}
            className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-bg-hover)] ${isFocused ? "bg-[var(--color-bg-hover)]" : ""}`}
          >
            {singleSelect ? (
              <span
                className={`w-4 h-4 rounded-full border flex items-center justify-center shrink-0 ${isActive ? "border-[var(--color-accent-primary)]" : "border-[var(--color-border-default)]"}`}
              >
                {isActive && (
                  <span className="w-2 h-2 rounded-full bg-[var(--color-accent-primary)]" />
                )}
              </span>
            ) : (
              <span
                className={`w-4 h-4 rounded-[3px] border flex items-center justify-center shrink-0 ${isActive ? "bg-[var(--color-accent-primary)] border-[var(--color-accent-primary)]" : "border-[var(--color-border-default)]"}`}
              >
                {isActive && (
                  <svg
                    className="w-3 h-3 text-white"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={3}
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                )}
              </span>
            )}
            <LabelBadge label={label} />
          </button>
          </div>
        );
      })}
      {canCreate && onConfigLabelsChange && (
        <button
          onClick={() => handleSelect(search.trim())}
          onMouseEnter={() => setFocusIndex(filtered.length)}
          className={`flex items-center gap-2 w-full px-3 py-1.5 text-sm transition-colors hover:bg-[var(--color-bg-hover)] ${focusIndex === filtered.length ? "bg-[var(--color-bg-hover)]" : ""}`}
        >
          <span className="text-[var(--color-text-muted)]">Create</span>
          <LabelBadge label={search.trim()} />
        </button>
      )}
      {filtered.length === 0 && !canCreate && (
        <div className="px-3 py-1.5 text-sm text-[var(--color-text-muted)]">
          No matching labels
        </div>
      )}
    </>
  );
}
