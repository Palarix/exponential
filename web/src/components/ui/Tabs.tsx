import { useCallback, useRef, useEffect, useMemo } from "react";
import { useKeyboardShortcuts } from "../../keyboard";
import { cycleTabs } from "./tabs-utils";

interface TabsProps<T extends string> {
  items: Readonly<Record<T, { label: string }>>;
  activeId: T;
  onChange: (id: T) => void;
  keyboardNavigationEnabled?: boolean;
}

export function Tabs<T extends string>({
  items,
  activeId,
  onChange,
  keyboardNavigationEnabled = true,
}: TabsProps<T>) {
  const entries = useMemo(
    () => Object.entries(items) as [T, { label: string }][],
    [items],
  );

  const activeIdRef = useRef(activeId);
  const onChangeRef = useRef(onChange);
  const entriesRef = useRef(entries);
  useEffect(() => { activeIdRef.current = activeId; }, [activeId]);
  useEffect(() => { onChangeRef.current = onChange; }, [onChange]);
  useEffect(() => { entriesRef.current = entries; }, [entries]);

  const cyclePrev = useCallback(() => {
    const next = cycleTabs(entriesRef.current.map(([k]) => k), activeIdRef.current, -1);
    if (next !== null) onChangeRef.current(next);
  }, []);

  const cycleNext = useCallback(() => {
    const next = cycleTabs(entriesRef.current.map(([k]) => k), activeIdRef.current, 1);
    if (next !== null) onChangeRef.current(next);
  }, []);

  useKeyboardShortcuts({
    scope: "tabs",
    priority: "control",
    enabled: keyboardNavigationEnabled,
    shortcuts: [
      {
        id: "tabs.prev",
        key: "[",
        label: "Previous tab",
        group: "Tabs",
        run: cyclePrev,
        preventDefault: true,
      },
      {
        id: "tabs.next",
        key: "]",
        label: "Next tab",
        group: "Tabs",
        run: cycleNext,
        preventDefault: true,
      },
    ],
  });

  return (
    <div className="flex items-center gap-4 h-full">
      {entries.map(([id, config]) => (
        <button
          key={id}
          onClick={() => onChange(id)}
          className={`text-sm font-medium h-full border-b-2 -mb-px transition-colors duration-[var(--duration-fast)] ${
            activeId === id
              ? "text-[var(--color-text-primary)] border-[var(--color-text-primary)]"
              : "text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]"
          }`}
        >
          {config.label}
        </button>
      ))}
    </div>
  );
}
