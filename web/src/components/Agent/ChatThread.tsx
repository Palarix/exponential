import { Fragment, useEffect, useRef, useState } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { ChevronRight, Copy, Check, X } from "lucide-react";
import { IssueMarkdown } from "../IssueRef/IssueMarkdown";
import { IssueRefText } from "../IssueRef/IssueRef";
import ApprovalCard from "./ApprovalCard";
import { approvalOutcomeLabel, collapsedFailureNote, failedSteps, failureReason, isFailed, isRunning, modeDividerText, pendingApprovals, splitTurn, stepSummary, turnSeconds, type ChatStep, type ToolStep, type ChatThread as Thread, type ChatTurn } from "./chat-thread-utils";

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />;
}

function useNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [active]);
  return now;
}

function AnswerText({ text }: { text: string }) {
  return (
    <div className="prose-exponential prose-activity">
      <IssueMarkdown>{text}</IssueMarkdown>
    </div>
  );
}

/**
 * A failed call's reason after its row: the first line in red, cut to the
 * row. The full message opens in a card on hover, on focus, or on click.
 */
function FailureReason({ step }: { step: ToolStep }) {
  const [open, setOpen] = useState(false);
  const { label, text, more } = failureReason(step);
  const full = step.message?.trim();
  // A fade, not an ellipsis: Safari shows its own tooltip for ellipsis-truncated
  // text, which would stack on the card.
  const reason = (
    <span className="min-w-0 flex-1 overflow-hidden whitespace-nowrap text-left text-[var(--color-error)] [mask-image:linear-gradient(to_right,black_calc(100%-1.5rem),transparent)]">
      {label}
      {text && `: ${text}`}
      {more && " …"}
    </span>
  );
  if (!full) return reason;
  return (
    <TooltipPrimitive.Root open={open} onOpenChange={setOpen} delayDuration={150}>
      <TooltipPrimitive.Trigger asChild>
        <button type="button" onClick={() => setOpen(true)} className="min-w-0 flex-1 flex rounded-[var(--radius-sm)] focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--color-border-focus)]">
          {reason}
        </button>
      </TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side="bottom"
          align="start"
          sideOffset={6}
          collisionPadding={12}
          className="z-50 max-w-md max-h-64 overflow-auto px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-surface-3)] border border-[var(--color-border-overlay)] shadow-[var(--shadow-md)] text-xs leading-relaxed text-[var(--color-text-secondary)] whitespace-pre-wrap break-words animate-in fade-in-0 zoom-in-95"
        >
          <div className="mb-1 font-medium text-[var(--color-error)]">{label === "denied" ? "Denied" : "Failed"}</div>
          {full}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}

/** A work-list row. A failed call carries its reason on the row. */
function WorkStep({ step }: { step: ChatStep }) {
  if (step.kind === "text") {
    return <div className="text-xs text-[var(--color-text-muted)] whitespace-pre-wrap">{step.text}</div>;
  }
  const { text, detail } = stepSummary(step);
  // Search patterns read best as code; file names and titles as text.
  const code = step.name === "Grep" || step.name === "Glob";
  // A call the user denied shows its approval outcome, not a failure.
  const failed = isFailed(step);
  return (
    <div className="flex items-center gap-1.5 text-xs min-w-0" title={failed ? undefined : step.arg || undefined}>
      {/* The slot keeps its width so the rows stay aligned. */}
      <span className="w-3 shrink-0 flex justify-center">
        {failed ? (
          <X className="w-3 h-3 text-[var(--color-error)]" aria-label={step.status === "denied" ? "Denied" : "Failed"} />
        ) : step.done ? (
          <span className="text-[var(--color-text-muted)]">·</span>
        ) : (
          <Spinner />
        )}
      </span>
      <span className="text-[var(--color-text-secondary)] shrink-0">{text}</span>
      {/* The detail gives way to a failure reason, but keeps a readable minimum. */}
      {detail && (
        <span className={`min-w-0 truncate text-[var(--color-text-muted)] ${failed ? "shrink-0 max-w-[40%]" : ""} ${code ? "font-mono" : ""}`}>
          <IssueRefText text={detail} />
        </span>
      )}
      {step.approval?.decision && (
        <span className="shrink-0 px-1.5 rounded-[var(--radius-sm)] bg-[var(--color-surface-3)] text-[11px] text-[var(--color-text-muted)]">
          {approvalOutcomeLabel(step.approval.decision)}
        </span>
      )}
      {step.approval && !step.approval.decision && <span className="shrink-0 text-[11px] text-[var(--color-warning)]">Waiting for approval</span>}
      {failed && <FailureReason step={step} />}
    </div>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label="Copy answer"
      onClick={() => {
        navigator.clipboard?.writeText(text).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        }).catch(() => {});
      }}
      className="p-1 -ml-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors"
    >
      {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
    </button>
  );
}

