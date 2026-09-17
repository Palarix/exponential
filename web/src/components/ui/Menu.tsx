import {
  Children,
  isValidElement,
  cloneElement,
  useState,
  useCallback,
  useRef,
  useEffect,
  type ReactNode,
  type ReactElement,
} from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "../../utils/cn";
import { useKeyboardHandler } from "../../keyboard";
import { nextIndex, matchShortcut, pointInTriangle } from "./menu-utils";

// ─── Internal props injected by Menu via cloneElement ──

interface MenuItemInternalProps {
  _menuIndex?: number;
  _focused?: boolean;
  _onMouseEnter?: (index: number) => void;
  _onClick?: (index: number) => void;
  _subMenuOpen?: boolean;
  _onSubMenuClose?: () => void;
}

// ─── Child metadata collected during render ────────────

interface ChildMeta {
  index: number;
  navigable: boolean;
  shortcut?: string;
  isSubMenu: boolean;
  isCheckbox: boolean;
  onClick?: () => void;
}

// ─── Menu ──────────────────────────────────────────────

interface MenuProps {
  children: ReactNode;
  onClose?: () => void;
  onParentArrowLeft?: () => void;
  className?: string;
  maxHeight?: string;
  autoFocus?: boolean;
  bare?: boolean;
  "aria-label"?: string;
}

