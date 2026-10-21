import { cn } from "../../utils/cn";

/**
 * The trigger of a bordered dropdown field: a light wash of the text colour
 * with a tint border, so it shows on any surface. A required field without a
 * value gets a red border instead.
 */
export function fieldTriggerClass(invalid: boolean): string {
  return cn(
    "border bg-[var(--color-field-rest)] hover:bg-[var(--color-field-hover)]",
    invalid ? "border-[var(--color-error)]/40" : "border-[var(--color-tint-border)]",
  );
}
