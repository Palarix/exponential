import type { ApprovalOutcome, BackgroundTask, ChatContext, ChatEvent, ChatMode, ToolCallStatus } from "../../api/chat";

/** An approval card on a tool step. No decision means it waits for the user. */
export interface StepApproval {
  id: string;
  input?: string;
  decision?: ApprovalOutcome | "expired";
}

export interface ToolStep {
  kind: "tool";
  id?: string;
  name: string;
  arg: string;
  done: boolean;
  status?: ToolCallStatus;
  message?: string;
  approval?: StepApproval;
}

export type ChatStep = { kind: "text"; text: string; live: boolean } | ToolStep;

export interface ChatTurn {
  user: { text: string; context?: ChatContext; time: string };
  /** The board started this turn after an approved start, with no user message. */
  continued?: boolean;
  /** The agent started this turn on its own; user.text says what woke it. */
  byAgent?: boolean;
  steps: ChatStep[];
  status: "running" | "done" | "stopped" | "error" | "interrupted";
  thinking: boolean;
  durationMs?: number;
  endedAt?: string;
  error?: string;
}

/** The chat entered or left Work Mode after `at` turns. */
export interface ModeDivider {
  at: number;
  mode: ChatMode;
  issueId?: string;
  /** Why, when the board changed the mode on its own. */
  message?: string;
}

export interface ChatThread {
  turns: ChatTurn[];
  dividers: ModeDivider[];
  /** Highest settled `seq` seen; settled events at or below it are replays. */
  lastSeq: number;
  /** Work the agent keeps running between turns. */
  background: BackgroundTask[];
}

export type ChatThreadAction =
  | { type: "event"; event: ChatEvent }
  | { type: "interrupted" }
  /** The server no longer knows an open approval, for example after a restart. */
  | { type: "approval_expired"; id: string }
  /** The background tasks when the chat loaded. */
  | { type: "background"; tasks: BackgroundTask[] }
  | { type: "reset" };

export const emptyThread: ChatThread = { turns: [], dividers: [], lastSeq: 0, background: [] };

/**
 * The background banner's title: how many agents work, or how many shells
 * and monitors run when no agent does. Undefined without tasks.
 */
export function backgroundBanner(tasks: BackgroundTask[]): string | undefined {
  if (tasks.length === 0) return undefined;
  const agents = tasks.filter((t) => t.kind === "agent").length;
  if (agents === 0) return `Monitoring ${tasks.length} ${tasks.length === 1 ? "task" : "tasks"}`;
  return `${agents} ${agents === 1 ? "agent" : "agents"} working`;
}

/** Nothing to show yet: no turn and no divider. */
export function isEmptyThread(state: ChatThread): boolean {
  return state.turns.length === 0 && state.dividers.length === 0;
}

/** A divider in words. */
export function modeDividerText(d: ModeDivider): string {
  if (d.message) return d.message;
  return d.mode === "work" ? `Entered Work Mode on ${d.issueId ?? "an issue"}` : "Entered Chat Mode";
}

