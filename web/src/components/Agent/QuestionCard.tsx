import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Check, CircleQuestionMark } from "lucide-react";
import { answerQuestion, decideApproval } from "../../api/chat";
import { ApiError } from "../../api/client";
import {
  autoAdvanceTo,
  emptyPicks,
  parseQuestions,
  pickOption,
  questionAdvance,
  questionAnswered,
  questionAnswerText,
  questionAnswers,
  questionKeyAction,
  questionTabLabel,
  reviewStep,
  setOtherText,
  stepInClass,
  toggleOther,
  type Question,
  type QuestionPick,
  type StepApproval,
  type StepDirection,
  type ToolStep,
} from "./chat-thread-utils";

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-current border-t-transparent rounded-full animate-spin" />;
}

// Buttons match the approval card's.
const BUTTON = "flex items-center justify-center gap-1.5 h-7 px-2.5 rounded-[var(--radius-md)] text-xs font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none";
const QUIET = "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-2)]";
// The approval card's secondary style, with the hover of the footer's ai-1 band.
const SECONDARY = "border border-[var(--color-border-ai)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]";
const SEND = "bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)]";

// How long a single-select pick stays on screen before the next question shows.
const AUTO_ADVANCE_MS = 250;

/**
 * A raised radio or checkbox row: the mark, the label with the description
 * under it, and the key that picks it, centered on each other. The row is a
 * container around its button, so children (the "Other…" field) share its
 * border and tint; an input cannot sit inside a button.
 */
export function Choice({
  multi,
  checked,
  label,
  description,
  hint,
  disabled,
  onClick,
  children,
}: {
  multi: boolean;
  checked: boolean;
  label: string;
  description?: string;
  hint?: number;
  disabled: boolean;
  onClick: () => void;
  children?: ReactNode;
}) {
  return (
    <div
      // The border and tint fade in with the mark's pop.
      className={`rounded-[var(--radius-md)] border transition-colors duration-[var(--duration-normal)] ${
        checked ? "border-[var(--color-button-ai)] bg-[var(--color-surface-ai-selected)]" : "border-[var(--color-border-ai)] bg-[var(--color-surface-ai-3)] hover:bg-[var(--color-hover-surface-ai-3)]"
      }`}
    >
      <button
        type="button"
        role={multi ? "checkbox" : "radio"}
        aria-checked={checked}
        disabled={disabled}
        onClick={onClick}
        className="w-full flex items-center gap-2.5 px-2.5 py-2 text-left disabled:opacity-50"
      >
        <span
          aria-hidden
          className={`w-3.5 h-3.5 shrink-0 flex items-center justify-center border transition-colors duration-[var(--duration-normal)] ${multi ? "rounded-[3px]" : "rounded-full"} ${
            checked ? "border-[var(--color-button-ai)] bg-[var(--color-button-ai)] text-white" : "border-[var(--color-border-ai)]"
          }`}
        >
          {/* The pick lands: the dot or tick pops in as it mounts. */}
          {checked && (multi ? <Check className="w-2.5 h-2.5 motion-safe:animate-pop-in" strokeWidth={3} /> : <span className="w-1.5 h-1.5 rounded-full bg-white motion-safe:animate-pop-in" />)}
        </span>
        <span className="flex-1 min-w-0">
          <span className="block text-sm text-[var(--color-text-primary)] break-words">{label}</span>
          {description && <span className="block mt-0.5 text-xs text-[var(--color-text-muted)] break-words">{description}</span>}
        </span>
        {hint !== undefined && (
          <kbd aria-hidden className="shrink-0 min-w-[1.125rem] h-[1.125rem] px-1 flex items-center justify-center rounded-[var(--radius-sm)] border border-[var(--color-border-ai)] font-mono text-[10px] text-[var(--color-text-muted)]">
            {hint}
          </kbd>
        )}
      </button>
      {/* Indented to the label: the row's padding, the mark and the gap. */}
      {children && <div className="pl-8.5 pr-2.5 pb-2">{children}</div>}
    </div>
  );
}

