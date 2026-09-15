import { useState, useRef, useEffect, useLayoutEffect, useMemo, useContext, type ReactNode } from "react";
import { createPortal } from "react-dom";
import type { Issue } from "../../api/client";
import { StatusIcon, Avatar, PriorityIcon } from "../ui";
import { labelColor } from "../../utils/labels";
import { LabelColorsContext } from "../ui/BadgeContexts";
import { STATUS_OPTIONS, PRIORITY_OPTIONS } from "../../constants";
import { getSelected, setSelected, type BacklogFilters, type FilterDimension } from "./filters";
import { Menu, MenuItem, MenuLabel, MenuDivider } from "../ui/Menu";
import { SubMenu } from "../ui/SubMenu";

interface FilterMenuProps {
  issues: Issue[];
  filters: BacklogFilters;
  onChange: (filters: BacklogFilters) => void;
  anchorRef: React.RefObject<HTMLButtonElement | null>;
  onClose: () => void;
}

type Dimension = FilterDimension;

const DIMENSIONS: { key: Dimension; label: string; icon: React.ReactNode }[] = [
  { key: "status", label: "Status", icon: <StatusIcon status="PLANNED" size={14} /> },
  {
    key: "assignee", label: "Assignee",
    icon: <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0" /></svg>,
  },
  { key: "priority", label: "Priority", icon: <PriorityIcon priority={2} size={14} /> },
  {
    key: "labels", label: "Labels",
    icon: <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" /></svg>,
  },
  {
    key: "epic", label: "Epic",
    icon: <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}><path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6z" /><path strokeLinecap="round" strokeLinejoin="round" d="M3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25z" /><path strokeLinecap="round" strokeLinejoin="round" d="M13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6z" /></svg>,
  },
];

interface FilterOption {
  value: string;
  label: string;
  icon?: ReactNode;
  count: number;
}

function computeFilterOptions(dim: Dimension, issues: Issue[], configColors?: Record<string, string>): FilterOption[] {
  switch (dim) {
    case "status":
      return STATUS_OPTIONS.map((s) => ({
        value: s.value, label: s.label,
        icon: <StatusIcon status={s.value} size={14} />,
        count: issues.filter((i) => i.status === s.value).length,
      }));
    case "assignee": {
      const counts = new Map<string, number>();
      let unassigned = 0;
      for (const i of issues) {
        if (i.assignee) counts.set(i.assignee, (counts.get(i.assignee) || 0) + 1);
        else unassigned++;
      }
      const rows: FilterOption[] = Array.from(counts.entries())
        .sort((a, b) => b[1] - a[1])
        .map(([assignee, count]) => ({
          value: assignee, label: assignee.split(" <")[0],
          icon: <Avatar name={assignee} size="xs" />,
          count,
        }));
      if (unassigned > 0) rows.push({ value: "__unassigned__", label: "Unassigned", count: unassigned, icon: <Avatar size="xs" /> });
      return rows;
    }
    case "priority":
      return PRIORITY_OPTIONS.map((p) => ({
        value: String(p.value), label: p.label,
        icon: <PriorityIcon priority={p.value} size={14} />,
        count: issues.filter((i) => (i.priority || 0) === p.value).length,
      }));
    case "labels": {
      const counts = new Map<string, number>();
      for (const i of issues) for (const l of i.labels || []) counts.set(l, (counts.get(l) || 0) + 1);
      return Array.from(counts.entries()).sort((a, b) => b[1] - a[1]).map(([label, count]) => {
        const display = label === label.toLowerCase() ? label.charAt(0).toUpperCase() + label.slice(1) : label;
        const color = labelColor(label, configColors ?? {});
        return {
          value: label, label: display, count,
          icon: <span className="w-2 h-2 rounded-full shrink-0" style={{ background: color }} />,
        };
      });
    }
    case "epic":
      return issues.filter((i) => i.labels?.includes("epic")).map((i) => ({
        value: i.id, label: i.title, count: issues.filter((c) => c.parent_id === i.id).length,
      }));
  }
}

