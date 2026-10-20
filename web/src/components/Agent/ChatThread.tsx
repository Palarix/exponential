import { Fragment, useEffect, useLayoutEffect, useRef, useState, type ComponentProps, type RefObject } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { Bot, ChevronRight, Copy, Check, CornerDownRight, FileText, Lightbulb, Pencil } from "lucide-react";
import type { ChatAttachment } from "../../api/chat";
import { useStickToBottom } from "../../hooks/useStickToBottom";
import { IssueMarkdown } from "../IssueRef/IssueMarkdown";
import { IssueRefText } from "../IssueRef/IssueRef";
import ApprovalCard from "./ApprovalCard";
import { attachmentUrl, formatSize, isAgentImage } from "./attachment-utils";
import QuestionCard from "./QuestionCard";
import ToolIcon from "./ToolIcon";
import Tooltip from "../ui/Tooltip";
import { activeCardId, approvalOutcomeLabel, cardsDone, userTurns, visibleTurns, failureReason, formatDuration, groupSummary, isFailed, isQuestion, isRunning, modeDividerText, parseQuestions, pendingApprovals, stepIcon, timelineSteps, turnSeconds, turnSegments, type ToolStep, type ChatThread as Thread, type ChatTurn } from "./chat-thread-utils";
import { scrollBehavior } from "../../preferences";
import { StreamIn, useStreamIn } from "./stream-in";

function useNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [active]);
  return now;
}

/** A div of the thread that fades in when it appears in a live thread. */
function Block({ className, ...props }: ComponentProps<"div">) {
  const fade = useStreamIn();
  return <div {...props} className={[className, fade].filter(Boolean).join(" ") || undefined} />;
}

function AnswerText({ text }: { text: string }) {
  return (
    <Block className="prose-exponential prose-activity">
      <IssueMarkdown>{text}</IssueMarkdown>
    </Block>
  );
}

/**
 * A failed call's reason after its row: the first line, cut to the row.
 * Only the row's icon is red: agents fail and recover often. The full
 * message opens in a card on hover, on focus, or on click.
 */
