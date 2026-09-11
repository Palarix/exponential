import type { ReactNode } from 'react';
import { cn } from '../../utils/cn';

interface TopBarProps {
  left?: ReactNode;
  center?: ReactNode;
  right?: ReactNode;
  className?: string;
}

export function TopBar({ left, center, right, className }: TopBarProps) {
  return (
    <div className={cn(
      "flex items-center shrink-0 h-12 pl-5 pr-3",
      "border-b border-[var(--color-border-subtle)]",
      className
    )}>
      <div className="flex-1 flex items-center min-w-0">{left}</div>
      {center ? (
        <div className="flex items-center justify-center px-4 w-full max-w-md">{center}</div>
      ) : null}
      <div className="flex-1 flex items-center justify-end min-w-0">{right}</div>
    </div>
  );
}