export function Menu({
  children,
  onClose,
  onParentArrowLeft,
  className,
  maxHeight,
  autoFocus = true,
  bare,
  "aria-label": ariaLabel,
}: MenuProps) {
  const [focusIndex, setFocusIndex] = useState(-1);
  const [openSubMenuIndex, setOpenSubMenuIndex] = useState<number | null>(null);

  const metas: ChildMeta[] = [];
  let childCount = 0;
  Children.forEach(children, (child) => {
    if (!isValidElement(child)) return;
    const props = child.props as Record<string, unknown>;
    const type = child.type;
    const isDivider = type === MenuDivider;
    const isLabel = type === MenuLabel;
    const isFilter = type === MenuFilter;
    const isSubMenu = (type as { _isSubMenu?: boolean })._isSubMenu === true
      || props.renderPanel !== undefined;
    const isCheckbox = props.checked !== undefined;
    metas.push({
      index: childCount,
      navigable: !isDivider && !isLabel && !isFilter && !props.disabled,
      shortcut: props.shortcut as string | undefined,
      isSubMenu,
      isCheckbox,
      onClick: props.onClick as (() => void) | undefined,
    });
    childCount++;
  });

  const skipSet = new Set<number>();
  const shortcutMap = new Map<number, string>();
  for (const m of metas) {
    if (!m.navigable) skipSet.add(m.index);
    if (m.navigable && m.shortcut) shortcutMap.set(m.index, m.shortcut);
  }

  const metasRef = useRef(metas);
  const skipSetRef = useRef(skipSet);
  const shortcutMapRef = useRef(shortcutMap);
  const itemCountRef = useRef(childCount);
  const openSubMenuRef = useRef(openSubMenuIndex);
  useEffect(() => {
    metasRef.current = metas;
    skipSetRef.current = skipSet;
    shortcutMapRef.current = shortcutMap;
    itemCountRef.current = childCount;
    openSubMenuRef.current = openSubMenuIndex;
  });

  const autoFocused = useRef(false);
  const focusViaKeyboard = useRef(false);
  useEffect(() => {
    if (!autoFocus || autoFocused.current) return;
    const timer = requestAnimationFrame(() => {
      if (itemCountRef.current > 0) {
        autoFocused.current = true;
        setFocusIndex(nextIndex(-1, itemCountRef.current, 1, skipSetRef.current));
      }
    });
    return () => cancelAnimationFrame(timer);
  }, [autoFocus]);

  const closeSubMenu = useCallback(() => {
    setOpenSubMenuIndex(null);
  }, []);

  const activateItem = useCallback((index: number) => {
    const meta = metasRef.current[index];
    if (!meta?.navigable) return;
    if (meta.isSubMenu) {
      setOpenSubMenuIndex(index);
    } else {
      meta.onClick?.();
      if (!meta.isCheckbox) onClose?.();
    }
  }, [onClose]);

  // ─── Safe triangle: suppress submenu switch during diagonal movement ──

  const mousePos = useRef<{ x: number; y: number } | null>(null);
  const safeTriangleTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingHover = useRef<number | null>(null);
  const menuContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = menuContainerRef.current;
    if (!container) return;
    const handleMouseMove = (e: MouseEvent) => {
      mousePos.current = { x: e.clientX, y: e.clientY };
    };
    container.addEventListener("mousemove", handleMouseMove);
    return () => container.removeEventListener("mousemove", handleMouseMove);
  }, []);

  useEffect(() => {
    return () => {
      if (safeTriangleTimer.current) clearTimeout(safeTriangleTimer.current);
    };
  }, []);

  const isInsideSafeTriangle = useCallback(() => {
    if (openSubMenuRef.current === null || !mousePos.current) return false;
    const flyout = document.querySelector("[data-submenu-flyout]") as HTMLElement;
    if (!flyout) return false;
    const rect = flyout.getBoundingClientRect();
    const { x, y } = mousePos.current;
    const container = menuContainerRef.current;
    if (!container) return false;
    const menuRect = container.getBoundingClientRect();
    const submenuIsRight = rect.left >= menuRect.right - 10;

    if (submenuIsRight) {
      return pointInTriangle(x, y, menuRect.right, rect.top, menuRect.right, rect.bottom, x, y)
        || (x >= menuRect.left && x <= rect.right && y >= rect.top - 8 && y <= rect.bottom + 8);
    } else {
      return pointInTriangle(x, y, menuRect.left, rect.top, menuRect.left, rect.bottom, x, y)
        || (x >= rect.left && x <= menuRect.right && y >= rect.top - 8 && y <= rect.bottom + 8);
    }
  }, []);

  const handleItemMouseEnter = useCallback((index: number) => {
    if (safeTriangleTimer.current) {
      clearTimeout(safeTriangleTimer.current);
      safeTriangleTimer.current = null;
    }

    if (openSubMenuRef.current !== null && openSubMenuRef.current !== index) {
      const meta = metasRef.current[index];
      if (!meta?.isSubMenu && isInsideSafeTriangle()) {
        pendingHover.current = index;
        safeTriangleTimer.current = setTimeout(() => {
          safeTriangleTimer.current = null;
          const pending = pendingHover.current;
          pendingHover.current = null;
          if (pending !== null) {
            setFocusIndex(pending);
            const m = metasRef.current[pending];
            if (m?.isSubMenu) {
              setOpenSubMenuIndex(pending);
            } else {
              setOpenSubMenuIndex(null);
            }
          }
        }, 300);
        return;
      }
    }

    pendingHover.current = null;
    setFocusIndex(index);
    const meta = metasRef.current[index];
    if (meta?.isSubMenu) {
      setOpenSubMenuIndex(index);
    } else {
      setOpenSubMenuIndex(null);
    }
  }, [isInsideSafeTriangle]);

  const handleKeyboard = useCallback((e: KeyboardEvent) => {
    if (openSubMenuRef.current !== null) {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        setOpenSubMenuIndex(null);
      }
      return;
    }

    const count = itemCountRef.current;
    const skip = skipSetRef.current;

    const activeEl = document.activeElement;
    if (activeEl?.tagName === "INPUT" && menuContainerRef.current?.contains(activeEl as Node)) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        e.stopImmediatePropagation();
        focusViaKeyboard.current = true;
        setFocusIndex((i) => nextIndex(i, count, 1, skip));
      } else if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        onClose?.();
      }
      return;
    }

    switch (e.key) {
      case "ArrowDown": {
        e.preventDefault();
        e.stopImmediatePropagation();
        focusViaKeyboard.current = true;
        setFocusIndex((i) => nextIndex(i, count, 1, skip));
        break;
      }
      case "ArrowUp": {
        e.preventDefault();
        e.stopImmediatePropagation();
        focusViaKeyboard.current = true;
        const next = nextIndex(focusIndex, count, -1, skip);
        if (next > focusIndex || focusIndex <= 0) {
          const input = menuContainerRef.current?.querySelector("input") as HTMLElement;
          if (input) { input.focus(); setFocusIndex(-1); break; }
        }
        setFocusIndex(next);
        break;
      }
      case "Home": {
        e.preventDefault();
        e.stopImmediatePropagation();
        setFocusIndex(nextIndex(-1, count, 1, skip));
        break;
      }
      case "End": {
        e.preventDefault();
        e.stopImmediatePropagation();
        setFocusIndex(nextIndex(-1, count, -1, skip));
        break;
      }
      case " ":
      case "Enter": {
        e.preventDefault();
        e.stopImmediatePropagation();
        activateItem(focusIndex);
        break;
      }
      case "ArrowRight": {
        const meta = metasRef.current[focusIndex];
        if (meta?.isSubMenu) {
          e.preventDefault();
          e.stopImmediatePropagation();
          setOpenSubMenuIndex(focusIndex);
        }
        break;
      }
      case "ArrowLeft": {
        if (onParentArrowLeft) {
          e.preventDefault();
          e.stopImmediatePropagation();
          onParentArrowLeft();
        }
        break;
      }
      case "Escape": {
        e.preventDefault();
        e.stopImmediatePropagation();
        onClose?.();
        break;
      }
      default: {
        if (e.key.length === 1 && !e.metaKey && !e.ctrlKey && !e.altKey) {
          const idx = matchShortcut(e.key, shortcutMapRef.current);
          if (idx >= 0) {
            e.preventDefault();
            e.stopImmediatePropagation();
            setFocusIndex(idx);
            activateItem(idx);
          }
        }
      }
    }
  }, [focusIndex, onClose, onParentArrowLeft, activateItem]);

  const ALPHA = "abcdefghijklmnopqrstuvwxyz0123456789";

  useKeyboardHandler({
    scope: "menu",
    priority: "overlay",
    handler: handleKeyboard,
    shortcuts: [
      ...["ArrowDown", "ArrowUp", "ArrowLeft", "ArrowRight", "Home", "End", "Enter", " ", "Escape"].map((key) => ({
        id: `menu.${key}`,
        key,
        label: key,
        showInHelp: false,
        allowInEditable: true,
        preventDefault: false,
      })),
      ...ALPHA.split("").map((key) => ({
        id: `menu.char.${key}`,
        key,
        label: key,
        showInHelp: false,
        allowInEditable: true,
        preventDefault: false,
      })),
    ],
  });

  let childIndex = 0;
  const rendered = Children.map(children, (child) => {
    if (!isValidElement(child)) return child;
    const i = childIndex++;
    return cloneElement(child as ReactElement<MenuItemInternalProps>, {
      _menuIndex: i,
      _focused: focusIndex === i,
      _onMouseEnter: handleItemMouseEnter,
      _onClick: activateItem,
      _subMenuOpen: openSubMenuIndex === i,
      _onSubMenuClose: closeSubMenu,
    });
  });

  useEffect(() => {
    if (focusIndex < 0) return;
    const container = menuContainerRef.current;
    if (!container) return;
    if (focusViaKeyboard.current) {
      focusViaKeyboard.current = false;
      const active = document.activeElement;
      if (active && active.tagName === "INPUT" && container.contains(active)) {
        (active as HTMLElement).blur();
      }
    }
    const el = container.querySelector(`[data-menu-index="${focusIndex}"]`) as HTMLElement;
    if (el) el.scrollIntoView({ block: "nearest" });
  }, [focusIndex]);

  return (
    <div
      ref={menuContainerRef}
      role="menu"
      aria-label={ariaLabel}
      tabIndex={-1}
      style={maxHeight ? { maxHeight } : undefined}
      className={cn(
        "outline-none",
        !bare && "min-w-64 p-2 bg-[var(--color-surface-2)] border border-[var(--color-border-elevated)] rounded-[var(--radius-xl)] shadow-[var(--shadow-popover)]",
        maxHeight && "overflow-y-auto scrollbar-gutter-both px-0 scrollbar-thin",
        className,
      )}
    >
      {rendered}
    </div>
  );
}