function Tab({ selected, label, title, disabled, onClick, className = "", children }: { selected: boolean; label: string; title: string; disabled: boolean; onClick: () => void; className?: string; children: ReactNode }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      aria-label={label}
      title={title}
      disabled={disabled}
      onClick={onClick}
      // The band's 12px of vertical padding sits on the tab, so the band's whole height is clickable.
      className={`${className} flex items-center gap-1 max-w-[10rem] px-2.5 py-3 text-xs transition-colors disabled:opacity-50 ${
        selected ? "text-[var(--color-text-primary)]" : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
      }`}
    >
      {children}
    </button>
  );
}

/**
 * One tab per question, then a Review tab, in a recessed band. The current
 * tab has primary text; an answered question's tab shows a check.
 */
export function QuestionTabs({ questions, picks, current, disabled, onSelect }: { questions: Question[]; picks: QuestionPick[]; current: number; disabled: boolean; onSelect: (i: number) => void }) {
  const review = questions.length;
  return (
    <div role="tablist" aria-label="Questions" className="flex flex-wrap items-center px-1.5 bg-[var(--color-surface-ai-1)] border-b border-[var(--color-border-ai-subtle)]">
      {questions.map((q, i) => {
        const label = questionTabLabel(q, i);
        const answered = questionAnswered(q, picks[i]);
        const selected = i === current;
        return (
          <Tab key={i} selected={selected} label={`${label}, ${answered ? "answered" : selected ? "current" : "not answered"}`} title={label} disabled={disabled} onClick={() => onSelect(i)}>
            {/* Pops in as the question gets its answer. */}
            {answered && <Check aria-hidden className="w-3 h-3 shrink-0 motion-safe:animate-pop-in" strokeWidth={2.5} />}
            <span className="truncate">{label}</span>
          </Tab>
        );
      })}
      {/* At the far right: the destination, apart from the questions. */}
      <Tab selected={current === review} label="Review" title="Review your answers" disabled={disabled} onClick={() => onSelect(review)} className="ml-auto">
        <span className="truncate">Review</span>
      </Tab>
    </div>
  );
}

/**
 * The review step: each full question over its answer, or "Not answered".
 * The rows read as results, not options: no border, no raised surface, no
 * radio mark; the answer carries the ✓ of an answered tab. Clickable rows
 * sent people round in circles, so only "Change" on hover (or focus) goes
 * back to the question.
 */
export function QuestionReview({ questions, picks, disabled, onSelect }: { questions: Question[]; picks: QuestionPick[]; disabled: boolean; onSelect: (i: number) => void }) {
  return (
    // The rows sit inside the body's padding: their edges line up with the
    // option rows and the footer buttons.
    <div className="flex flex-col gap-0.5 min-w-0">
      {questions.map((q, i) => {
        const answer = questionAnswerText(q, picks[i]);
        return (
          <div
            key={i}
            className="group flex items-center gap-2.5 px-2.5 py-2 rounded-[var(--radius-md)] hover:bg-[var(--color-hover-surface-ai-2)] focus-within:bg-[var(--color-hover-surface-ai-2)] transition-colors"
          >
            <div className="flex-1 min-w-0">
              <div className="text-sm text-[var(--color-text-secondary)] break-words">{q.question}</div>
              {/* A status mark in one slot: ✓ for an answer, a muted ? for an open question. */}
              <div className="mt-1 flex items-start gap-1.5">
                <span aria-hidden className="w-3.5 h-5 shrink-0 flex items-center">
                  {answer ? (
                    <Check className="w-3.5 h-3.5 text-[var(--color-button-ai)]" strokeWidth={2.5} />
                  ) : (
                    <CircleQuestionMark className="w-3.5 h-3.5 text-[var(--color-text-muted)]" />
                  )}
                </span>
                {answer ? (
                  <span className="min-w-0 text-sm text-[var(--color-text-primary)] break-words">{answer}</span>
                ) : (
                  <span className="text-sm italic text-[var(--color-text-muted)]">Not answered</span>
                )}
              </div>
            </div>
            <button
              type="button"
              aria-label={`Change answer to: ${q.question}`}
              disabled={disabled}
              onClick={() => onSelect(i)}
              className={`${BUTTON} ${QUIET} shrink-0 opacity-0 group-hover:opacity-100 focus-visible:opacity-100`}
            >
              Change
            </button>
          </div>
        );
      })}
    </div>
  );
}

