import { useRef, useState, type ReactNode } from "react";
import { ChevronDown, Lock } from "lucide-react";
import type { AgentStatus } from "../../api/chat";
import { Menu, MenuItem, MenuLabel, MenuDivider, Popover, PopoverPanel } from "../ui";
import { composerOptions, pickModel, type ChatPrefs } from "./agent-state-utils";
import ProviderIcon from "./ProviderIcon";

const PROVIDER_LOCKED = "A chat stays with the agent it started with. Start a new chat to switch.";
const READ_ONLY = "Chat can read code and change issues. Code changes happen in work sessions.";

function Rule() {
  return <span data-segment-rule aria-hidden className="w-px h-3.5 shrink-0 bg-[var(--color-border-default)]" />;
}

/** A compact trigger that opens a menu below it. */
function Segment({
  label,
  icon,
  ariaLabel,
  className,
  children,
}: {
  label: string;
  icon?: ReactNode;
  ariaLabel: string;
  className?: string;
  children(close: () => void): ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLButtonElement>(null);
  const close = () => setOpen(false);
  return (
    <>
      <button
        ref={ref}
        type="button"
        aria-label={ariaLabel}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={`flex items-center gap-1 h-7 px-1.5 min-w-0 rounded-[var(--radius-md)] text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors ${className ?? ""}`}
      >
        {icon}
        <span className="truncate">{label}</span>
        <ChevronDown className="w-3 h-3 shrink-0 text-[var(--color-text-muted)]" />
      </button>
      {open && (
        <Popover anchorRef={ref} onClose={close}>
          <PopoverPanel className="min-w-48">{children(close)}</PopoverPanel>
        </Popover>
      )}
    </>
  );
}

/** The composer footer's model, effort and permission segments. */
export default function ComposerPickers({
  status,
  prefs,
  providerLocked,
  onPrefsChange,
}: {
  status: AgentStatus | undefined;
  prefs: ChatPrefs;
  /** The chat has started: its session belongs to the picked provider. */
  providerLocked?: boolean;
  onPrefsChange(prefs: ChatPrefs): void;
}) {
  const options = composerOptions(status, prefs, !!providerLocked);
  const headings = options.groups.length > 1;

  return (
    <div className="flex items-center gap-1 min-w-0">
      <Segment
        label={options.modelLabel}
        icon={<ProviderIcon provider={prefs.provider} />}
        ariaLabel="Model"
        className="max-w-48 shrink"
      >
        {(close) => (
          <Menu onClose={close} aria-label="Model" bare maxHeight="20rem">
            {options.groups.flatMap((group, i) => [
              ...(headings && i > 0 ? [<MenuDivider key={`${group.provider}-rule`} />] : []),
              ...(headings ? [<MenuLabel key={`${group.provider}-label`}>{group.label}</MenuLabel>] : []),
              ...group.options.map((opt) => (
                <MenuItem
                  key={opt.value}
                  label={opt.label}
                  icon={<ProviderIcon provider={group.provider} />}
                  radio
                  checked={opt.value === options.model}
                  disabled={group.disabled}
                  truncate
                  onClick={() => { onPrefsChange(pickModel(prefs, opt.value, status)); close(); }}
                />
              )),
            ])}
            {options.groups.some((g) => g.disabled) && (
              <MenuLabel className="normal-case tracking-normal font-normal max-w-60">{PROVIDER_LOCKED}</MenuLabel>
            )}
          </Menu>
        )}
      </Segment>
      <Rule />
      <Segment label={options.effortLabel} ariaLabel="Effort" className="shrink-0">
        {(close) => (
          <Menu onClose={close} aria-label="Effort" bare>
            {options.efforts.map((opt) => (
              <MenuItem
                key={opt.value}
                label={opt.label}
                radio
                checked={opt.value === prefs.effort}
                onClick={() => { onPrefsChange({ ...prefs, effort: opt.value }); close(); }}
              />
            ))}
          </Menu>
        )}
      </Segment>
      <Rule />
      <span title={READ_ONLY} className="flex items-center gap-1 h-7 px-1.5 shrink-0 text-xs text-[var(--color-text-muted)]">
        <Lock className="w-3 h-3" />
        <span>Read-only</span>
      </span>
    </div>
  );
}
