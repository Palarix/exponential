import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { decideApproval, type ApprovalDecision } from "../../api/chat";
import { ApiError } from "../../api/client";
import { IssueRefText } from "../IssueRef/IssueRef";
import { Menu, MenuItem, Popover, PopoverPanel } from "../ui";
import { approvalArgs, approvalKeyDecision, approvalTitle, toolLabel, typing, type StepApproval, type ToolStep } from "./chat-thread-utils";

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-current border-t-transparent rounded-full animate-spin" />;
}

/** A button's label. While sending, the spinner covers the hidden label, which keeps the button's size. */
export function ButtonLabel({ text, sending }: { text: string; sending: boolean }) {
  return (
    <span className="relative inline-grid place-items-center">
      <span className={sending ? "invisible" : undefined}>{text}</span>
      {sending && (
        <span className="absolute inset-0 flex items-center justify-center">
          <Spinner />
        </span>
      )}
    </span>
  );
}

// Buttons share the composer's control size.
const BUTTON = "flex items-center justify-center gap-1.5 h-7 text-xs font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none";
const DENY = `${BUTTON} px-2.5 rounded-[var(--radius-md)] text-[var(--color-text-primary)] bg-[var(--color-tint-rest)] hover:bg-[var(--color-tint-hover)] active:bg-[var(--color-tint-active)]`;
const PRIMARY = `${BUTTON} bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)] active:bg-[var(--color-button-ai-active)]`;

/**
 * An agent asks before a tool call runs. The card stays until the decision
 * arrives on the chat's event stream topic. onExpired runs when the server no longer has
 * the request open. The active card takes focus, unless the user is typing:
 * there Enter allows and Esc denies, as in a form.
 */