/** Folds settled and live chat events into turns. */
export function chatThreadReducer(state: ChatThread, action: ChatThreadAction): ChatThread {
  if (action.type === "reset") return emptyThread;
  if (action.type === "interrupted") {
    return updateLast(state, (t) => (t.status === "running" ? { ...t, status: "interrupted", thinking: false, steps: expireApprovals(t.steps) } : t));
  }
  if (action.type === "approval_expired") {
    return updateLast(state, (t) => updateApproval(t, action.id, "expired"));
  }
  if (action.type === "background") return { ...state, background: action.tasks };

  const e = action.event;
  if (e.seq !== undefined && e.seq > 0) {
    if (e.seq <= state.lastSeq) return state;
    state = { ...state, lastSeq: e.seq };
  }

  if (e.type === "background_tasks") return { ...state, background: e.tasks ?? [] };
  if (e.type === "user_message" || e.type === "turn_continued" || e.type === "agent_turn") {
    const turn: ChatTurn = {
      user: { text: e.text ?? "", context: e.context, time: e.time },
      steps: [],
      status: "running",
      thinking: false,
    };
    if (e.type === "turn_continued") turn.continued = true;
    if (e.type === "agent_turn") turn.byAgent = true;
    return { ...state, turns: [...state.turns, turn] };
  }
  if (e.type === "mode_changed") {
    const divider: ModeDivider = { at: state.turns.length, mode: e.mode ?? "chat", issueId: e.issue_id, message: e.message };
    return { ...state, dividers: [...state.dividers, divider] };
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
      const i = e.tool_id ? openToolIndex(steps, e) : last?.kind === "tool" && !last.done && last.name === name ? steps.length - 1 : -1;
      if (i >= 0) {
        return { ...t, thinking: false, steps: steps.map((s, j) => (j === i && s.kind === "tool" ? { ...s, arg: e.arg || s.arg } : s)) };
      }
      return { ...t, thinking: false, steps: [...steps, { kind: "tool", id: e.tool_id, name, arg: e.arg ?? "", done: false }] };
    }
    case "tool_call": {
      const i = openToolIndex(steps, e);
      const step: ToolStep = { kind: "tool", id: e.tool_id, name: e.name ?? "", arg: e.arg ?? "", done: true, status: e.status, message: e.message };
      if (i < 0) return { ...t, steps: [...steps, step] };
      return { ...t, steps: steps.map((s, j) => (j === i ? { ...step, approval: s.kind === "tool" ? s.approval : undefined } : s)) };
    }
    case "approval_requested": {
      const approval: StepApproval = { id: e.approval_id ?? "", input: e.input };
      const i = e.tool_id ? openToolIndex(steps, e) : -1;
      if (i < 0) {
        return { ...t, steps: [...steps, { kind: "tool", id: e.tool_id, name: e.name ?? "", arg: e.arg ?? "", done: false, approval }] };
      }
      return { ...t, steps: steps.map((s, j) => (j === i && s.kind === "tool" ? { ...s, arg: s.arg || (e.arg ?? ""), approval } : s)) };
    }
    case "approval_resolved":
      return e.approval_id && e.decision ? updateApproval(t, e.approval_id, e.decision) : t;
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
    steps: expireApprovals(t.steps).map((s) => (s.kind === "tool" && !s.done ? { ...s, done: true } : s.kind === "text" && s.live ? { ...s, live: false } : s)),
  };
}

/** Marks every open approval expired: its turn ended without an answer. */
function expireApprovals(steps: ChatStep[]): ChatStep[] {
  return steps.map((s) => (s.kind === "tool" && s.approval && !s.approval.decision ? { ...s, approval: { ...s.approval, decision: "expired" } } : s));
}

function updateApproval(t: ChatTurn, id: string, decision: NonNullable<StepApproval["decision"]>): ChatTurn {
  const i = t.steps.findIndex((s) => s.kind === "tool" && s.approval?.id === id);
  if (i < 0) return t;
  return {
    ...t,
    steps: t.steps.map((s, j) => (j === i && s.kind === "tool" && s.approval ? { ...s, approval: { ...s.approval, decision } } : s)),
  };
}

/** Tool steps whose approval card waits for the user. */
export function pendingApprovals(t: ChatTurn): (ToolStep & { approval: StepApproval })[] {
  return t.steps.filter((s): s is ToolStep & { approval: StepApproval } => s.kind === "tool" && !!s.approval && !s.approval.decision);
}

