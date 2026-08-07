import { createContext, useContext, type ReactNode } from "react";
import { labelColor as resolveLabelColor } from "../../utils/labels";

type BadgeVariant =
  | "default"
  | "success"
  | "warning"
  | "error"
  | "info"
  | "backlog"
  | "planned"
  | "doing"
  | "blocked"
  | "done";

interface BadgeProps {
  children: ReactNode;
  variant?: BadgeVariant;
  size?: "sm" | "md";
  dot?: boolean;
}

const variantStyles: Record<BadgeVariant, string> = {
  default: "bg-[var(--color-surface-1)] text-[var(--color-text-secondary)]",
  success: "bg-[var(--color-success-bg)] text-[var(--color-success)]",
  warning: "bg-[var(--color-warning-bg)] text-[var(--color-warning)]",
  error: "bg-[var(--color-error-bg)] text-[var(--color-error)]",
  info: "bg-[var(--color-info-bg)] text-[var(--color-info)]",
  backlog: "bg-[var(--color-surface-1)] text-[var(--color-status-backlog)]",
  planned: "bg-[var(--color-surface-1)] text-[var(--color-status-planned)]",
  doing: "bg-[var(--color-warning-bg)] text-[var(--color-status-doing)]",
  blocked: "bg-[var(--color-error-bg)] text-[var(--color-status-blocked)]",
  done: "bg-[var(--color-success-bg)] text-[var(--color-status-done)]",
};

const sizeStyles: Record<string, string> = {
  sm: "px-2 py-1 text-xs",
  md: "px-2 py-1 text-xs",
};

export default function Badge({
  children,
  variant = "default",
  size = "md",
  dot = false,
}: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center gap-1 font-medium
        rounded-[var(--radius-sm)]
        ${variantStyles[variant]}
        ${sizeStyles[size]}
      `
        .trim()
        .replace(/\s+/g, " ")}
    >
      {dot && (
        <span className="w-2 h-2 rounded-full bg-current opacity-80" />
      )}
      {children}
    </span>
  );
}

export const LabelColorsContext = createContext<Record<string, string>>({});
export const HideDefaultLabelsContext = createContext<boolean>(false);
export const DefaultLabelsContext = createContext<{ name: string; color: string }[]>([]);

export function LabelBadge({ label, borderless }: { label: string; borderless?: boolean }) {
  const configColors = useContext(LabelColorsContext);
  const color = resolveLabelColor(label, configColors);
  const displayLabel =
    label === label.toLowerCase()
      ? label.charAt(0).toUpperCase() + label.slice(1)
      : label;

  return (
    <span className={`inline-flex items-center gap-2 text-xs font-medium text-text-secondary rounded-2xl h-6 px-2 ${borderless ? "" : "border border-[var(--color-border-label)]"}`}>
      <span
        className="w-2 h-2 rounded-full shrink-0"
        style={{ background: color }}
      />
      {displayLabel}
    </span>
  );
}

export function LabelIndicator({ labels }: { labels: string[] }) {
  const configColors = useContext(LabelColorsContext);
  if (labels.length === 0) return null;
  return (
    <span className="inline-flex items-center gap-0.5 shrink-0">
      {labels.map((label) => (
        <span
          key={label}
          className="w-1.5 h-4 rounded-sm"
          style={{ background: resolveLabelColor(label, configColors) }}
          title={label}
        />
      ))}
    </span>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const variant = status.toLowerCase() as BadgeVariant;
  return (
    <Badge
      variant={
        ["backlog", "planned", "doing", "blocked", "done"].includes(variant)
          ? variant
          : "default"
      }
      dot
    >
      {status}
    </Badge>
  );
}
