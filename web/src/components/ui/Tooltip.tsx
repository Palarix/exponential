import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import type { ReactNode } from "react";

export default function Tooltip({
  children,
  content,
}: {
  children: ReactNode;
  content: string;
}) {
  if (!content) return <>{children}</>;
  return (
    <TooltipPrimitive.Root delayDuration={0}>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side="top"
          sideOffset={4}
          className="z-50 px-2 py-1 text-xs leading-snug rounded-[var(--radius-sm)] bg-[var(--color-surface-3)] border border-[var(--color-border-default)] text-[var(--color-text-secondary)] shadow-[var(--shadow-sm)] animate-in fade-in-0 zoom-in-95"
        >
          {content}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}