// ─── MenuItem ──────────────────────────────────────────

interface MenuItemProps extends MenuItemInternalProps {
  label: string;
  icon?: ReactNode;
  shortcut?: string;
  disabled?: boolean;
  checked?: boolean;
  active?: boolean;
  destructive?: boolean;
  onClick?: () => void;
  suffix?: ReactNode;
}

export function MenuItem({
  label,
  icon,
  shortcut,
  disabled,
  checked,
  active,
  destructive,
  onClick,
  suffix,
  _menuIndex = 0,
  _focused,
  _onMouseEnter,
  _onClick,
}: MenuItemProps) {
  const isCheckbox = checked !== undefined;

  const handleClick = () => {
    if (disabled) return;
    if (_onClick) {
      _onClick(_menuIndex);
    } else {
      onClick?.();
    }
  };

  const handleMouseEnter = () => {
    _onMouseEnter?.(_menuIndex);
  };

  return (
    <button
      role={isCheckbox ? "menuitemcheckbox" : "menuitem"}
      aria-checked={isCheckbox ? checked : undefined}
      aria-disabled={disabled || undefined}
      data-menu-index={_menuIndex}
      tabIndex={_focused ? 0 : -1}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      className={cn(
        "flex items-center gap-2 w-full px-2 py-1.5 text-sm transition-colors rounded-md outline-none",
        _focused
          ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]"
          : active
            ? "bg-[var(--color-hover-surface-4)] text-[var(--color-text-primary)]"
            : destructive
              ? "text-[var(--color-error)]"
              : "text-[var(--color-text-secondary)]",
        disabled && "opacity-50 cursor-default",
        !disabled && !_focused && !active && "hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)]",
      )}
    >
      {icon && (
        <span className="shrink-0 flex items-center justify-center">
          {icon}
        </span>
      )}
      <span className="flex-1 text-left min-w-0">{label}</span>
      <span className="ml-auto w-4 flex items-center justify-center shrink-0">
        {isCheckbox && checked ? (
          <svg className="w-4 h-4 shrink-0 text-[var(--color-accent-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        ) : shortcut ? (
          <span className="text-xs text-[var(--color-text-muted)]">{shortcut}</span>
        ) : suffix ? (
          suffix
        ) : null}
      </span>
    </button>
  );
}