/** The open tool step an event belongs to: by tool ID, or by name for chats saved without IDs. */
function openToolIndex(steps: ChatStep[], e: ChatEvent): number {
  if (e.tool_id) return lastIndexWhere(steps, (s) => s.kind === "tool" && s.id === e.tool_id);
  return lastIndexWhere(steps, (s) => s.kind === "tool" && !s.done && s.name === (e.name ?? ""));
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

/** A run of text steps, shown as a message, or a run of tool calls, shown as one line. */
export type TurnSegment = { kind: "text"; text: string; live: boolean } | { kind: "tools"; steps: ToolStep[] };

/** Splits a turn into messages and the groups of tool calls between them, in order. */
export function turnSegments(t: ChatTurn): TurnSegment[] {
  const segs: TurnSegment[] = [];
  for (const step of t.steps) {
    const last = segs[segs.length - 1];
    if (step.kind === "tool") {
      if (last?.kind === "tools") last.steps.push(step);
      else segs.push({ kind: "tools", steps: [step] });
    } else if (last?.kind === "text") {
      segs[segs.length - 1] = { kind: "text", text: `${last.text}\n\n${step.text}`, live: step.live };
    } else {
      segs.push({ kind: "text", text: step.text, live: step.live });
    }
  }
  return segs;
}

/** A finished group's line: "3 actions", and "2 failed" when calls failed. */
export function groupSummary(steps: ToolStep[]): { text: string; failed: string | null } {
  const n = steps.filter(isFailed).length;
  return { text: `${steps.length} action${steps.length === 1 ? "" : "s"}`, failed: n > 0 ? `${n} failed` : null };
}

/** The calls a group's timeline lists: the active group's line already shows its latest. */
export function timelineSteps(steps: ToolStep[], active: boolean): ToolStep[] {
  return active ? steps.slice(0, -1) : steps;
}

/** "12s", "1m", "1m 24s". */
export function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) return `${s}s`;
  return s === 0 ? `${m}m` : `${m}m ${s}s`;
}

/** Tool calls of the turn that failed or that the agent's rules denied. A call the user denied on a card is not a failure. */
export function failedSteps(t: ChatTurn): ToolStep[] {
  return t.steps.filter((s): s is ToolStep => s.kind === "tool" && isFailed(s));
}

/** The call failed, or the agent's rules denied it. */
export function isFailed(s: ToolStep): boolean {
  return !userDenied(s) && (s.status === "error" || s.status === "denied");
}

/** The user answered the step's approval card with a denial, or the turn ended first. */
export function userDenied(s: ToolStep): boolean {
  const d = s.approval?.decision;
  return d === "deny" || d === "cancelled" || d === "expired";
}

/** Whole seconds the turn ran: the reported duration, or the time since the user message. */
export function turnSeconds(t: ChatTurn, now: number): number {
  if (t.durationMs) return Math.round(t.durationMs / 1000);
  const end = t.endedAt ? Date.parse(t.endedAt) : now;
  return Math.max(0, Math.floor((end - Date.parse(t.user.time)) / 1000));
}

// Readable verbs for known tools: `verb` for a call that ran, `action` for
// one that was only asked for (an approval card, a failed or denied call).
// A tool without a detail uses `bare` / `bareAction`. `icon` stands in for the
// verb on a work-list row; ToolIcon draws it.
// Icons group tools by what they do: `read` looks at something known, `edit`
// changes something, `search` finds things. A few xpo tools keep their own.
export type ToolIconId = "read" | "edit" | "search" | "shell" | "skill" | "add" | "comment" | "link" | "unlink" | "artifact" | "git_status" | "git_diff" | "spec" | "walkthrough" | "start" | "merge" | "tool";

interface ToolVerbs {
  icon: ToolIconId;
  verb: string;
  bare?: string;
  action: string;
  bareAction?: string;
  /** Verbs per operation, for tools whose detail starts with one: "write xpo-1". */
  ops?: Record<string, { verb: string; action: string }>;
}

