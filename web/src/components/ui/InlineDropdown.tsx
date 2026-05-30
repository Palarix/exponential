import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";

export interface DropdownOption {
  value: string;
  label: string;
  dot?: string;
}

export default function InlineDropdown({
  placeholder,
  options,
  value,
  onChange,
  required,
}: {
  placeholder: string;
  options: DropdownOption[];
  value: string;
  onChange: (v: string) => void;
  required?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const selected = options.find((o) => o.value === value);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        menuRef.current && !menuRef.current.contains(e.target as Node) &&
        btnRef.current && !btnRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const handleOpen = () => {
    if (btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setPos({ top: rect.bottom + 4, left: rect.left });
    }
    setOpen(!open);
  };

  return (
    <div>
      <button
        ref={btnRef}
        type="button"
        onClick={handleOpen}
        className={`flex items-center gap-2 h-8 px-3 rounded-[var(--radius-md)] text-sm transition-colors border border-[var(--color-border-default)] hover:border-[var(--color-border-focus)] ${!selected && required ? "border-[var(--color-error)]/40" : ""}`}
      >
        {selected ? (
          <>
            {selected.dot && <span className="w-2 h-2 rounded-full" style={{ background: selected.dot }} />}
            <span className="text-[var(--color-text-primary)]">{selected.label}</span>
          </>
        ) : (
          <span className="text-[var(--color-text-muted)]">{placeholder}</span>
        )}
        <svg className="w-3 h-3 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && createPortal(
        <div
          ref={menuRef}
          className="fixed z-[100] min-w-[160px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
          style={{ top: pos.top, left: pos.left }}
        >
          {options.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className="flex items-center gap-2 w-full h-8 px-3 text-sm hover:bg-[var(--color-bg-hover)] transition-colors"
            >
              {opt.dot && <span className="w-2 h-2 rounded-full shrink-0" style={{ background: opt.dot }} />}
              <span className="text-[var(--color-text-primary)]">{opt.label}</span>
              {opt.value === value && (
                <svg className="w-4 h-4 ml-auto text-[var(--color-accent-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              )}
            </button>
          ))}
        </div>,
        document.body,
      )}
    </div>
  );
}
