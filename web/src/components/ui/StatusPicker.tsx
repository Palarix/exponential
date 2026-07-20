import { useState, useEffect, useRef, useMemo } from "react";
import { STATUS_OPTIONS } from "../../constants";
import StatusIcon from "./StatusIcon";

interface StatusPickerProps {
  current: string;
  onSelect: (status: string) => void;
  onClose?: () => void;
}

export default function StatusPicker({ current, onSelect, onClose }: StatusPickerProps) {
  const [filter, setFilter] = useState("");
  const [focusIndex, setFocusIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = useMemo(() => {
    const q = filter.toLowerCase();
    return q ? STATUS_OPTIONS.filter(o => o.label.toLowerCase().includes(q)) : STATUS_OPTIONS;
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
      onSelect(filtered[focusIndex].value);
      return;
    }
    const num = parseInt(e.key);
    if (num >= 1 && num <= filtered.length) {
      e.preventDefault();
      onSelect(filtered[num - 1].value);
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
          placeholder="Change status..."
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="border-t border-[var(--color-border-subtle)]" />
      {filtered.map((opt, i) => {
        const isCurrent = opt.value === current;
        return (
          <button
            key={opt.value}
            onClick={() => onSelect(opt.value)}
            onMouseEnter={() => setFocusIndex(i)}
            className={`flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors hover:bg-[var(--color-hover-surface-3)] ${focusIndex === i ? "bg-[var(--color-hover-surface-3)]" : ""} ${isCurrent ? "text-[var(--color-accent-primary)]" : "text-[var(--color-text-primary)]"}`}
          >
            <StatusIcon status={opt.value} size={14} />
            <span>{opt.label}</span>
            <span className="ml-auto w-4 flex items-center justify-center shrink-0">
              {isCurrent ? <CheckIcon /> : <span className="text-xs text-[var(--color-text-muted)]">{i + 1}</span>}
            </span>
          </button>
        );
      })}
      {filtered.length === 0 && <div className="px-3 py-2 text-sm text-[var(--color-text-muted)]">No matching statuses</div>}
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