function fileOps(thing: string): ToolVerbs["ops"] {
  return {
    read: { verb: `Read the ${thing} of`, action: `Read the ${thing} of` },
    write: { verb: `Wrote the ${thing} of`, action: `Write the ${thing} of` },
    delete: { verb: `Deleted the ${thing} of`, action: `Delete the ${thing} of` },
  };
}
const TOOL_VERBS: Record<string, ToolVerbs> = {
  Bash: { icon: "shell", verb: "Ran a shell command", action: "Run a shell command" },
  Edit: { icon: "edit", verb: "Edited", action: "Edit" },
  Write: { icon: "edit", verb: "Wrote", action: "Write" },
  Read: { icon: "read", verb: "Read", action: "Read" },
  Grep: { icon: "search", verb: "Searched the code for", bare: "Searched the code", action: "Search the code for", bareAction: "Search the code" },
  Glob: { icon: "search", verb: "Found files matching", bare: "Listed files", action: "Find files matching", bareAction: "List files" },
  Skill: { icon: "skill", verb: "Loaded the skill", action: "Load the skill" },
  "xpo list": { icon: "search", verb: "Listed issues matching", bare: "Listed issues", action: "List issues matching", bareAction: "List issues" },
  "xpo show": { icon: "read", verb: "Opened", action: "Open" },
  "xpo history": { icon: "read", verb: "Read the history of", action: "Read the history of" },
  "xpo add": { icon: "add", verb: "Created issue", action: "Create issue" },
  "xpo update": { icon: "edit", verb: "Updated", action: "Update" },
  "xpo comment": { icon: "comment", verb: "Commented on", action: "Comment on" },
  "xpo link": { icon: "link", verb: "Linked", action: "Link" },
  "xpo unlink": { icon: "unlink", verb: "Unlinked", action: "Unlink" },
  "xpo start": { icon: "start", verb: "Started", action: "Enter Work Mode on" },
  "xpo merge": { icon: "merge", verb: "Merged", action: "Merge" },
  "xpo spec": { icon: "spec", verb: "Read the spec of", action: "Read the spec of", ops: fileOps("spec") },
  "xpo walkthrough": { icon: "walkthrough", verb: "Read the walkthrough of", action: "Read the walkthrough of", ops: fileOps("walkthrough") },
  "xpo artifact": {
    icon: "artifact",
    verb: "Read an artifact of",
    action: "Read an artifact of",
    ops: {
      read: { verb: "Read an artifact of", action: "Read an artifact of" },
      add: { verb: "Added an artifact to", action: "Add an artifact to" },
      delete: { verb: "Deleted an artifact of", action: "Delete an artifact of" },
      list: { verb: "Listed the artifacts of", action: "List the artifacts of" },
    },
  },
  "xpo rationale": { icon: "search", verb: "Searched design notes for", bare: "Searched design notes", action: "Search design notes for", bareAction: "Search design notes" },
  "xpo git_status": { icon: "git_status", verb: "Checked the git status of", action: "Check the git status of" },
  "xpo git_diff": { icon: "git_diff", verb: "Read the diff of", action: "Read the diff of" },
  "xpo git_log": { icon: "read", verb: "Listed commits of", bare: "Listed commits", action: "List commits of", bareAction: "List commits" },
  "xpo git_show": { icon: "read", verb: "Read commit", action: "Read commit" },
  "xpo code_history": { icon: "read", verb: "Traced the history of", action: "Trace the history of" },
};

/** The detail shown next to a tool's verb: file names, not paths; short SHAs. */
function toolDetail(key: string, arg: string): string {
  let detail = arg.replace(/\s+/g, " ").trim();
  if ((key === "Read" || key === "Edit" || key === "Write") && detail) detail = detail.split("/").pop() ?? detail;
  if (key === "xpo git_show") detail = detail.slice(0, 12);
  return detail;
}

/**
 * A known tool's verbs for this call, and its arg without the operation. A
 * chat saved before details named the operation keeps the default verbs.
 */
function callVerbs(key: string, arg: string): { known?: ToolVerbs; arg: string } {
  const known = TOOL_VERBS[key];
  const m = /^(\w+)(?:\s+([\s\S]*))?$/.exec(arg.trim());
  const op = m && known?.ops?.[m[1]];
  if (!op) return { known, arg };
  return { known: { icon: known.icon, ...op }, arg: m[2] ?? "" };
}

export function toolSummary(name: string, arg: string): { text: string; detail: string } {
  const key = toolLabel(name);
  const call = callVerbs(key, arg);
  const detail = toolDetail(key, call.arg);
  const known = call.known;
  if (!known) return { text: key, detail };
  return { text: detail || !known.bare ? known.verb : known.bare, detail };
}