/**
 * "Chat about this" at the far left, away from the primary button (Next or
 * "Submit answers") at the right end, so the right side only moves forward.
 */
export function QuestionFooter({
  primary,
  primaryDisabled,
  sending,
  error,
  onChat,
  onPrimary,
}: {
  primary: "Next" | "Submit answers";
  primaryDisabled: boolean;
  sending: "answer" | "deny" | null;
  error: string | null;
  onChat: () => void;
  onPrimary: () => void;
}) {
  const busy = sending !== null;
  return (
    // The body's padding, so the buttons line up with the option rows.
    <div className="flex items-center gap-3 px-3 py-3 min-w-0 bg-[var(--color-surface-ai-1)] border-t border-[var(--color-border-ai-subtle)]">
      <button type="button" disabled={busy} onClick={onChat} className={`${BUTTON} ${SECONDARY} shrink-0`}>
        {sending === "deny" && <Spinner />}
        Chat about this
      </button>
      {error && <div className="min-w-0 text-xs text-[var(--color-error)] break-words">{error}</div>}
      <button type="button" disabled={busy || primaryDisabled} onClick={onPrimary} className={`${BUTTON} ${SEND} ml-auto shrink-0`}>
        {sending === "answer" && <Spinner />}
        {primary}
      </button>
    </div>
  );
}

/**
 * The agent asks the user questions (AskUserQuestion). Each question offers
 * its options, and "Other…" for a free-text answer. With several questions,
 * the card shows one at a time behind tabs, then a review step; the tabs are
 * the way back. "Submit answers" returns the answers; "Chat about this"
 * denies the call, so the user can talk instead. The card stays until the
 * outcome arrives on the chat stream. onExpired runs when the server no
 * longer has it open.
 */