function Turn({ turn, chatId, onApprovalExpired }: { turn: ChatTurn; chatId?: string | null; onApprovalExpired?: (id: string) => void }) {
  // A turn with a failed call opens its work list, also when the failure
  // arrives mid-turn, until the user toggles it.
  const [toggled, setToggled] = useState<boolean | null>(null);
  const running = turn.status === "running";
  const now = useNow(running);
  const { work, answer } = splitTurn(turn);
  const answerText = answer.map((s) => (s.kind === "text" ? s.text : "")).join("\n\n").trim();
  const seconds = turnSeconds(turn, now);
  const expanded = toggled ?? failedSteps(turn).length > 0;
  const failureNote = collapsedFailureNote(turn, expanded);
  const showWork = work.length > 0 || (running && answer.length === 0);
  // Open cards stay visible while the work list is collapsed.
  const pending = chatId ? pendingApprovals(turn) : [];

  return (
    <div className="flex flex-col gap-3">
      <div className="self-end max-w-[85%] px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-surface-3)] text-sm text-[var(--color-text-primary)] whitespace-pre-wrap break-words">
        <IssueRefText text={turn.user.text} />
      </div>

      {showWork && (
        <div>
          <button
            type="button"
            onClick={() => setToggled(!expanded)}
            aria-expanded={work.length > 0 ? expanded : undefined}
            disabled={work.length === 0}
            className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] disabled:hover:text-[var(--color-text-muted)] transition-colors"
          >
            {running && pending.length > 0 ? (
              <span className="w-3 flex justify-center" aria-hidden>
                <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)]" />
              </span>
            ) : (
              running && <Spinner />
            )}
            <span>
              {running && pending.length > 0
                ? "Waiting for your approval"
                : running
                  ? `${turn.thinking ? "Thinking" : "Working"} for ${seconds}s`
                  : `Worked for ${seconds} second${seconds === 1 ? "" : "s"}`}
            </span>
            {failureNote && <span className="text-[var(--color-error)]">· {failureNote}</span>}
            {work.length > 0 && <ChevronRight className={`w-3 h-3 transition-transform ${expanded ? "rotate-90" : ""}`} />}
          </button>
          {expanded && (
            <div className="mt-2 ml-1 pl-3 border-l border-[var(--color-border-subtle)] flex flex-col gap-1.5">
              {work.map((step, i) => <WorkStep key={i} step={step} />)}
            </div>
          )}
        </div>
      )}

      {chatId && pending.map((step) => <ApprovalCard key={step.approval.id} chatId={chatId} step={step} onExpired={onApprovalExpired} />)}

      {answerText && (
        <div className="flex flex-col gap-1">
          <AnswerText text={answerText} />
          {!running && <div><CopyButton text={answerText} /></div>}
        </div>
      )}

      {turn.status === "stopped" && <div className="text-xs text-[var(--color-text-muted)]">Stopped</div>}
      {turn.status === "interrupted" && (
        <div className="text-xs text-[var(--color-text-muted)]">Interrupted. Send a message to continue.</div>
      )}
      {turn.status === "error" && (
        <div className="px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-error-bg)] text-xs text-[var(--color-error)] whitespace-pre-wrap break-words">
          {turn.error}
        </div>
      )}
    </div>
  );
}