function FailureReason({ step }: { step: ToolStep }) {
  const [open, setOpen] = useState(false);
  const { label, text, more } = failureReason(step);
  const full = step.message?.trim();
  // A fade, not an ellipsis: Safari shows its own tooltip for ellipsis-truncated
  // text, which would stack on the card.
  const reason = (
    <span className="min-w-0 flex-1 overflow-hidden whitespace-nowrap text-left text-[var(--color-text-muted)] [mask-image:linear-gradient(to_right,black_calc(100%-1.5rem),transparent)]">
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
          className="z-50 max-w-md max-h-64 overflow-auto px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] border border-[var(--color-border-ai)] shadow-[var(--shadow-md)] text-xs leading-relaxed text-[var(--color-text-secondary)] whitespace-pre-wrap break-words animate-in fade-in-0 zoom-in-95"
        >
          <div className="mb-1 font-medium text-[var(--color-text-primary)]">{label === "denied" ? "Denied" : "Failed"}</div>
          {full}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}

// The timeline's rail runs through the middle of each row's first line.
const RAIL_Y = 11;

/** A group's calls as a timeline: each call branches off one continuous rail. */
export function ToolTimeline({ steps }: { steps: ToolStep[] }) {
  return (
    <ol className="mt-1.5 flex flex-col">
      {steps.map((step, i) => {
        // Each row draws its own piece of the rail, from the first row's branch to the last row's.
        const first = i === 0;
        const last = i === steps.length - 1;
        const rail = { top: first ? RAIL_Y : 0, ...(last ? { height: first ? 0 : RAIL_Y } : { bottom: 0 }) };
        return (
          <li key={step.id ?? i} data-step="tool" className="relative pl-5 py-[3px] min-w-0">
            <span className="absolute left-[5px] w-px bg-[var(--color-border-ai)]" style={rail} aria-hidden />
            <span className="absolute left-[6px] w-2.5 h-px bg-[var(--color-border-ai)]" style={{ top: RAIL_Y }} aria-hidden />
            <ToolRow step={step} />
          </li>
        );
      })}
    </ol>
  );
}

/** A tool call: its icon, its verb, then what it worked on. A failed call carries its reason on the row. */
function ToolRow({ step }: { step: ToolStep }) {
  const { icon, verb, name, detail } = stepIcon(step);
  // Search patterns read best as code; file names and titles as text.
  const code = step.name === "Grep" || step.name === "Glob";
  // A call the user denied shows its approval outcome, not a failure.
  const failed = isFailed(step);
  // A running call's text shimmers.
  const text = step.done ? "text-[var(--color-text-secondary)]" : "text-shimmer";
  return (
    <div className="flex items-center gap-1.5 text-xs min-w-0" title={failed ? undefined : step.arg || undefined}>
      {/* The slot keeps its width so the rows stay aligned. */}
      <span className="w-3.5 shrink-0 flex justify-center">
        <ToolIcon icon={icon} className={failed ? "text-[var(--color-error)]" : "text-[var(--color-text-muted)]"} />
      </span>
      {name && <span className={`shrink-0 ${text}`}>{name}</span>}
      {/* The verb stays whole; the detail truncates. */}
      {verb && <span className={`shrink-0 ${step.done ? "text-[var(--color-text-muted)]" : "text-shimmer"}`}>{verb}</span>}
      {/* The detail gives way to a failure reason, but keeps a readable minimum. */}
      {detail && (
        <span className={`min-w-0 truncate ${text} ${failed ? "shrink-0 max-w-[40%]" : ""} ${code ? "font-mono" : ""}`}>
          <IssueRefText text={detail} />
        </span>
      )}
      {step.approval?.decision && (
        <span className="shrink-0 px-1.5 rounded-[var(--radius-sm)] bg-[var(--color-surface-ai-2)] text-[11px] text-[var(--color-text-muted)]">
          {approvalOutcomeLabel(step.approval.decision, isQuestion(step))}
        </span>
      )}
      {step.approval && !step.approval.decision && <span className="shrink-0 text-[11px] text-[var(--color-warning)]">{isQuestion(step) ? "Waiting for an answer" : "Waiting for approval"}</span>}
      {failed && <FailureReason step={step} />}
    </div>
  );
}

/**
 * What the active group is doing: its latest call, as icon, verb and detail.
 * The verb stays progressive until the call moves into the list. The text shimmers while it works.
 */
function CurrentCall({ step }: { step: ToolStep }) {
  const { icon, verb, name, detail } = stepIcon(step, true);
  return (
    <>
      <ToolIcon icon={icon} />
      {(name || verb) && <span className="shrink-0 text-shimmer">{name || verb}</span>}
      {detail && <span className="min-w-0 truncate text-shimmer">{detail}</span>}
    </>
  );
}

/**
 * A group of tool calls on one line. The active group, the turn's latest,
 * shows its current call and the turn's timer; a finished one counts its
 * calls. Either expands into the timeline, which leaves out the call the
 * active line shows.
 */
function ToolGroup({ steps, active, waiting, seconds }: { steps: ToolStep[]; active: boolean; waiting: "approval" | "answer" | null; seconds: number }) {
  const [open, setOpen] = useState(false);
  const summary = groupSummary(steps);
  const listed = timelineSteps(steps, active);
  const expandable = listed.length > 0;
  return (
    <Block>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        disabled={!expandable}
        aria-expanded={expandable ? open : undefined}
        data-active-group={active || undefined}
        className="w-full flex items-center gap-1.5 min-w-0 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] disabled:hover:text-[var(--color-text-muted)] transition-colors"
      >
        {active && waiting ? (
          <>
            <span className="w-3 shrink-0 flex justify-center" aria-hidden>
              <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)]" />
            </span>
            <span className="min-w-0 truncate">Waiting for your {waiting}</span>
          </>
        ) : active ? (
          <CurrentCall step={steps[steps.length - 1]} />
        ) : (
          <>
            <span className="shrink-0">{summary.text}</span>
            {summary.failed && <span className="shrink-0 text-[var(--color-error)]">· {summary.failed}</span>}
          </>
        )}
        {expandable && <ChevronRight className={`w-3 h-3 shrink-0 transition-transform ${open ? "rotate-90" : ""}`} />}
        {active && <span className="ml-auto shrink-0 pl-2 tabular-nums">{formatDuration(seconds)}</span>}
      </button>
      {open && expandable && <ToolTimeline steps={listed} />}
    </Block>
  );
}

