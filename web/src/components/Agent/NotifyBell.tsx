import { Bell, BellOff } from "lucide-react";
import type { NotifyState } from "./agent-state-utils";

const TITLES: Record<Exclude<NotifyState, "unsupported">, string> = {
  on: "Desktop notifications on: you're notified when an agent needs your input while this tab is hidden",
  off: "Turn on desktop notifications for when an agent needs your input",
  blocked: "Desktop notifications are blocked in this browser's site settings",
};

/** Turns desktop notifications on and off; hidden when the browser has none. */
export default function NotifyBell({ state, onToggle }: { state: NotifyState; onToggle: () => void }) {
  if (state === "unsupported") return null;
  const Icon = state === "on" ? Bell : BellOff;
  return (
    <button
      type="button"
      aria-label="Desktop notifications"
      aria-pressed={state === "on"}
      title={TITLES[state]}
      disabled={state === "blocked"}
      onClick={onToggle}
      className="shrink-0 p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)] disabled:opacity-50 disabled:cursor-default disabled:hover:bg-transparent disabled:hover:text-[var(--color-text-muted)]"
    >
      <Icon className="w-4 h-4" />
    </button>
  );
}