export default function FilterMenu({ issues, filters, onChange, anchorRef, onClose }: FilterMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const configColors = useContext(LabelColorsContext);
  const [searches, setSearches] = useState<Partial<Record<Dimension, string>>>({});

  const allOptions = useMemo(() => {
    const result: Record<Dimension, FilterOption[]> = {} as Record<Dimension, FilterOption[]>;
    for (const dim of DIMENSIONS) result[dim.key] = computeFilterOptions(dim.key, issues, configColors);
    return result;
  }, [issues, configColors]);

  useLayoutEffect(() => {
    const menu = menuRef.current;
    if (!menu) return;

    const position = () => {
      const anchor = anchorRef.current;
      if (!anchor) return;
      const aRect = anchor.getBoundingClientRect();
      const mRect = menu.getBoundingClientRect();
      let top = aRect.bottom + 4;
      let left = aRect.left + aRect.width / 2 - mRect.width / 2;
      if (top + mRect.height > window.innerHeight - 8) top = aRect.top - mRect.height - 4;
      if (left < 8) left = 8;
      if (left + mRect.width > window.innerWidth - 8) left = window.innerWidth - mRect.width - 8;
      menu.style.top = `${top}px`;
      menu.style.left = `${left}px`;
      menu.style.visibility = "visible";
    };

    if (anchorRef.current) {
      position();
    } else {
      queueMicrotask(position);
    }
  });

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (menuRef.current?.contains(target)) return;
      if (anchorRef.current?.contains(target)) return;
      const flyout = document.querySelector("[data-submenu-flyout]");
      if (flyout?.contains(target)) return;
      onClose();
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose, anchorRef]);

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
    <div ref={menuRef} style={{ position: "fixed", visibility: "hidden" }} className="z-50">
      <Menu onClose={onClose} className="w-52 rounded-[var(--radius-md)]" aria-label="Filter dimensions">
        <MenuLabel>Add Filter...</MenuLabel>
        {DIMENSIONS.map((dim) => {
          const selected = getSelected(dim.key, filters);
          const options = allOptions[dim.key];
          const search = searches[dim.key] ?? "";
          const filtered = search
            ? options.filter(o => o.label.toLowerCase().includes(search.toLowerCase()))
            : options;
          const showSearch = options.length > 5;
          return (
            <SubMenu
              key={dim.key}
              label={dim.label}
              icon={dim.icon}
              maxHeight={showSearch ? "24rem" : undefined}
              suffix={selected.length > 0 ? <span className="text-xs text-[var(--color-accent-primary)] tabular-nums">{selected.length}</span> : undefined}
            >
              {showSearch && (
                <MenuLabel className="px-3 py-1.5 normal-case tracking-normal font-normal">
                  <input
                    autoFocus
                    value={search}
                    onChange={e => setSearches(s => ({ ...s, [dim.key]: e.target.value }))}
                    onKeyDown={e => { if (e.key === "Escape" && search) { e.stopPropagation(); setSearches(s => ({ ...s, [dim.key]: "" })); } }}
                    placeholder="Filter..."
                    className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
                  />
                </MenuLabel>
              )}
              {showSearch && <MenuDivider />}
              {filtered.map((opt) => (
                <MenuItem
                  key={opt.value}
                  label={opt.label}
                  icon={opt.icon}
                  checked={selected.includes(opt.value)}
                  onClick={() => handleToggle(dim.key, opt.value)}
                  suffix={<span className="text-[var(--color-text-muted)] tabular-nums shrink-0 text-xs">{opt.count}</span>}
                />
              ))}
              {filtered.length === 0 && <MenuLabel>No matches</MenuLabel>}
            </SubMenu>
          );
        })}
      </Menu>
    </div>,
    document.body,
  );
}
