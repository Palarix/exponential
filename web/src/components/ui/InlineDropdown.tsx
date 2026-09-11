import { useState, useRef, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import Popover from "./Popover";

export interface DropdownOption {
  value: string;
  label: string;
  dot?: string;
  icon?: ReactNode;
}

export default function InlineDropdown({
  placeholder,
  options,
  value,
  onChange,
  required,
  borderless,
}: {
  placeholder: string;
  options: DropdownOption[];
  value: string;
  onChange: (v: string) => void;
  required?: boolean;
  borderless?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const selected = options.find((o) => o.value === value);

  return (
    <div>
      <button
        ref={btnRef}
        type="button"
        onClick={() => setOpen(!open)}
        className={`flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm transition-colors ${borderless ? "hover:bg-[var(--color-hover-surface-3)]" : `border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] ${!selected && required ? "border-[var(--color-error)]/40" : ""}`}`}
      >
        {selected ? (
          <>
            {selected.icon || (selected.dot && <span className="w-2 h-2 rounded-full" style={{ background: selected.dot }} />)}
            <span className="text-[var(--color-text-primary)]">{selected.label}</span>
          </>
        ) : (
          <span className="text-[var(--color-text-muted)]">{placeholder}</span>
        )}
        <ChevronDown className="w-3 h-3 text-[var(--color-text-muted)]" />
      </button>
      {open && (
        <Popover anchorRef={btnRef} onClose={() => setOpen(false)} className="min-w-40">
          {options.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className="flex items-center gap-2 w-full h-8 px-3 text-sm hover:bg-[var(--color-hover-surface-3)] transition-colors"
            >
              {opt.icon || (opt.dot && <span className="w-2 h-2 rounded-full shrink-0" style={{ background: opt.dot }} />)}
              <span className="text-[var(--color-text-primary)]">{opt.label}</span>
              {opt.value === value && (
                <svg className="w-4 h-4 ml-auto text-[var(--color-accent-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              )}
            </button>
          ))}
        </Popover>
      )}
    </div>
  );
}
