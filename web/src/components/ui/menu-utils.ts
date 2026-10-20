import { cn } from "../../utils/cn";

export function nextIndex(
  current: number,
  count: number,
  direction: 1 | -1,
  skip: Set<number>,
): number {
  if (count === 0) return -1;

  if (current === -1) {
    const start = direction === 1 ? 0 : count - 1;
    if (!skip.has(start)) return start;
    current = start;
  }

  for (let i = 1; i < count; i++) {
    const candidate = ((current + direction * i) % count + count) % count;
    if (!skip.has(candidate)) return candidate;
  }
  return -1;
}

export function matchShortcut(
  key: string,
  shortcuts: Map<number, string>,
): number {
  const lower = key.toLowerCase();
  let result = -1;
  shortcuts.forEach((shortcut, index) => {
    if (result === -1 && shortcut.toLowerCase() === lower) result = index;
  });
  return result;
}

/**
 * Check if point (px,py) is inside the triangle formed by three vertices.
 * Uses barycentric coordinate method.
 */
export function pointInTriangle(
  px: number, py: number,
  x1: number, y1: number,
  x2: number, y2: number,
  x3: number, y3: number,
): boolean {
  const d = (x2 - x1) * (y3 - y1) - (x3 - x1) * (y2 - y1);
  if (d === 0) return false;
  const a = ((x2 - px) * (y3 - py) - (x3 - px) * (y2 - py)) / d;
  const b = ((x3 - px) * (y1 - py) - (x1 - px) * (y3 - py)) / d;
  const c = 1 - a - b;
  return a >= 0 && b >= 0 && c >= 0;
}

/**
 * Whether activating an item closes its menu. Checkbox items stay open so
 * several can be toggled; keepOpen items open a next step inside the menu.
 */
export function closesOnActivate({ isCheckbox, keepOpen }: { isCheckbox: boolean; keepOpen?: boolean }): boolean {
  return !isCheckbox && !keepOpen;
}

/** Whether hovering an item opens it as a submenu: only an enabled submenu does (xpo-a71a45). */
export function opensOnHover({ isSubMenu, navigable }: { isSubMenu: boolean; navigable: boolean }): boolean {
  return isSubMenu && navigable;
}

/** The inset a framed menu keeps on every side, in px (p-2). */
const MENU_INSET = 8;

/**
 * Padding for a framed menu that scrolls. Its stable scrollbar gutter, gutter
 * px wide on each edge, already insets the sides; the browser decides how wide
 * (0 with overlay scrollbars). Pad so every side ends up with the same inset.
 */
export function scrollMenuPadding(gutter: number): { block: number; inline: number } {
  return { block: Math.max(MENU_INSET, gutter), inline: Math.max(0, MENU_INSET - gutter) };
}

/** A MenuItem's classes. Destructive items stay red when highlighted, on a red tint. */
export function menuItemClass({ focused, active, destructive, checked, disabled }: {
  focused: boolean; active: boolean; destructive: boolean; checked: boolean; disabled: boolean;
}): string {
  const highlight = destructive
    ? "bg-[var(--color-error-bg)] text-[var(--color-error)]"
    : "bg-[var(--color-hover-surface-3)] text-[var(--color-text-primary)]";
  return cn(
    "flex items-center gap-2 w-full px-2 py-1.5 text-sm transition-colors rounded-md outline-none",
    focused
      ? highlight
      : active
        ? "bg-[var(--color-hover-surface-4)] text-[var(--color-text-primary)]"
        : destructive
          ? "text-[var(--color-error)]"
          : checked
            ? "text-[var(--color-text-primary)]"
            : "text-[var(--color-text-secondary)]",
    disabled && "opacity-50 cursor-default",
    !disabled && !focused && !active && (destructive
      ? "hover:bg-[var(--color-error-bg)]"
      : "hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)]"),
  );
}