/** An answered question in the thread: each question, with the user's answer under it. */
function AnsweredQuestion({ step }: { step: ToolStep }) {
  const answers = step.approval?.answers ?? {};
  const asked = parseQuestions(step.approval?.input).map((q) => q.question);
  // A question the input no longer names still shows with its answer.
  const shown = asked.length > 0 ? asked : Object.keys(answers);
  return (
    <Block data-answered-question className="flex flex-col gap-2 pl-3 border-l-2 border-[var(--color-border-ai)] min-w-0">
      {shown.map((q) => (
        <div key={q} className="min-w-0">
          <div className="text-xs text-[var(--color-text-muted)] whitespace-pre-line break-words">{q}</div>
          <div className="text-sm text-[var(--color-text-primary)] break-words">{answers[q] ?? "No answer"}</div>
        </div>
      ))}
    </Block>
  );
}

/** While the agent thinks between groups, before its next message. */
function ThinkingLine({ seconds }: { seconds: number }) {
  return (
    <Block data-thinking className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)]">
      <Lightbulb className="w-3.5 h-3.5 shrink-0" aria-hidden />
      <span className="text-shimmer">Thinking</span>
      <span className="ml-auto shrink-0 pl-2 tabular-nums">{formatDuration(seconds)}</span>
    </Block>
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
      className="p-1 -ml-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)] transition-colors"
    >
      {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
    </button>
  );
}

/** A message's attachments: image thumbnails that open the full image, and file chips that download. */
function MessageAttachments({ chatId, attachments }: { chatId: string; attachments: ChatAttachment[] }) {
  return (
    <Block data-message-attachments className="self-end flex flex-wrap justify-end gap-1.5 max-w-[85%]">
      {attachments.map((a) => {
        const url = attachmentUrl(chatId, a.id);
        const name = a.name || "attachment";
        return isAgentImage(a.mime) ? (
          <a key={a.id} href={url} target="_blank" rel="noreferrer" title={name} className="block rounded-[var(--radius-md)] overflow-hidden border border-[var(--color-border-ai-subtle)] hover:border-[var(--color-border-ai)] transition-colors">
            <img src={url} alt={name} className="block h-20 max-w-48 object-cover" />
          </a>
        ) : (
          <a key={a.id} href={url} download={name} title={name} className="flex items-center gap-1.5 max-w-56 px-2 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] border border-[var(--color-border-ai-subtle)] hover:border-[var(--color-border-ai)] text-xs transition-colors">
            <FileText className="w-3.5 h-3.5 shrink-0 text-[var(--color-text-muted)]" aria-hidden />
            <span className="truncate text-[var(--color-text-primary)]">{name}</span>
            <span className="shrink-0 text-[var(--color-text-muted)]">{formatSize(a.size)}</span>
          </a>
        );
      })}
    </Block>
  );
}

