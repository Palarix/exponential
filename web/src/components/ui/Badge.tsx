import { createContext, useContext, type ReactNode } from "react";

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
  default: "bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)]",
  success: "bg-[var(--color-success-bg)] text-[var(--color-success)]",
  warning: "bg-[var(--color-warning-bg)] text-[var(--color-warning)]",
  error: "bg-[var(--color-error-bg)] text-[var(--color-error)]",
  info: "bg-[var(--color-info-bg)] text-[var(--color-info)]",
  backlog: "bg-[var(--color-bg-tertiary)] text-[var(--color-status-backlog)]",
  planned: "bg-[var(--color-bg-tertiary)] text-[var(--color-status-planned)]",
  doing: "bg-[var(--color-warning-bg)] text-[var(--color-status-doing)]",
  blocked: "bg-[var(--color-error-bg)] text-[var(--color-status-blocked)]",
  done: "bg-[var(--color-success-bg)] text-[var(--color-status-done)]",
};

const sizeStyles: Record<string, string> = {
  sm: "px-1.5 py-0.5 text-xs",
  md: "px-2 py-0.5 text-xs",
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
        <span className="w-1.5 h-1.5 rounded-full bg-current opacity-80" />
      )}
      {children}
    </span>
  );
}

export const LabelColorsContext = createContext<Record<string, string>>({});
export const HideDefaultLabelsContext = createContext<boolean>(false);
export const DefaultLabelsContext = createContext<{ name: string; color: string }[]>([]);

export function LabelBadge({ label }: { label: string }) {
  const configColors = useContext(LabelColorsContext);
  const color =
    configColors[label] ||
    configColors[label.toLowerCase()] ||
    "var(--color-text-muted)";
  const displayLabel =
    label === label.toLowerCase()
      ? label.charAt(0).toUpperCase() + label.slice(1)
      : label;

  return (
    <span className="inline-flex items-center gap-1.5 text-xs font-medium text-text-secondary border rounded-2xl h-6 px-2 border-[var(--color-border-default)]">
      <span
        className="w-2 h-2 rounded-full shrink-0"
        style={{ background: color }}
      />
      {displayLabel}
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
