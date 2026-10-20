import { useRef, useState, type ReactNode } from "react";
import { ChevronDown, ExternalLink, Hammer, LogOut, MessageSquare } from "lucide-react";
import type { AgentStatus } from "../../api/chat";
import { Menu, MenuItem, MenuLabel, MenuDivider, Popover, PopoverPanel } from "../ui";
import { composerOptions, pickModel, type ChatPrefs } from "./agent-state-utils";
import ProviderIcon from "./ProviderIcon";

const PROVIDER_LOCKED = "A chat stays with the agent it started with. Start a new chat to switch.";
const CHAT_MODE = "Ask the agent to start an issue, or use Start Work on an issue.";

/** The issue a chat in Work Mode works on. */
export interface WorkModeInfo {
  issueId: string;
  title?: string;
}

function Rule() {
  return <span data-segment-rule aria-hidden className="w-px h-3.5 shrink-0 bg-[var(--color-border-ai)]" />;
}

/** A compact trigger that opens a menu below it. */
function Segment({
  label,
  icon,
  ariaLabel,
  title,
  disabled,
  className,
  children,
}: {
  label: string;
  icon?: ReactNode;
  ariaLabel: string;
  title?: string;
  disabled?: boolean;
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
        title={title}
        disabled={disabled}
        onClick={() => setOpen(!open)}
        className={`flex items-center gap-1 h-7 px-1.5 min-w-0 rounded-[var(--radius-md)] text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-2)] disabled:opacity-50 disabled:pointer-events-none transition-colors ${className ?? ""}`}
      >
        {icon}
        <span className="truncate">{label}</span>
        <ChevronDown className="w-3 h-3 shrink-0 text-[var(--color-text-muted)]" />
      </button>
      {open && !disabled && (
        <Popover anchorRef={ref} onClose={close}>
          <PopoverPanel className="surface-ai min-w-48">{children(close)}</PopoverPanel>
        </Popover>
      )}
    </>
  );
}

/**
 * The mode: a plain "Chat" label, or the hammer and "<issue>" with a menu to open
 * the issue or leave Work Mode. Nothing here enters Work Mode: that starts
 * from Start Work or from asking the agent.
 */
function ModeSegment({ work, running, onLeaveWork, onOpenIssue }: { work?: WorkModeInfo; running?: boolean; onLeaveWork?(): void; onOpenIssue?(id: string): void }) {
  if (!work) {
    return (
      <span title={CHAT_MODE} className="flex items-center gap-1 h-7 px-1.5 shrink-0 text-xs text-[var(--color-text-muted)]">
        <MessageSquare className="w-3 h-3" />
        <span>Chat</span>
      </span>
    );
  }
  return (
    <Segment
      label={work.issueId}
      icon={<Hammer className="w-3 h-3 shrink-0" />}
      ariaLabel={`Work Mode on ${work.issueId}`}
      title={work.title ? `${work.issueId} ${work.title}` : work.issueId}
      disabled={running}
      className="shrink-0"
    >
      {(close) => (
        <Menu onClose={close} aria-label="Mode" bare>
          <MenuItem label="Go to issue" icon={<ExternalLink size={14} />} onClick={() => { close(); onOpenIssue?.(work.issueId); }} />
          <MenuItem label="Exit work mode" icon={<LogOut size={14} />} onClick={() => { close(); onLeaveWork?.(); }} />
        </Menu>
      )}
    </Segment>
  );
}

/** The model and effort segments: the composer's, and the Preferences page's default model. */
export function ModelPickers({
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
    <>
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
    </>
  );
}

/** The composer footer's model, effort and mode segments. */
export default function ComposerPickers({
  status,
  prefs,
  providerLocked,
  onPrefsChange,
  work,
  running,
  onLeaveWork,
  onOpenIssue,
}: {
  status: AgentStatus | undefined;
  prefs: ChatPrefs;
  /** The chat has started: its session belongs to the picked provider. */
  providerLocked?: boolean;
  onPrefsChange(prefs: ChatPrefs): void;
  /** Set in Work Mode. */
  work?: WorkModeInfo;
  /** A turn runs: the mode cannot change. */
  running?: boolean;
  onLeaveWork?(): void;
  onOpenIssue?(issueId: string): void;
}) {
  return (
    <div className="flex items-center gap-1 min-w-0">
      <ModelPickers status={status} prefs={prefs} providerLocked={providerLocked} onPrefsChange={onPrefsChange} />
      <Rule />
      <ModeSegment work={work} running={running} onLeaveWork={onLeaveWork} onOpenIssue={onOpenIssue} />
    </div>
  );
}
