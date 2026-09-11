import { useCallback, useRef } from "react";
import { useKeyboardShortcuts } from "../../keyboard";
import { searchInputKbdHint } from "./search-input-utils";

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  keyboardNavigationEnabled?: boolean;
}

export function SearchInput({
  value,
  onChange,
  placeholder = "Search...",
  keyboardNavigationEnabled = true,
}: SearchInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  const focusInput = useCallback(() => {
    inputRef.current?.focus();
  }, []);

  useKeyboardShortcuts({
    scope: "search-input",
    priority: "control",
    enabled: keyboardNavigationEnabled,
    shortcuts: [
      {
        id: "search.focus",
        key: "/",
        label: "Focus search",
        group: "Search",
        run: focusInput,
        preventDefault: true,
      },
    ],
  });

  return (
    <div className="relative w-full max-w-md">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            onChange("");
            inputRef.current?.blur();
          }
        }}
        placeholder={placeholder}
        className="text-sm h-8 pl-8 pr-10 w-full rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-0)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-border-focus)] placeholder:text-[var(--color-text-muted)]"
      />
      <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
      </svg>
      <kbd className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-[var(--color-text-muted)] border border-[var(--color-border-subtle)] rounded px-1 py-0.5 leading-none pointer-events-none">
        {searchInputKbdHint(value)}
      </kbd>
    </div>
  );
}