/** A line across the thread where the chat entered or left Work Mode. */
/**
 * A labeled horizontal rule in the thread. Mode changes are solid; a dashed
 * rule marks something that is not a mode change, such as a cold cache.
 */
function Divider({ text, title, dashed }: { text: string; title?: string; dashed?: boolean }) {
  const line = dashed
    ? "flex-1 border-t border-dashed border-[var(--color-border-default)]"
    : "h-px flex-1 bg-[var(--color-border-subtle)]";
  return (
    <div role="separator" title={title} className="flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
      <span className={line} />
      <span className="shrink min-w-0 text-center">
        <IssueRefText text={text} />
      </span>
      <span className={line} />
    </div>
  );
}

/**
 * Whether the time at has passed. It re-renders when that happens, so a
 * marker can appear while the user watches.
 */
function usePassed(at: string | undefined): boolean {
  const target = at ? Date.parse(at) : NaN;
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (Number.isNaN(target) || now >= target) return;
    const timer = setTimeout(() => setNow(Date.now()), Math.max(0, target - Date.now()) + 50);
    return () => clearTimeout(timer);
  }, [target, now]);
  return !Number.isNaN(target) && now >= target;
}

function dayLabel(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const time = d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  return d.toDateString() === today.toDateString() ? `Today ${time}` : `${d.toLocaleDateString()} ${time}`;
}

/**
 * `wide` centers the thread in a reading column, for the full-page Agent view.
 * Approval cards need the chat's ID to send decisions.
 */
export default function ChatThread({
  thread,
  wide,
  chatId,
  onApprovalExpired,
  cacheExpiresAt,
}: {
  thread: Thread;
  wide?: boolean;
  chatId?: string | null;
  onApprovalExpired?: (id: string) => void;
  /** When the prompt cache of the last finished turn expires. */
  cacheExpiresAt?: string;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const stickRef = useRef(true);
  const cold = usePassed(cacheExpiresAt) && thread.turns.length > 0 && !isRunning(thread);

  // Follow new output, and the cold cache marker, while the user is at the bottom.
  useEffect(() => {
    const el = scrollRef.current;
    if (el && stickRef.current) el.scrollTop = el.scrollHeight;
  }, [thread, cold]);

  const first = thread.turns[0];
  return (
    <div
      ref={scrollRef}
      onScroll={(e) => {
        const el = e.currentTarget;
        stickRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
      }}
      className={`flex-1 overflow-y-auto px-4 ${wide ? "py-8" : "py-3"}`}
    >
      {/* The composer's box: its max-w-3xl column minus its px-4, so the thread and the reply box share both edges. */}
      <div className={wide ? "mx-auto max-w-[46rem]" : undefined}>
      {first && <div className="text-center text-xs text-[var(--color-text-muted)] mb-4">{dayLabel(first.user.time)}</div>}
      <div className="flex flex-col gap-6">
        {thread.turns.map((turn, i) => (
          <Fragment key={i}>
            {thread.dividers.filter((d) => d.at === i).map((d, j) => <Divider key={`d${j}`} text={modeDividerText(d)} />)}
            <Turn turn={turn} chatId={chatId} onApprovalExpired={onApprovalExpired} />
          </Fragment>
        ))}
        {thread.dividers.filter((d) => d.at >= thread.turns.length).map((d, j) => <Divider key={`end${j}`} text={modeDividerText(d)} />)}
        {cold && <Divider dashed text="Context cache expired" title="Your next message re-sends the whole conversation." />}
      </div>
      </div>
    </div>
  );
}
