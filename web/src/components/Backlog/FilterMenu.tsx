import { useState, useRef, useEffect, useMemo, forwardRef } from "react";
import { createPortal } from "react-dom";
import type { Issue } from "../../api/client";
import { StatusIcon, LabelBadge, Avatar, PriorityIcon, ChevronRightIcon } from "../ui";
import { STATUS_OPTIONS, PRIORITY_OPTIONS } from "../../constants";
import type { BacklogFilters } from "./filters";

interface FilterMenuProps {
  issues: Issue[];
  filters: BacklogFilters;
  onChange: (filters: BacklogFilters) => void;
  anchorRef: React.RefObject<HTMLButtonElement | null>;
  onClose: () => void;
}

type Dimension = "status" | "assignee" | "priority" | "labels" | "epic";

const DIMENSIONS: { key: Dimension; label: string; icon: React.ReactNode }[] = [
  {
    key: "status",
    label: "Status",
    icon: <StatusIcon status="PLANNED" size={14} />,
  },
  {
    key: "assignee",
    label: "Assignee",
    icon: (
      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0" />
      </svg>
    ),
  },
  {
    key: "priority",
    label: "Priority",
    icon: <PriorityIcon priority={2} size={14} />,
  },
  {
    key: "labels",
    label: "Labels",
    icon: (
      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
      </svg>
    ),
  },
  {
    key: "epic",
    label: "Epic",
    icon: (
      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6z" />
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25z" />
        <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6z" />
      </svg>
    ),
  },
];

interface SubMenuOption {
  value: string;
  label: React.ReactNode;
  count: number;
}

function useSubMenuOptions(
  dim: Dimension,
  issues: Issue[],
): SubMenuOption[] {
  return useMemo(() => {
    switch (dim) {
      case "status":
        return STATUS_OPTIONS.map((s) => ({
          value: s.value,
          label: (
            <span className="flex items-center gap-2">
              <StatusIcon status={s.value} size={14} />
              {s.label}
            </span>
          ),
          count: issues.filter((i) => i.status === s.value).length,
        }));
      case "assignee": {
        const counts = new Map<string, number>();
        let unassigned = 0;
        for (const i of issues) {
          if (i.assignee) counts.set(i.assignee, (counts.get(i.assignee) || 0) + 1);
          else unassigned++;
        }
        const rows: SubMenuOption[] = Array.from(counts.entries())
          .sort((a, b) => b[1] - a[1])
          .map(([assignee, count]) => ({
            value: assignee,
            label: (
              <span className="flex items-center gap-2">
                <Avatar name={assignee} size="xs" />
                <span className="truncate">{assignee.split(" <")[0]}</span>
              </span>
            ),
            count,
          }));
        if (unassigned > 0) {
          rows.push({
            value: "__unassigned__",
            label: <span className="italic text-[var(--color-text-muted)]">Unassigned</span>,
            count: unassigned,
          });
        }
        return rows;
      }
      case "priority":
        return PRIORITY_OPTIONS.map((p) => ({
          value: String(p.value),
          label: (
            <span className="flex items-center gap-2">
              <PriorityIcon priority={p.value} size={14} />
              {p.label}
            </span>
          ),
          count: issues.filter((i) => (i.priority || 0) === p.value).length,
        }));
      case "labels": {
        const counts = new Map<string, number>();
        for (const i of issues) for (const l of i.labels || []) counts.set(l, (counts.get(l) || 0) + 1);
        return Array.from(counts.entries())
          .sort((a, b) => b[1] - a[1])
          .map(([label, count]) => ({
            value: label,
            label: <LabelBadge label={label} />,
            count,
          }));
      }
      case "epic":
        return issues
          .filter((i) => i.labels?.includes("epic"))
          .map((i) => ({
            value: i.id,
            label: <span className="truncate">{i.title}</span>,
            count: issues.filter((c) => c.parent_id === i.id).length,
          }));
    }
  }, [dim, issues]);
}

function getSelected(dim: Dimension, filters: BacklogFilters): string[] {
  switch (dim) {
    case "status": return filters.statuses;
    case "assignee": return filters.assignees;
    case "priority": return filters.priorities.map(String);
    case "labels": return filters.labels;
    case "epic": return filters.epicId ? [filters.epicId] : [];
  }
}

function setSelected(dim: Dimension, filters: BacklogFilters, values: string[]): BacklogFilters {
  switch (dim) {
    case "status": return { ...filters, statuses: values };
    case "assignee": return { ...filters, assignees: values };
    case "priority": return { ...filters, priorities: values.map(Number) };
    case "labels": return { ...filters, labels: values };
    case "epic": return { ...filters, epicId: values[0] || null };
  }
}

