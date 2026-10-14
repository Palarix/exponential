import { useId, useMemo, useState } from "react";
import { Check } from "lucide-react";
import { answerQuestion, decideApproval } from "../../api/chat";
import { ApiError } from "../../api/client";
import { emptyPicks, parseQuestions, pickOption, questionAnswers, setOtherText, toggleOther, type StepApproval, type ToolStep } from "./chat-thread-utils";

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-current border-t-transparent rounded-full animate-spin" />;
}

// Footer buttons match the approval card's.
const BUTTON = "flex items-center justify-center gap-1.5 h-7 px-2.5 rounded-[var(--radius-md)] text-xs font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none";
const SKIP = "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-2)]";
const SEND = "bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)]";

/** A radio or checkbox row: the mark, the label, and the description under it. */
function Choice({ multi, checked, label, description, disabled, onClick }: { multi: boolean; checked: boolean; label: string; description?: string; disabled: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      role={multi ? "checkbox" : "radio"}
      aria-checked={checked}
      disabled={disabled}
      onClick={onClick}
      className={`w-full flex items-start gap-2 px-2 py-1.5 rounded-[var(--radius-md)] border text-left transition-colors disabled:opacity-50 ${
        checked ? "border-[var(--color-border-ai)] bg-[var(--color-surface-ai-1)]" : "border-transparent hover:bg-[var(--color-hover-surface-ai-2)]"
      }`}
    >
      <span
        aria-hidden
        className={`mt-[3px] w-3.5 h-3.5 shrink-0 flex items-center justify-center border ${multi ? "rounded-[3px]" : "rounded-full"} ${
          checked ? "border-[var(--color-button-ai)] bg-[var(--color-button-ai)] text-white" : "border-[var(--color-border-ai)]"
        }`}
      >
        {checked && (multi ? <Check className="w-2.5 h-2.5" strokeWidth={3} /> : <span className="w-1.5 h-1.5 rounded-full bg-white" />)}
      </span>
      <span className="min-w-0">
        <span className="block text-sm text-[var(--color-text-primary)] break-words">{label}</span>
        {description && <span className="block text-xs text-[var(--color-text-muted)] break-words">{description}</span>}
      </span>
    </button>
  );
}

/**
 * The agent asks the user questions (AskUserQuestion). Each question offers
 * its options, and "Other…" for a free-text answer. Send returns the
 * answers; Skip denies the call. The card stays until the outcome arrives on
 * the chat stream. onExpired runs when the server no longer has it open.
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
  const [sending, setSending] = useState<"answer" | "deny" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const answers = questionAnswers(questions, picks);
  const busy = sending !== null;

  const send = (kind: "answer" | "deny") => {
    if (busy || (kind === "answer" && !answers)) return;
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

  return (
    <div role="group" aria-label="Question" className="rounded-[var(--radius-lg)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-2)]">
      <div className="flex flex-col gap-3 px-3 pt-2.5 pb-1">
        {questions.map((q, i) => {
          const pick = picks[i];
          const labelId = `${id}-q${i}`;
          return (
            <div key={i} role={q.multiSelect ? "group" : "radiogroup"} aria-labelledby={labelId} className="min-w-0">
              <div className="flex items-baseline gap-2 min-w-0">
                {q.header && (
                  <span className="shrink-0 px-1.5 rounded-[var(--radius-sm)] bg-[var(--color-surface-ai-1)] border border-[var(--color-border-ai-subtle)] text-[11px] text-[var(--color-text-muted)]">
                    {q.header}
                  </span>
                )}
                <span id={labelId} className="min-w-0 text-sm font-medium text-[var(--color-text-primary)] break-words">
                  {q.question}
                </span>
              </div>
              <div className="mt-1.5 -mx-2 flex flex-col gap-0.5">
                {q.options.map((o) => (
                  <Choice
                    key={o.label}
                    multi={q.multiSelect}
                    checked={pick.picked.includes(o.label)}
                    label={o.label}
                    // Agents often repeat the label as its description.
                    description={o.description !== o.label ? o.description : undefined}
                    disabled={busy}
                    onClick={() => setPicks(pickOption(picks, questions, i, o.label))}
                  />
                ))}
                {q.options.length > 0 && (
                  <Choice multi={q.multiSelect} checked={pick.other !== null} label="Other…" disabled={busy} onClick={() => setPicks(toggleOther(picks, questions, i))} />
                )}
              </div>
              {pick.other !== null && (
                <input
                  type="text"
                  aria-label={`Your answer to: ${q.question}`}
                  placeholder="Type your answer"
                  value={pick.other}
                  disabled={busy}
                  // The user just picked Other; a question without options opens with the field.
                  autoFocus={q.options.length > 0}
                  onChange={(e) => setPicks(setOtherText(picks, i, e.target.value))}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.nativeEvent.isComposing) {
                      e.preventDefault();
                      send("answer");
                    }
                  }}
                  className="mt-1 w-full h-8 px-2.5 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-1)] border border-[var(--color-border-ai-subtle)] text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-border-focus)]"
                />
              )}
            </div>
          );
        })}
      </div>
      <div className="flex items-center gap-3 pl-3 pr-1.5 pb-1.5 min-w-0">
        {error && <div className="min-w-0 text-xs text-[var(--color-error)] break-words">{error}</div>}
        <div className="ml-auto flex shrink-0 items-center gap-1">
          <button type="button" disabled={busy} onClick={() => send("deny")} className={`${BUTTON} ${SKIP}`}>
            {sending === "deny" && <Spinner />}
            Skip
          </button>
          <button type="button" disabled={busy || !answers} onClick={() => send("answer")} className={`${BUTTON} ${SEND}`}>
            {sending === "answer" && <Spinner />}
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
