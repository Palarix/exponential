import type { ChatContext, ChatEvent } from "../../api/chat";

export type ChatStep =
  | { kind: "text"; text: string; live: boolean }
  | { kind: "tool"; name: string; arg: string; done: boolean };

export interface ChatTurn {
  user: { text: string; context?: ChatContext; time: string };
  steps: ChatStep[];
  status: "running" | "done" | "stopped" | "error" | "interrupted";
  thinking: boolean;
  durationMs?: number;
  endedAt?: string;
  error?: string;
}

export interface ChatThread {
  turns: ChatTurn[];
  /** Highest settled `seq` seen; settled events at or below it are replays. */
  lastSeq: number;
}

export type ChatThreadAction =
  | { type: "event"; event: ChatEvent }
  | { type: "interrupted" }
  | { type: "reset" };

export const emptyThread: ChatThread = { turns: [], lastSeq: 0 };

/** Folds settled and live chat events into turns. */
export function chatThreadReducer(state: ChatThread, action: ChatThreadAction): ChatThread {
  if (action.type === "reset") return emptyThread;
  if (action.type === "interrupted") {
    return updateLast(state, (t) => (t.status === "running" ? { ...t, status: "interrupted", thinking: false } : t));
  }

  const e = action.event;
  if (e.seq !== undefined && e.seq > 0) {
    if (e.seq <= state.lastSeq) return state;
    state = { ...state, lastSeq: e.seq };
  }

  if (e.type === "user_message") {
    const turn: ChatTurn = {
      user: { text: e.text ?? "", context: e.context, time: e.time },
      steps: [],
      status: "running",
      thinking: false,
    };
    return { ...state, turns: [...state.turns, turn] };
  }

  return updateLast(state, (t) => applyToTurn(t, e));
}

function applyToTurn(t: ChatTurn, e: ChatEvent): ChatTurn {
  const steps = t.steps;
  const last = steps[steps.length - 1];
  switch (e.type) {
    case "thinking":
      return { ...t, thinking: true };
    case "text_delta":
      if (last?.kind === "text" && last.live) {
        return { ...t, thinking: false, steps: [...steps.slice(0, -1), { ...last, text: last.text + (e.text ?? "") }] };
      }
      return { ...t, thinking: false, steps: [...steps, { kind: "text", text: e.text ?? "", live: true }] };
    case "assistant_text": {
      const step: ChatStep = { kind: "text", text: e.text ?? "", live: false };
      const rest = last?.kind === "text" && last.live ? steps.slice(0, -1) : steps;
      return { ...t, thinking: false, steps: [...rest, step] };
    }
    case "tool_start": {
      const name = e.name ?? "";
      if (last?.kind === "tool" && !last.done && last.name === name) {
        return { ...t, thinking: false, steps: [...steps.slice(0, -1), { ...last, arg: e.arg || last.arg }] };
      }
      return { ...t, thinking: false, steps: [...steps, { kind: "tool", name, arg: e.arg ?? "", done: false }] };
    }
    case "tool_call": {
      const name = e.name ?? "";
      const i = lastIndexWhere(steps, (s) => s.kind === "tool" && !s.done && s.name === name);
      const step: ChatStep = { kind: "tool", name, arg: e.arg ?? "", done: true };
      if (i < 0) return { ...t, steps: [...steps, step] };
      return { ...t, steps: steps.map((s, j) => (j === i ? step : s)) };
    }
    case "turn_done":
      return finish(t, e, "done");
    case "turn_stopped":
      return finish(t, e, "stopped");
    case "error":
      return { ...finish(t, e, "error"), error: e.message ?? "Unknown error" };
    default:
      return t;
  }
}

function finish(t: ChatTurn, e: ChatEvent, status: ChatTurn["status"]): ChatTurn {
  return {
    ...t,
    status,
    thinking: false,
    durationMs: e.duration_ms,
    endedAt: e.time,
    steps: t.steps.map((s) => (s.kind === "tool" && !s.done ? { ...s, done: true } : s.kind === "text" && s.live ? { ...s, live: false } : s)),
  };
}

function lastIndexWhere<T>(items: T[], pred: (item: T) => boolean): number {
  for (let i = items.length - 1; i >= 0; i--) if (pred(items[i])) return i;
  return -1;
}

function updateLast(state: ChatThread, fn: (t: ChatTurn) => ChatTurn): ChatThread {
  const n = state.turns.length;
  if (n === 0) return state;
  const next = fn(state.turns[n - 1]);
  if (next === state.turns[n - 1]) return state;
  return { ...state, turns: [...state.turns.slice(0, -1), next] };
}

export function isRunning(state: ChatThread): boolean {
  return state.turns[state.turns.length - 1]?.status === "running";
}

/**
 * Splits a turn into the work (steps up to and including the last tool call,
 * shown collapsed) and the answer (the text after it).
 */
export function splitTurn(t: ChatTurn): { work: ChatStep[]; answer: ChatStep[] } {
  const lastTool = lastIndexWhere(t.steps, (s) => s.kind === "tool");
  return { work: t.steps.slice(0, lastTool + 1), answer: t.steps.slice(lastTool + 1) };
}

/** Whole seconds the turn ran: the reported duration, or the time since the user message. */
export function turnSeconds(t: ChatTurn, now: number): number {
  if (t.durationMs) return Math.round(t.durationMs / 1000);
  const end = t.endedAt ? Date.parse(t.endedAt) : now;
  return Math.max(0, Math.floor((end - Date.parse(t.user.time)) / 1000));
}

// Readable verbs for known tools. A tool without a detail uses `bare`.
const TOOL_VERBS: Record<string, { verb: string; bare?: string }> = {
  Read: { verb: "Read" },
  Grep: { verb: "Searched the code for", bare: "Searched the code" },
  Glob: { verb: "Found files matching", bare: "Listed files" },
  "xpo list": { verb: "Listed issues matching", bare: "Listed issues" },
  "xpo show": { verb: "Opened" },
  "xpo history": { verb: "Read the history of" },
  "xpo add": { verb: "Created issue" },
  "xpo update": { verb: "Updated" },
  "xpo comment": { verb: "Commented on" },
  "xpo link": { verb: "Linked" },
  "xpo unlink": { verb: "Unlinked" },
  "xpo start": { verb: "Started" },
  "xpo merge": { verb: "Merged" },
  "xpo spec": { verb: "Read the spec of" },
  "xpo walkthrough": { verb: "Read the walkthrough of" },
  "xpo artifact": { verb: "Read an artifact of" },
  "xpo rationale": { verb: "Searched design notes for", bare: "Searched design notes" },
};

/**
 * A tool call in words: "Read board.go", "Listed issues matching login".
 * MCP names read as "server tool"; unknown tools keep their name.
 */
export function toolSummary(name: string, arg: string): { text: string; detail: string } {
  const parts = name.split("__");
  const key = parts.length === 3 && parts[0] === "mcp" ? `${parts[1]} ${parts[2]}` : name;
  let detail = arg.replace(/\s+/g, " ").trim();
  if (key === "Read" && detail) detail = detail.split("/").pop() ?? detail;
  const known = TOOL_VERBS[key];
  if (!known) return { text: key, detail };
  return { text: detail || !known.bare ? known.verb : known.bare, detail };
}