function SubMenu({
  options,
  selected,
  onToggle,
  focusIndex,
  hasFocus,
}: {
  options: SubMenuOption[];
  selected: string[];
  onToggle: (value: string) => void;
  focusIndex?: number;
  hasFocus?: boolean;
}) {
  const [search, setSearch] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (hasFocus) inputRef.current?.focus();
  }, [hasFocus]);

  const filtered = search
    ? options.filter((o) => {
        const text = typeof o.label === "string" ? o.label : o.value;
        return text.toLowerCase().includes(search.toLowerCase());
      })
    : options;

  const effectiveIndex = hasFocus ? Math.min(focusIndex ?? 0, filtered.length - 1) : -1;

  return (
    <div className="py-1 w-52">
      {options.length > 5 && (
        <div className="px-2 pb-1">
          <input
            ref={inputRef}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter..."
            className="w-full px-2 py-1 text-xs bg-[var(--color-surface-1)] rounded-[var(--radius-sm)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none border border-[var(--color-border-subtle)] focus:border-[var(--color-border-default)]"
          />
        </div>
      )}
      <div className="max-h-64 overflow-y-auto">
        {filtered.map((opt, i) => {
          const isSelected = selected.includes(opt.value);
          const isFocused = i === effectiveIndex;
          return (
            <button
              key={opt.value}
              data-filter-option
              onClick={() => onToggle(opt.value)}
              className={`flex items-center gap-2 w-full px-3 py-1.5 text-xs text-left transition-colors ${isFocused ? "bg-[var(--color-hover-surface-3)]" : "hover:bg-[var(--color-hover-surface-3)]"}`}
            >
              <span className={`w-3.5 h-3.5 rounded-sm border flex items-center justify-center shrink-0 ${isSelected ? "bg-[var(--color-accent-primary)] border-[var(--color-accent-primary)]" : "border-[var(--color-border-control)]"}`}>
                {isSelected && (
                  <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                )}
              </span>
              <span className="flex-1 min-w-0">{opt.label}</span>
              <span className="text-[var(--color-text-muted)] tabular-nums shrink-0">{opt.count}</span>
            </button>
          );
        })}
        {filtered.length === 0 && (
          <div className="px-3 py-2 text-xs text-[var(--color-text-muted)]">No matches</div>
        )}
      </div>
    </div>
  );
}

