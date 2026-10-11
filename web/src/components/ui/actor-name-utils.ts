import { cn } from '../../utils/cn';
import { agentDisplayName, agentTooltip, shortName } from '../../utils/format';

// The dashed underline hints that the name hides more: an agent acted for
// this person. The muted decoration colour keeps it subtle in both themes.
const ON_BEHALF =
  'underline decoration-dashed decoration-1 underline-offset-[3px] decoration-[var(--color-text-muted)] cursor-help rounded-[var(--radius-sm)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-border-focus)]';

export interface ActorNameDisplay {
  label: string;
  tooltip: string;
  srText: string;
  className: string;
  focusable: boolean;
}

// name is the principal; via is the agent that acted for them, either a raw
// identity ("claude-code/2.1.294 <agent@mcp>") or a display name.
// focusable: false when the name sits inside an interactive control that
// already takes focus (the sr-only text then joins that control's name).
export function actorNameDisplay(
  name: string,
  via?: string,
  className?: string,
  { focusable = true }: { focusable?: boolean } = {},
): ActorNameDisplay {
  const label = shortName(name);
  if (!via) {
    return { label, tooltip: '', srText: '', className: cn(className), focusable: false };
  }
  return {
    label,
    tooltip: agentTooltip(via, label),
    srText: ` (via ${agentDisplayName(via)})`,
    className: cn(ON_BEHALF, className),
    focusable,
  };
}
