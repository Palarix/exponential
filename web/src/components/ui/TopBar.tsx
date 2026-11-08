import type { ReactNode } from 'react';
import { cn } from '../../utils/cn';

interface TopBarProps {
  left?: ReactNode;
  center?: ReactNode;
  right?: ReactNode;
  className?: string;
}

/**
 * The view header: left, an optional centered slot (usually the search), and
 * right. With a center slot, the sides keep their content width and the center
 * shrinks first, so the slots never overlap. Without one, the sides can
 * shrink and truncate.
 */
export function TopBar({ left, center, right, className }: TopBarProps) {
  const side = center ? "shrink-0" : "min-w-0";
  return (
    <div className={cn(
      "flex items-center shrink-0 h-12 pl-5 pr-3",
      "border-b border-[var(--color-border-subtle)]",
      className
    )}>
      <div className={cn("flex-1 flex items-center", side)} data-slot="left">{left}</div>
      {center ? (
        <div className="flex items-center justify-center px-4 w-full max-w-md min-w-0" data-slot="center">{center}</div>
      ) : null}
      <div className={cn("flex-1 flex items-center justify-end", side)} data-slot="right">{right}</div>
    </div>
  );
}
