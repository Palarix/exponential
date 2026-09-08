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
      {left}
      {center ? (
        <div className="flex-1 flex justify-center min-w-0 px-4">{center}</div>
      ) : (
        <div className="flex-1" />
      )}
      {right}
    </div>
  );
}