export default function ApprovalCard({
  chatId,
  step,
  active = false,
  onExpired,
  onFocusChange,
  defaultOpen = false,
}: {
  chatId: string;
  step: ToolStep & { approval: StepApproval };
  active?: boolean;
  onExpired?: (approvalId: string) => void;
  /** Runs with true when focus enters the card and false when it leaves. */
  onFocusChange?: (focused: boolean) => void;
  /** Whether a call's details start expanded. */
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const [sending, setSending] = useState<ApprovalDecision | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const chevronRef = useRef<HTMLButtonElement>(null);

  const decide = (decision: ApprovalDecision) => {
    setSending(decision);
    setError(null);
    decideApproval(chatId, step.approval.id, decision)
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.status === 404) {
          onExpired?.(step.approval.id);
          return;
        }
        setError(err instanceof Error ? err.message : "Could not send the decision");
      })
      .finally(() => setSending(null));
  };

  const cardRef = useRef<HTMLDivElement>(null);
  // The active card takes focus, but never from a field the user is typing in.
  useEffect(() => {
    if (active && !typing(document.activeElement)) cardRef.current?.focus({ preventScroll: true });
  }, [active]);

  // Enter and Esc answer for the card itself; a focused button keeps its own Enter.
  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const decision = approvalKeyDecision({
      key: e.key,
      onCard: e.target === e.currentTarget,
      composing: e.nativeEvent.isComposing,
      modified: e.metaKey || e.ctrlKey || e.altKey || e.shiftKey,
    });
    if (!decision) return;
    e.preventDefault();
    e.stopPropagation();
    if (sending === null) decide(decision);
  };

  const { action, detail } = approvalTitle(step.name, step.arg);
  const { command, fields } = approvalArgs(step.name, step.approval.input);
  const hasArgs = !!command || fields.length > 0;
  const busy = sending !== null;

  const expanded = hasArgs && open;

  const title = (
    <>
      {action}
      {detail && (
        <>
          {" "}
          <IssueRefText text={detail} />
        </>
      )}{" "}
      <span className="whitespace-nowrap text-[var(--color-text-muted)]">
        via <span className="font-mono text-[0.85em] px-1.5 py-px rounded-[var(--radius-sm)] bg-[var(--color-tint-rest)]">{toolLabel(step.name)}</span>
      </span>
    </>
  );
  const headlineClass = `min-w-0 text-sm text-[var(--color-text-primary)] break-words ${expanded ? "" : "flex-[1_1_auto] py-1"}`;
  // A call with details toggles them from its headline.
  const headline = hasArgs ? (
    <button type="button" aria-expanded={open} onClick={() => setOpen(!open)} className={`${headlineClass} flex items-center gap-1 text-left`}>
      <span className="min-w-0">{title}</span>
      <ChevronRight className={`shrink-0 w-3.5 h-3.5 text-[var(--color-text-muted)] transition-transform ${open ? "rotate-90" : ""}`} />
    </button>
  ) : (
    <div className={headlineClass}>{title}</div>
  );

  // Deny, and Allow with the chat-wide grant behind its chevron.
  const buttons = (
    <div className="ml-auto flex shrink-0 items-center gap-2">
      <button type="button" disabled={busy} onClick={() => decide("deny")} className={DENY}>
        <ButtonLabel text="Deny" sending={sending === "deny"} />
      </button>
      <div className="flex items-center">
        <button type="button" disabled={busy} onClick={() => decide("allow")} className={`${PRIMARY} pl-3.5 pr-3 rounded-l-[var(--radius-md)]`}>
          <ButtonLabel text="Allow" sending={sending === "allow" || sending === "allow_session"} />
        </button>
        <button
          ref={chevronRef}
          type="button"
          disabled={busy}
          aria-label="More ways to allow"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen(!menuOpen)}
          className={`${PRIMARY} px-1.5 rounded-r-[var(--radius-md)] border-l border-white/20`}
        >
          <ChevronDown className="w-3.5 h-3.5" />
        </button>
        {menuOpen && !busy && (
          <Popover anchorRef={chevronRef} onClose={() => setMenuOpen(false)} placement="bottom-end">
            <PopoverPanel className="surface-ai min-w-48">
              <Menu onClose={() => setMenuOpen(false)} aria-label="More ways to allow" bare>
                <MenuItem
                  label="Allow for this chat"
                  onClick={() => {
                    setMenuOpen(false);
                    decide("allow_session");
                  }}
                />
              </Menu>
            </PopoverPanel>
          </Popover>
        )}
      </div>
    </div>
  );

  return (
    <div
      ref={cardRef}
      role="group"
      aria-label="Approval needed"
      tabIndex={0}
      onKeyDown={onKeyDown}
      onFocus={() => onFocusChange?.(true)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget)) onFocusChange?.(false);
      }}
      className="rounded-[var(--radius-lg)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-2)] outline-none focus-within:border-[var(--color-border-focus)] transition-colors"
    >
      {expanded ? (
        <>
          {/* The headline and the call on top, the buttons in a footer. */}
          <div className="flex flex-col gap-2.5 px-3 pt-3 pb-2.5 min-w-0">
            {headline}
            <div className="max-h-40 overflow-auto px-2.5 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-1)] border border-[var(--color-border-ai-subtle)] font-mono text-xs">
              {command && <pre className="whitespace-pre-wrap break-words text-[var(--color-text-primary)]">{command}</pre>}
              {fields.length > 0 && (
                <dl className={`grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 min-w-0 ${command ? "mt-1.5" : ""}`}>
                  {fields.map(([key, value]) => (
                    <div key={key} className="contents">
                      <dt className="text-[var(--color-text-muted)]">{key}</dt>
                      <dd className="min-w-0 text-[var(--color-text-primary)] whitespace-pre-wrap break-words line-clamp-3" title={value}>
                        {value}
                      </dd>
                    </div>
                  ))}
                </dl>
              )}
            </div>
            {error && <div className="text-xs text-[var(--color-error)] break-words">{error}</div>}
          </div>
          <div className="flex px-3 pb-3">{buttons}</div>
        </>
      ) : (
        <>
          {/* Collapsed, or nothing to show but the headline: it shares a row with the buttons, which wrap below when they do not fit. */}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 px-3 py-2.5 min-w-0">
            {headline}
            {buttons}
          </div>
          {error && <div className="px-3 pb-2.5 text-xs text-[var(--color-error)] break-words">{error}</div>}
        </>
      )}
    </div>
  );
}