export default function QuestionCard({
  chatId,
  step,
  onExpired,
}: {
  chatId: string;
  step: ToolStep & { approval: StepApproval };
  onExpired?: (approvalId: string) => void;
}) {
  const id = useId();
  const questions = useMemo(() => parseQuestions(step.approval.input), [step.approval.input]);
  const [picks, setPicks] = useState(() => emptyPicks(questions));
  const [current, setCurrent] = useState(0);
  // Which way the last move went, so the new step slides in from that side. Null until the first move.
  const [direction, setDirection] = useState<StepDirection>(null);
  const [sending, setSending] = useState<"answer" | "deny" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const answers = questionAnswers(questions, picks);
  const busy = sending !== null;
  const review = reviewStep(questions);
  const tabbed = review !== null;
  const onReview = current === review;
  // "Submit answers" sends: on the review step, or on the only question.
  const submits = onReview || !tabbed;

  const clearTimer = () => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
  };
  useEffect(() => clearTimer, []);

  // Focus goes back to the card, so its keys keep working and focus does not
  // stay on a button that the new question hides.
  const go = (i: number) => {
    clearTimer();
    if (i !== current) {
      setDirection(i > current ? "forward" : "back");
      setCurrent(i);
    }
    rootRef.current?.focus();
  };

  const send = (kind: "answer" | "deny") => {
    if (busy || (kind === "answer" && !answers)) return;
    clearTimer();
    setSending(kind);
    setError(null);
    const call = kind === "answer" && answers ? answerQuestion(chatId, step.approval.id, answers) : decideApproval(chatId, step.approval.id, "deny");
    call
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.status === 404) {
          onExpired?.(step.approval.id);
          return;
        }
        setError(err instanceof Error ? err.message : "Could not send the answer");
      })
      .finally(() => setSending(null));
  };

  const advance = () => {
    if (busy) return;
    const move = questionAdvance(questions, picks, current);
    if (move.kind === "send") send("answer");
    else go(move.index);
  };

  const pick = (label: string) => {
    setPicks(pickOption(picks, questions, current, label));
    clearTimer();
    const next = autoAdvanceTo(questions, current);
    if (next !== null) timer.current = setTimeout(() => go(next), AUTO_ADVANCE_MS);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (busy || e.target instanceof HTMLInputElement || e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
    const action = questionKeyAction(e.key, questions, current);
    if (!action) return;
    // The board's own ←/→ listeners sit on the document; keep the keys here.
    e.preventDefault();
    e.stopPropagation();
    if (action.kind === "pick") pick(action.label);
    else if (action.kind === "go") go(action.index);
  };

  // The free-text answer: inside the "Other…" row, or on its own for a
  // question without options.
  const otherField = (q: Question, value: string, picked: boolean) => (
    <input
      type="text"
      aria-label={`Your answer to: ${q.question}`}
      placeholder="Type your answer"
      value={value}
      disabled={busy}
      // The user just picked Other.
      autoFocus={picked}
      onChange={(e) => setPicks(setOtherText(picks, current, e.target.value))}
      onKeyDown={(e) => {
        if (e.key === "Enter" && !e.nativeEvent.isComposing) {
          e.preventDefault();
          advance();
        }
      }}
      className="w-full h-8 px-2.5 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-1)] border border-[var(--color-border-ai)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-border-focus)]"
    />
  );

  const q = questions[current];
  const p = picks[current];
  const labelId = `${id}-q${current}`;

  return (
    <div ref={rootRef} role="group" aria-label="Question" tabIndex={-1} onKeyDown={onKeyDown} className="overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-2)] focus:outline-none">
      {tabbed && <QuestionTabs questions={questions} picks={picks} current={current} disabled={busy} onSelect={go} />}
      {onReview && (
        <div role="tabpanel" aria-label="Review your answers" className={`px-3 pt-3 pb-3 ${stepInClass(direction)}`}>
          <QuestionReview questions={questions} picks={picks} disabled={busy} onSelect={go} />
        </div>
      )}
      {q && p && (
        // Keyed by step, so each step mounts fresh and its slide-in plays.
        <div key={current} role={tabbed ? "tabpanel" : undefined} aria-labelledby={tabbed ? labelId : undefined} className={`px-3 pt-3 pb-3 ${stepInClass(direction)}`}>
          <div role={q.multiSelect ? "group" : "radiogroup"} aria-labelledby={labelId} className="min-w-0">
            {/* The header is metadata at the line's right end; with tabs, the tab shows it. */}
            <div className="flex items-start gap-2 min-w-0">
              <span id={labelId} className="flex-1 min-w-0 text-sm font-semibold text-[var(--color-text-primary)] break-words">
                {q.question}
              </span>
              {q.header && !tabbed && (
                <span className="shrink-0 mt-px px-1.5 rounded-[var(--radius-sm)] bg-[var(--color-surface-ai-1)] border border-[var(--color-border-ai)] text-[11px] text-[var(--color-text-muted)]">
                  {q.header}
                </span>
              )}
            </div>
            {q.options.length > 0 ? (
              <div className="mt-3 flex flex-col gap-1.5">
                {q.options.map((o, i) => (
                  <Choice
                    key={o.label}
                    multi={q.multiSelect}
                    checked={p.picked.includes(o.label)}
                    label={o.label}
                    // Agents often repeat the label as its description.
                    description={o.description !== o.label ? o.description : undefined}
                    // The keys 1–9 pick the first nine options.
                    hint={i < 9 ? i + 1 : undefined}
                    disabled={busy}
                    onClick={() => pick(o.label)}
                  />
                ))}
                <Choice
                  multi={q.multiSelect}
                  checked={p.other !== null}
                  label="Other…"
                  disabled={busy}
                  onClick={() => {
                    clearTimer();
                    setPicks(toggleOther(picks, questions, current));
                  }}
                >
                  {p.other !== null && otherField(q, p.other, true)}
                </Choice>
              </div>
            ) : (
              <div className="mt-3">{otherField(q, p.other ?? "", false)}</div>
            )}
          </div>
        </div>
      )}
      <QuestionFooter
        primary={submits ? "Submit answers" : "Next"}
        primaryDisabled={submits && !answers}
        sending={sending}
        error={error}
        onChat={() => send("deny")}
        onPrimary={advance}
      />
    </div>
  );
}