// ─── MenuDivider ───────────────────────────────────────

export function MenuDivider({ _menuIndex }: MenuItemInternalProps) {
  return (
    <div
      role="separator"
      data-menu-index={_menuIndex}
      className="my-0.5 border-t border-[var(--color-border-subtle)]"
    />
  );
}

// ─── MenuLabel ─────────────────────────────────────────

interface MenuLabelProps extends MenuItemInternalProps {
  children: ReactNode;
  className?: string;
}

export function MenuLabel({ children, className }: MenuLabelProps) {
  return (
    <div
      role="presentation"
      className={cn(
        "px-2 py-1 text-[11px] font-medium uppercase tracking-wider text-[var(--color-text-muted)]",
        className,
      )}
    >
      {children}
    </div>
  );
}

// ─── MenuFilter ───────────────────────────────────────

interface MenuFilterProps extends MenuItemInternalProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function MenuFilter({ value, onChange, placeholder, _menuIndex }: MenuFilterProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const raf = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(raf);
  }, []);

  return (
    <>
      <div data-menu-index={_menuIndex} className="px-2 py-1.5">
        <input
          ref={inputRef}
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder={placeholder}
          className="w-full text-sm bg-transparent text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
      </div>
      <div className="my-0.5 border-t border-[var(--color-border-subtle)]" />
    </>
  );
}

// ─── SubMenuTrigger (used internally by SubMenu) ──────

interface SubMenuTriggerButtonProps {
  label: string;
  icon?: ReactNode;
  shortcut?: string;
  disabled?: boolean;
  suffix?: ReactNode;
  open?: boolean;
  focused?: boolean;
  menuIndex?: number;
  onMouseEnter?: () => void;
  onClick?: () => void;
}

export function SubMenuTrigger({
  label,
  icon,
  shortcut,
  disabled,
  suffix,
  open,
  focused,
  menuIndex,
  onMouseEnter,
  onClick,
}: SubMenuTriggerButtonProps) {
  return (
    <button
      role="menuitem"
      aria-haspopup="menu"
      aria-expanded={open || undefined}
      aria-disabled={disabled || undefined}
      data-menu-index={menuIndex}
      tabIndex={focused ? 0 : -1}
      onClick={onClick}
      onMouseEnter={onMouseEnter}
      className={cn(
        "flex items-center gap-2 w-full px-2 py-1.5 text-sm transition-colors rounded-md outline-none",
        focused
          ? "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]"
          : open
            ? "bg-[var(--color-hover-surface-4)] text-[var(--color-text-primary)]"
            : "text-[var(--color-text-secondary)]",
        disabled && "opacity-50 cursor-default",
        !disabled && !focused && !open && "hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)]",
      )}
    >
      {icon && (
        <span className="w-4 shrink-0 flex items-center justify-center">
          {icon}
        </span>
      )}
      <span className="flex-1 text-left">{label}</span>
      {shortcut && (
        <span className="text-xs text-[var(--color-text-muted)]">
          {shortcut}
        </span>
      )}
      {suffix}
      <ChevronRight className="w-3 h-3 ml-auto text-[var(--color-text-muted)]" />
    </button>
  );
}