export function Turn({
  index,
  turn,
  chatId,
  onApprovalExpired,
  onCardsDone,
  onEdit,
}: {
  index: number;
  turn: ChatTurn;
  chatId?: string | null;
  onApprovalExpired?: (id: string) => void;
  onCardsDone?: () => void;
  /** Set on the message the user can edit and send again. */
  onEdit?: () => void;
}) {
  const running = turn.status === "running";
  const now = useNow(running);
  const seconds = turnSeconds(turn, now);
  const segments = turnSegments(turn);
  const last = segments[segments.length - 1];
  // Open cards show under the active group.
  const pending = chatId ? pendingApprovals(turn) : [];
  const [focusedCard, setFocusedCard] = useState<string | null>(null);
  const active = activeCardId(pending, focusedCard);
  // The last card went away with focus in it: focus fell to the page, so hand it on.
  const pendingCount = useRef(pending.length);
  useEffect(() => {
    const before = pendingCount.current;
    pendingCount.current = pending.length;
    if (cardsDone(before, pending.length, !document.activeElement || document.activeElement === document.body)) onCardsDone?.();
  }, [pending.length, onCardsDone]);
  const waiting = pending.length === 0 ? null : pending.some(isQuestion) ? "answer" : "approval";
  // Streaming narration is progress enough; otherwise the agent is thinking,
  // as it is right after the user answers a question.
  const thinking = running && (!last || last.kind === "question" || last.kind === "steer" || (last.kind === "text" && turn.thinking));
  const lastText = !running && last?.kind === "text" ? last.text.trim() : "";

  return (
    <div data-turn={index} className="flex flex-col gap-3">
      {turn.byAgent && (
        <Block data-agent-turn className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] min-w-0">
          <Bot className="w-3.5 h-3.5 shrink-0" aria-hidden />
          <span className="truncate">{turn.user.text || "The agent continued on its own"}</span>
        </Block>
      )}
      {!turn.continued && !turn.byAgent && chatId && turn.user.attachments && (
        <MessageAttachments chatId={chatId} attachments={turn.user.attachments} />
      )}
      {!turn.continued && !turn.byAgent && (turn.user.text || !turn.user.attachments) && (
        <Block data-user-message className="self-end max-w-[85%] px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-surface-ai-2)] text-sm text-[var(--color-text-primary)] whitespace-pre-wrap break-words">
          <IssueRefText text={turn.user.text} />
        </Block>
      )}
      {onEdit && (
        <div className="self-end -mt-2">
          <Tooltip content="Edit and send again. Files the agent changed stay changed.">
            <button
              type="button"
              aria-label="Edit message"
              onClick={onEdit}
              className="flex items-center gap-1 px-1.5 py-0.5 rounded-[var(--radius-sm)] text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]"
            >
              <Pencil className="w-3 h-3" aria-hidden />
              Edit
            </button>
          </Tooltip>
        </div>
      )}

      {segments.map((seg, i) =>
        seg.kind === "text" ? (
          seg.text.trim() && <AnswerText key={i} text={seg.text} />
        ) : seg.kind === "question" ? (
          <AnsweredQuestion key={i} step={seg.step} />
        ) : seg.kind === "steer" ? (
          <Fragment key={i}>
            {chatId && seg.attachments && <MessageAttachments chatId={chatId} attachments={seg.attachments} />}
            {seg.text && (
              <Block data-steer-message className="self-end max-w-[85%] flex gap-1.5 px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-surface-ai-2)] text-sm text-[var(--color-text-primary)] whitespace-pre-wrap break-words">
                <CornerDownRight className="w-3.5 h-3.5 mt-0.5 shrink-0 text-[var(--color-text-muted)]" aria-label="Sent during the turn" />
                <span className="min-w-0">
                  <IssueRefText text={seg.text} />
                </span>
              </Block>
            )}
          </Fragment>
        ) : (
          <ToolGroup key={i} steps={seg.steps} active={running && seg === last} waiting={waiting} seconds={seconds} />
        ),
      )}
      {thinking && <ThinkingLine seconds={seconds} />}

      {chatId &&
        pending.map((step) => {
          const card = {
            chatId,
            step,
            active: step.approval.id === active,
            onExpired: onApprovalExpired,
            onFocusChange: (focused: boolean) => setFocusedCard((id) => (focused ? step.approval.id : id === step.approval.id ? null : id)),
          };
          return <Block key={step.approval.id}>{isQuestion(step) ? <QuestionCard {...card} /> : <ApprovalCard {...card} />}</Block>;
        })}

      {turn.status === "stopped" && <Block className="text-xs text-[var(--color-text-muted)]">Stopped</Block>}
      {turn.status === "interrupted" && (
        <Block className="text-xs text-[var(--color-text-muted)]">Interrupted. Send a message to continue.</Block>
      )}
      {turn.status === "error" && (
        <Block className="px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-error-bg)] text-xs text-[var(--color-error)] whitespace-pre-wrap break-words">
          {turn.error}
        </Block>
      )}
      {!running && turn.status !== "interrupted" && (
        <Block className="flex items-center min-h-[1.375rem] text-xs text-[var(--color-text-muted)]">
          {lastText && <CopyButton text={lastText} />}
          <span className="ml-auto tabular-nums" title="Time the agent worked on this turn">{formatDuration(seconds)}</span>
        </Block>
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
    ? "flex-1 border-t border-dashed border-[var(--color-border-ai)]"
    : "h-px flex-1 bg-[var(--color-border-ai-subtle)]";
  return (
    <Block role="separator" title={title} className="flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
      <span className={line} />
      <span className="shrink min-w-0 text-center">
        <IssueRefText text={text} />
      </span>
      <span className={line} />
    </Block>
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

// How far below the thread's top a turn lands when a bar scrolls to it.
const TURN_MARGIN = 16;

/** Where an element sits in the scroller's content, in pixels. */
function contentSpan(scroller: HTMLElement, el: Element | null): { top: number; bottom: number } {
  if (!el) return { top: 0, bottom: 0 };
  const rect = el.getBoundingClientRect();
  const top = rect.top - scroller.getBoundingClientRect().top + scroller.scrollTop;
  return { top, bottom: top + rect.height };
}

/**
 * One bar per user message on the thread's left edge. Hovering the rail
 * magnifies the bars, the hovered one most. A bar scrolls to its message;
 * a bar is highlighted while its message is in view.
 */
function TurnRail({ thread, scrollRef }: { thread: Thread; scrollRef: RefObject<HTMLDivElement | null> }) {
  const turns = userTurns(thread);
  const [shown, setShown] = useState<number[]>([]);
  const key = turns.join(",");

  useEffect(() => {
    const scroller = scrollRef.current;
    if (!scroller || !key) return;
    const indices = key.split(",").map(Number);
    let frame = 0;
    const update = () => {
      frame = 0;
      const spans = indices.map((i) => contentSpan(scroller, scroller.querySelector(`[data-turn="${i}"] [data-user-message]`)));
      setShown(visibleTurns(spans, scroller.scrollTop, scroller.scrollTop + scroller.clientHeight).map((t) => indices[t]));
    };
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(update);
    };
    update();
    scroller.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      scroller.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(frame);
    };
  }, [scrollRef, key]);

  if (turns.length < 2) return null;
  const jump = (turn: number) => {
    const scroller = scrollRef.current;
    if (scroller) scroller.scrollTo({ top: contentSpan(scroller, scroller.querySelector(`[data-turn="${turn}"]`)).top - TURN_MARGIN, behavior: scrollBehavior() });
  };
  // left-6 lines the bars up with the chat title in the Agent view's top bar (px-6).
  return (
    <nav aria-label="Turns" className="group absolute left-6 top-1/2 -translate-y-1/2 z-10 flex flex-col max-h-[calc(100%-4rem)] py-2">
      {turns.map((i) => {
        const text = thread.turns[i].user.text;
        const current = shown.includes(i);
        return (
          <TooltipPrimitive.Root key={i} delayDuration={0}>
            <TooltipPrimitive.Trigger asChild>
              <button
                type="button"
                data-turn-bar={i}
                data-current={current || undefined}
                aria-label={`Go to message: ${text}`}
                onClick={() => jump(i)}
                className="group/bar h-2.5 min-h-0 group-hover:h-3.5 group-has-[:focus-visible]:h-3.5 w-12 flex items-center rounded-[var(--radius-sm)] transition-[height] focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--color-border-focus)]"
              >
                <span
                  className={`h-0.5 w-3 rounded-full transition-all group-hover:h-[3px] group-has-[:focus-visible]:h-[3px] group-hover:w-6 group-has-[:focus-visible]:w-6 group-hover/bar:w-10 group-hover/bar:bg-[var(--color-text-primary)] group-focus-visible/bar:w-10 group-focus-visible/bar:bg-[var(--color-text-primary)] ${current ? "bg-[var(--color-text-muted)]" : "bg-[var(--color-border-ai)]"}`}
                />
              </button>
            </TooltipPrimitive.Trigger>
            <TooltipPrimitive.Portal>
              <TooltipPrimitive.Content
                side="right"
                sideOffset={4}
                collisionPadding={12}
                className="z-50 max-w-xs truncate px-2 py-1 text-xs leading-snug rounded-[var(--radius-sm)] bg-[var(--color-surface-ai-2)] border border-[var(--color-border-ai)] text-[var(--color-text-secondary)] shadow-[var(--shadow-sm)] animate-in fade-in-0 zoom-in-95"
              >
                {text}
              </TooltipPrimitive.Content>
            </TooltipPrimitive.Portal>
          </TooltipPrimitive.Root>
        );
      })}
    </nav>
  );
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
  onCardsDone,
  cacheExpiresAt,
  editTarget,
  onEditMessage,
}: {
  thread: Thread;
  wide?: boolean;
  chatId?: string | null;
  onApprovalExpired?: (id: string) => void;
  /** Runs when the last open card is answered and focus has nowhere to go. */
  onCardsDone?: () => void;
  /** When the prompt cache of the last finished turn expires. */
  cacheExpiresAt?: string;
  /** The turn whose user message the user can edit; null or undefined for none. */
  editTarget?: number | null;
  onEditMessage?: () => void;
}) {
  // Live once the commit that brought the history in has painted: from then
  // on, new blocks fade in and following glides. Until then, both jump.
  const live = useRef(false);
  useEffect(() => {
    live.current = thread.loaded;
  }, [thread.loaded]);
  const { ref: scrollRef, pin } = useStickToBottom<HTMLDivElement>(live);
  const cold = usePassed(cacheExpiresAt) && thread.turns.length > 0 && !isRunning(thread);

  // Follow new output, and the cold cache marker, while the user is at the bottom.
  useLayoutEffect(pin, [pin, thread, cold]);

  const first = thread.turns[0];
  return (
    <StreamIn.Provider value={live}>
    <div className="relative flex-1 min-h-0 flex flex-col">
    {wide && <TurnRail thread={thread} scrollRef={scrollRef} />}
    <div
      ref={scrollRef}
      data-thread-scroller
      className={`flex-1 scroll-stable px-4 ${wide ? "py-8" : "py-3"}`}
    >
      {/* The composer's box: its max-w-3xl column minus its px-4, so the thread and the reply box share both edges. */}
      <div className={wide ? "mx-auto max-w-[46rem]" : undefined}>
      {first && <div className="text-center text-xs text-[var(--color-text-muted)] mb-4">{dayLabel(first.user.time)}</div>}
      <div className="flex flex-col gap-6">
        {thread.turns.map((turn, i) => (
          <Fragment key={i}>
            {thread.dividers.filter((d) => d.at === i).map((d, j) => <Divider key={`d${j}`} text={modeDividerText(d)} />)}
            <Turn index={i} turn={turn} chatId={chatId} onApprovalExpired={onApprovalExpired} onCardsDone={onCardsDone} onEdit={i === editTarget ? onEditMessage : undefined} />
          </Fragment>
        ))}
        {thread.dividers.filter((d) => d.at >= thread.turns.length).map((d, j) => <Divider key={`end${j}`} text={modeDividerText(d)} />)}
        {cold && <Divider dashed text="Context cache expired" title="Your next message re-sends the whole conversation." />}
      </div>
      </div>
    </div>
    </div>
    </StreamIn.Provider>
  );
}
