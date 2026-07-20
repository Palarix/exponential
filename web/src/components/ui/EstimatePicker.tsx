import { useState, useEffect, useRef, useMemo } from "react";
import { ESTIMATE_OPTIONS } from "../../constants";

interface EstimatePickerProps {
  current: number;
  onSelect: (estimate: number) => void;
  onClose?: () => void;
}

function estimateLabel(est: number): string {
  return est === 0 ? "No estimate" : `${est} Point${est !== 1 ? "s" : ""}`;
}

export default function EstimatePicker({ current, onSelect, onClose }: EstimatePickerProps) {
  const [filter, setFilter] = useState("");
  const [focusIndex, setFocusIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase();
    if (!q) return ESTIMATE_OPTIONS;
    return ESTIMATE_OPTIONS.filter(est => estimateLabel(est).toLowerCase().includes(q));
  }, [filter]);

  useEffect(() => {
    const raf = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(raf);
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      onClose?.();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setFocusIndex(i => Math.min(i + 1, filtered.length - 1));
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setFocusIndex(i => Math.max(i - 1, 0));
      return;
    }
    if (e.key === "Enter" && focusIndex >= 0 && focusIndex < filtered.length) {
      e.preventDefault();
      onSelect(filtered[focusIndex]);
      return;
    }
    const num = parseInt(e.key);
    if (num >= 1 && num <= filtered.length) {
      e.preventDefault();
      onSelect(filtered[num - 1]);
    }
  };

  return (
    <div>
      <div className="px-3 py-2">
        <input
          ref={inputRef}
          value={filter}
          onChange={e => setFilter(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Set estimate..."
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="border-t border-[var(--color-border-subtle)]" />
      {filtered.map((est, i) => {
        const isCurrent = est === current;
        return (
          <button
            key={est}
            onClick={() => onSelect(est)}
            onMouseEnter={() => setFocusIndex(i)}
            className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${focusIndex === i ? "bg-[var(--color-hover-surface-3)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
          >
            <span>{estimateLabel(est)}</span>
            <span className="ml-auto w-4 flex items-center justify-center shrink-0">
              {isCurrent ? <CheckIcon /> : <span className="text-xs text-[var(--color-text-muted)]">{i + 1}</span>}
            </span>
          </button>
        );
      })}
      {filtered.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching estimates</div>}
    </div>
  );
}

function CheckIcon() {
  return (
    <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
    </svg>
  );
}