export default function FilterMenu({ issues, filters, onChange, anchorRef, onClose }: FilterMenuProps) {
  const [openDim, setOpenDim] = useState<Dimension | null>(DIMENSIONS[0].key);
  const [inSubMenu, setInSubMenu] = useState(false);
  const [subFocusIndex, setSubFocusIndex] = useState(0);
  const menuRef = useRef<HTMLDivElement>(null);
  const subMenuRef = useRef<HTMLDivElement>(null);
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({ position: "fixed", visibility: "hidden" });
  const [subStyle, setSubStyle] = useState<React.CSSProperties>({ position: "fixed", visibility: "hidden" });
  const dimRowRefs = useRef<Map<Dimension, HTMLButtonElement>>(new Map());

  // Position the main menu below the anchor
  useEffect(() => {
    const anchor = anchorRef.current;
    const menu = menuRef.current;
    if (!anchor || !menu) return;
    const aRect = anchor.getBoundingClientRect();
    const mRect = menu.getBoundingClientRect();
    let top = aRect.bottom + 4;
    let left = aRect.right - mRect.width;
    if (top + mRect.height > window.innerHeight - 8) top = aRect.top - mRect.height - 4;
    if (left < 8) left = 8;
    setMenuStyle({ position: "fixed", top, left, visibility: "visible" });
  }, [anchorRef]);

  // Position sub-menu next to the hovered row, preferring right when space allows
  useEffect(() => {
    if (!openDim) { setSubStyle({ position: "fixed", visibility: "hidden" }); return; }
    const row = dimRowRefs.current.get(openDim);
    const menu = menuRef.current;
    if (!row || !menu) return;
    const rRect = row.getBoundingClientRect();
    const mRect = menu.getBoundingClientRect();
    const subWidth = 212;
    const spaceRight = window.innerWidth - mRect.right;
    const left = spaceRight >= subWidth ? mRect.right + 4 : mRect.left - subWidth - 4;
    setSubStyle({
      position: "fixed",
      top: rRect.top,
      left,
      visibility: "visible",
    });
  }, [openDim]);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (menuRef.current?.contains(target)) return;
      if (subMenuRef.current?.contains(target)) return;
      if (anchorRef.current?.contains(target)) return;
      onClose();
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose, anchorRef]);

  // Keyboard: Escape, arrow keys for main menu + sub-menu navigation
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Let typing work in the sub-menu search input, but still handle navigation keys
      const inInput = (e.target as HTMLElement)?.tagName === "INPUT";
      const navKey = e.key === "Escape" || e.key === "ArrowRight" || e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || e.key === " ";
      if (inInput && !navKey) return;

      if (e.key === "Escape") {
        e.preventDefault();
        if (inSubMenu) { (document.activeElement as HTMLElement)?.blur(); setInSubMenu(false); setSubFocusIndex(0); }
        else if (openDim) setOpenDim(null);
        else onClose();
        return;
      }

      const dimKeys = DIMENSIONS.map(d => d.key);

      if (inSubMenu && openDim) {
        const optionCount = subMenuRef.current?.querySelectorAll("button[data-filter-option]").length ?? 0;
        if (e.key === "ArrowDown") {
          e.preventDefault();
          if (inInput) (document.activeElement as HTMLElement)?.blur();
          setSubFocusIndex(i => Math.min(i + 1, optionCount - 1));
        } else if (e.key === "ArrowUp") {
          e.preventDefault();
          if (inInput) (document.activeElement as HTMLElement)?.blur();
          if (subFocusIndex <= 0) {
            const input = subMenuRef.current?.querySelector("input");
            if (input) { (input as HTMLElement).focus(); setSubFocusIndex(-1); return; }
          }
          setSubFocusIndex(i => Math.max(i - 1, 0));
        } else if (e.key === "ArrowRight") {
          e.preventDefault();
          (document.activeElement as HTMLElement)?.blur();
          setInSubMenu(false);
          setSubFocusIndex(0);
        } else if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          const buttons = subMenuRef.current?.querySelectorAll("button[data-filter-option]");
          if (buttons && buttons[subFocusIndex]) {
            (buttons[subFocusIndex] as HTMLButtonElement).click();
          }
        }
        return;
      }

      if (e.key === "ArrowDown") {
        e.preventDefault();
        const idx = openDim ? dimKeys.indexOf(openDim) : -1;
        const next = dimKeys[Math.min(idx + 1, dimKeys.length - 1)];
        setOpenDim(next);
        setInSubMenu(false);
        setSubFocusIndex(0);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        const idx = openDim ? dimKeys.indexOf(openDim) : dimKeys.length;
        const next = dimKeys[Math.max(idx - 1, 0)];
        setOpenDim(next);
        setInSubMenu(false);
        setSubFocusIndex(0);
      } else if ((e.key === "ArrowLeft" || e.key === "Enter" || e.key === " ") && openDim) {
        e.preventDefault();
        setInSubMenu(true);
        setSubFocusIndex(0);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [openDim, onClose, inSubMenu, subFocusIndex]);

  const handleToggle = (dim: Dimension, value: string) => {
    const isMulti = dim !== "epic";
    const current = getSelected(dim, filters);
    let next: string[];
    if (isMulti) {
      next = current.includes(value) ? current.filter((v) => v !== value) : [...current, value];
    } else {
      next = current.includes(value) ? [] : [value];
    }
    onChange(setSelected(dim, filters, next));
  };

  return createPortal(
    <>
      {/* Main menu */}
      <div ref={menuRef} style={menuStyle} className="z-50 w-52 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)] py-1">
        <div className="px-3 py-1.5 text-xs uppercase tracking-wider text-[var(--color-text-muted)]">
          Add Filter...
        </div>
        {DIMENSIONS.map((dim) => {
          const sel = getSelected(dim.key, filters);
          const isOpen = openDim === dim.key;
          return (
            <button
              key={dim.key}
              ref={(el) => { if (el) dimRowRefs.current.set(dim.key, el); }}
              onMouseEnter={() => setOpenDim(dim.key)}
              onClick={() => setOpenDim(isOpen ? null : dim.key)}
              className={`flex items-center gap-2.5 w-full px-3 py-1.5 text-xs text-left transition-colors ${isOpen ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)]"}`}
            >
              <span className="text-[var(--color-text-muted)] shrink-0">{dim.icon}</span>
              <span className="flex-1">{dim.label}</span>
              {sel.length > 0 && (
                <span className="text-xs text-[var(--color-accent-primary)] tabular-nums">{sel.length}</span>
              )}
              <ChevronRightIcon className="w-3 h-3 text-[var(--color-text-muted)] shrink-0" />
            </button>
          );
        })}
      </div>

      {/* Sub-menu */}
      {openDim && (
        <SubMenuPortal ref={subMenuRef} dim={openDim} issues={issues} filters={filters} style={subStyle} onToggle={handleToggle} focusIndex={subFocusIndex} hasFocus={inSubMenu} />
      )}
    </>,
    document.body,
  );
}

const SubMenuPortal = forwardRef<HTMLDivElement, {
  dim: Dimension;
  issues: Issue[];
  filters: BacklogFilters;
  style: React.CSSProperties;
  onToggle: (dim: Dimension, value: string) => void;
  focusIndex: number;
  hasFocus: boolean;
}>(function SubMenuPortal({ dim, issues, filters, style, onToggle, focusIndex, hasFocus }, ref) {
  const options = useSubMenuOptions(dim, issues);
  const selected = getSelected(dim, filters);

  return (
    <div ref={ref} style={style} className="z-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] shadow-[var(--shadow-popover)]">
      <SubMenu
        options={options}
        selected={selected}
        onToggle={(value) => onToggle(dim, value)}
        focusIndex={focusIndex}
        hasFocus={hasFocus}
      />
    </div>
  );
});