/** A tool's name as the user knows it: MCP names read as "server tool". */
export function toolLabel(name: string): string {
  const parts = name.split("__");
  return parts.length === 3 && parts[0] === "mcp" ? `${parts[1]} ${parts[2]}` : name;
}

const APPROVAL_OUTCOMES: Record<NonNullable<StepApproval["decision"]>, string> = {
  allow: "Allowed once",
  allow_session: "Allowed for this chat",
  deny: "Denied",
  cancelled: "Cancelled",
  expired: "Expired",
};

/** How an approval card ended, in words. */
export function approvalOutcomeLabel(decision: NonNullable<StepApproval["decision"]>): string {
  return APPROVAL_OUTCOMES[decision];
}

/** What a call that has not run would do: "Merge upe-1", "Edit main.go". Unknown tools read "Use <tool>". */
export function approvalTitle(name: string, arg: string): { action: string; detail: string } {
  const key = toolLabel(name);
  const call = callVerbs(key, arg);
  const detail = toolDetail(key, call.arg);
  const known = call.known;
  if (!known) return { action: `Use ${key}`, detail };
  return { action: detail || !known.bareAction ? known.action : known.bareAction, detail };
}

/**
 * A work-list row in words. A call that did not run (the user denied it, or it
 * failed) reads as what was asked: "Merge upe-1", not "Merged upe-1".
 */
export function stepSummary(step: ToolStep): { text: string; detail: string } {
  const ran = !userDenied(step) && step.status !== "error" && step.status !== "denied";
  if (ran) return toolSummary(step.name, step.arg);
  const key = toolLabel(step.name);
  if (!TOOL_VERBS[key]) return { text: key, detail: toolDetail(key, step.arg) };
  const { action, detail } = approvalTitle(step.name, step.arg);
  return { text: action, detail };
}

/**
 * A work-list row's icon. `label` is the row's verb, for the icon's tooltip
 * and accessible name. An unknown tool gets the generic icon, and its `name`
 * is shown on the row.
 */
export function stepIcon(step: ToolStep): { icon: ToolIconId; label: string; name?: string; detail: string } {
  const { text, detail } = stepSummary(step);
  const known = TOOL_VERBS[toolLabel(step.name)];
  if (!known) return { icon: "tool", label: text, name: text, detail };
  return { icon: known.icon, label: text, detail };
}

function argText(v: unknown): string {
  if (typeof v === "string") return v;
  if (Array.isArray(v) && v.every((x) => typeof x !== "object" || x === null)) return v.join(", ");
  return typeof v === "object" ? JSON.stringify(v) : String(v);
}

/**
 * What an approval card shows besides its headline: the command of a shell
 * call, and the arguments the headline does not already show.
 */
export function approvalArgs(name: string, input: string | undefined, detail: string): { command?: string; fields: [string, string][] } {
  let args: Record<string, unknown>;
  try {
    const parsed: unknown = JSON.parse(input ?? "");
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return { fields: [] };
    args = parsed as Record<string, unknown>;
  } catch {
    return { fields: [] };
  }
  const shell = name === "Bash" && typeof args.command === "string";
  const fields: [string, string][] = [];
  for (const [key, value] of Object.entries(args)) {
    if (shell && (key === "command" || key === "description")) continue;
    if (value === null || value === undefined || value === "" || (Array.isArray(value) && value.length === 0)) continue;
    if (value === detail) continue;
    fields.push([key, argText(value)]);
  }
  return shell ? { command: args.command as string, fields } : { fields };
}

/**
 * The reason shown after a failed call's row: the message's first non-blank
 * line, labelled "failed" or "denied" (the agent's own rules blocked it).
 * `more` is set when the message has further lines.
 */
export function failureReason(step: ToolStep): { label: string; text: string; more: boolean } {
  const lines = (step.message ?? "").split("\n").map((l) => l.trim());
  const first = lines.findIndex((l) => l !== "");
  return {
    label: step.status === "denied" ? "denied" : "failed",
    text: first < 0 ? "" : lines[first],
    more: first >= 0 && lines.slice(first + 1).some((l) => l !== ""),
  };
}
