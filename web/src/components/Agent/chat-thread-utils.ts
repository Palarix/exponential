import type { ShortcutBinding } from "../../keyboard";
import type { ApprovalOutcome, BackgroundTask, ChatAttachment, ChatContext, ChatEvent, ChatMode, QueuedMessage, ToolCallStatus } from "../../api/chat";

/** An approval or question card on a tool step. No decision means it waits for the user. */
export interface StepApproval {
  id: string;
  input?: string;
  decision?: ApprovalOutcome | "expired";
  /** A question card's answers, by question text. */
  answers?: Record<string, string>;
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

/** A message the user sent into the running turn. */
export interface SteerStep {
  kind: "steer";
  text: string;
  attachments?: ChatAttachment[];
}

export type ChatStep = { kind: "text"; text: string; live: boolean } | ToolStep | SteerStep;

export interface ChatTurn {
  user: { text: string; context?: ChatContext; time: string; attachments?: ChatAttachment[] };
  /** The board started this turn after an approved start, with no user message. */
  continued?: boolean;
  /** The agent started this turn on its own; user.text says what woke it. */
  byAgent?: boolean;
  /** The seq of the event that started the turn; a rewind drops turns by it. */
  seq?: number;
  /** Where the agent's session stood when the turn ended; an edit rewinds to it. */
  anchor?: string;
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
  /** Messages sent while a turn ran, waiting on the board. */
  queue: QueuedMessage[];
  /** The chat's history is in: what arrives from now on is new. */
  loaded: boolean;
}

export type ChatThreadAction =
  | { type: "event"; event: ChatEvent }
  | { type: "interrupted" }
  /** The server no longer knows an open approval, for example after a restart. */
  | { type: "approval_expired"; id: string }
  /** The background tasks when the chat loaded. */
  | { type: "background"; tasks: BackgroundTask[] }
  /** The queue when the chat loaded. */
  | { type: "queue"; queue: QueuedMessage[] }
  /** The chat's history is in, or there is none: a new chat. */
  | { type: "loaded" }
  | { type: "reset" };

// Events that change a chat's title, state or mode in the chat list.
const META_EVENTS = new Set<ChatEvent["type"]>(["user_message", "turn_done", "turn_stopped", "error", "approval_requested", "approval_resolved", "mode_changed", "turn_continued", "agent_turn", "title"]);

/** Whether an event changes the chat list: its title, state or mode. */
export function changesChatMeta(type: ChatEvent["type"]): boolean {
  return META_EVENTS.has(type);
}

export const emptyThread: ChatThread = { turns: [], dividers: [], lastSeq: 0, background: [], queue: [], loaded: false };

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
  if (action.type === "queue") return { ...state, queue: action.queue };
  if (action.type === "loaded") return { ...state, loaded: true };

  const e = action.event;
  if (e.seq !== undefined && e.seq > 0) {
    if (e.seq <= state.lastSeq) return state;
    state = { ...state, lastSeq: e.seq };
  }

  if (e.type === "background_tasks") return { ...state, background: e.tasks ?? [] };
  if (e.type === "queue") return { ...state, queue: e.queue ?? [] };
  if (e.type === "user_message" || e.type === "turn_continued" || e.type === "agent_turn") {
    const turn: ChatTurn = {
      user: { text: e.text ?? "", context: e.context, time: e.time, ...(e.attachments?.length ? { attachments: e.attachments } : {}) },
      steps: [],
      status: "running",
      thinking: false,
      ...(e.seq !== undefined && { seq: e.seq }),
    };
    if (e.type === "turn_continued") turn.continued = true;
    if (e.type === "agent_turn") turn.byAgent = true;
    return { ...state, turns: [...state.turns, turn] };
  }
  if (e.type === "rewound") {
    const turns = state.turns.filter((t) => t.seq === undefined || t.seq < (e.from_seq ?? 0));
    return { ...state, turns, dividers: state.dividers.filter((d) => d.at <= turns.length) };
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
    case "steer": {
      const step: SteerStep = { kind: "steer", text: e.text ?? "", ...(e.attachments?.length ? { attachments: e.attachments } : {}) };
      const rest = steps.map((s) => (s.kind === "text" && s.live ? { ...s, live: false } : s));
      return { ...t, steps: [...rest, step] };
    }
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
      return e.approval_id && e.decision ? updateApproval(t, e.approval_id, e.decision, e.answers) : t;
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
    ...(e.anchor && { anchor: e.anchor }),
    steps: expireApprovals(t.steps).map((s) => (s.kind === "tool" && !s.done ? { ...s, done: true } : s.kind === "text" && s.live ? { ...s, live: false } : s)),
  };
}

/** Marks every open approval expired: its turn ended without an answer. */
function expireApprovals(steps: ChatStep[]): ChatStep[] {
  return steps.map((s) => (s.kind === "tool" && s.approval && !s.approval.decision ? { ...s, approval: { ...s.approval, decision: "expired" } } : s));
}

function updateApproval(t: ChatTurn, id: string, decision: NonNullable<StepApproval["decision"]>, answers?: Record<string, string>): ChatTurn {
  const i = t.steps.findIndex((s) => s.kind === "tool" && s.approval?.id === id);
  if (i < 0) return t;
  return {
    ...t,
    steps: t.steps.map((s, j) => (j === i && s.kind === "tool" && s.approval ? { ...s, approval: { ...s.approval, decision, ...(answers && { answers }) } } : s)),
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

/**
 * A run of text steps, shown as a message; a run of tool calls, shown as one
 * line; or an answered question, shown with its answers.
 */
export type TurnSegment = { kind: "text"; text: string; live: boolean } | { kind: "tools"; steps: ToolStep[] } | { kind: "question"; step: ToolStep } | SteerStep;

/** Splits a turn into messages and the groups of tool calls between them, in order. */
export function turnSegments(t: ChatTurn): TurnSegment[] {
  const segs: TurnSegment[] = [];
  for (const step of t.steps) {
    const last = segs[segs.length - 1];
    if (step.kind === "steer") {
      segs.push(step);
    } else if (step.kind === "tool" && step.approval?.decision === "answer") {
      segs.push({ kind: "question", step });
    } else if (step.kind === "tool") {
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

// Readable verbs for known tools: `verb` for a call that ran, `doing` for one
// that runs, `action` for one that was only asked for (an approval card, a
// failed or denied call). A tool without a detail uses `bare` / `bareDoing` /
// `bareAction`. `icon` stands before the verb on a work-list row; ToolIcon draws it.
// Icons group tools by what they do: `read` looks at something known, `edit`
// changes something, `search` finds things. A few xpo tools keep their own.
export type ToolIconId = "read" | "edit" | "search" | "shell" | "skill" | "add" | "comment" | "link" | "unlink" | "artifact" | "git_status" | "git_diff" | "spec" | "walkthrough" | "start" | "merge" | "agent" | "question" | "tool";

interface Verbs {
  verb: string;
  doing: string;
  action: string;
}

interface ToolVerbs extends Verbs {
  icon: ToolIconId;
  bare?: string;
  bareDoing?: string;
  bareAction?: string;
  /** Verbs per operation, for tools whose detail starts with one: "write xpo-1". */
  ops?: Record<string, Verbs>;
}

function fileOps(thing: string): ToolVerbs["ops"] {
  return {
    read: { verb: `Read the ${thing} of`, doing: `Reading the ${thing} of`, action: `Read the ${thing} of` },
    write: { verb: `Wrote the ${thing} for`, doing: `Writing the ${thing} for`, action: `Write the ${thing} for` },
    delete: { verb: `Deleted the ${thing} of`, doing: `Deleting the ${thing} of`, action: `Delete the ${thing} of` },
  };
}
// A subagent call. Claude Code names the tool Agent; older versions, Task.
const AGENT_VERBS: ToolVerbs = { icon: "agent", verb: "Delegated", doing: "Delegating", bare: "Delegated to a subagent", bareDoing: "Delegating to a subagent", action: "Delegate", bareAction: "Delegate to a subagent" };

const TOOL_VERBS: Record<string, ToolVerbs> = {
  Bash: { icon: "shell", verb: "Ran a shell command", doing: "Running a shell command", action: "Run a shell command" },
  Edit: { icon: "edit", verb: "Edited", doing: "Editing", action: "Edit" },
  Write: { icon: "edit", verb: "Wrote", doing: "Writing", action: "Write" },
  Read: { icon: "read", verb: "Read", doing: "Reading", action: "Read" },
  Grep: { icon: "search", verb: "Searched for", doing: "Searching for", bare: "Searched the code", bareDoing: "Searching the code", action: "Search for", bareAction: "Search the code" },
  Glob: { icon: "search", verb: "Found files matching", doing: "Finding files matching", bare: "Listed files", bareDoing: "Listing files", action: "Find files matching", bareAction: "List files" },
  Skill: { icon: "skill", verb: "Loaded the skill", doing: "Loading the skill", action: "Load the skill" },
  WebSearch: { icon: "search", verb: "Searched the web for", doing: "Searching the web for", bare: "Searched the web", bareDoing: "Searching the web", action: "Search the web for", bareAction: "Search the web" },
  WebFetch: { icon: "read", verb: "Fetched", doing: "Fetching", bare: "Fetched a page", bareDoing: "Fetching a page", action: "Fetch", bareAction: "Fetch a page" },
  LSP: { icon: "search", verb: "Looked up symbols in", doing: "Looking up symbols in", bare: "Looked up symbols", bareDoing: "Looking up symbols", action: "Look up symbols in", bareAction: "Look up symbols" },
  TaskCreate: { icon: "add", verb: "Added the to-do", doing: "Adding the to-do", bare: "Added a to-do", bareDoing: "Adding a to-do", action: "Add the to-do", bareAction: "Add a to-do" },
  TaskUpdate: { icon: "edit", verb: "Updated to-do", doing: "Updating to-do", bare: "Updated a to-do", bareDoing: "Updating a to-do", action: "Update to-do", bareAction: "Update a to-do" },
  TaskList: { icon: "read", verb: "Listed the to-dos", doing: "Listing the to-dos", action: "List the to-dos" },
  TaskGet: { icon: "read", verb: "Read to-do", doing: "Reading to-do", bare: "Read a to-do", bareDoing: "Reading a to-do", action: "Read to-do", bareAction: "Read a to-do" },
  Agent: AGENT_VERBS,
  Task: AGENT_VERBS,
  AskUserQuestion: { icon: "question", verb: "Asked", doing: "Asking", bare: "Asked a question", bareDoing: "Asking a question", action: "Ask", bareAction: "Ask a question" },
  "xpo list": { icon: "search", verb: "Listed issues matching", doing: "Listing issues matching", bare: "Listed issues", bareDoing: "Listing issues", action: "List issues matching", bareAction: "List issues" },
  "xpo show": { icon: "read", verb: "Read", doing: "Reading", action: "Read" },
  "xpo history": { icon: "read", verb: "Read the history of", doing: "Reading the history of", action: "Read the history of" },
  "xpo add": { icon: "add", verb: "Created issue", doing: "Creating issue", action: "Create issue" },
  "xpo update": { icon: "edit", verb: "Updated", doing: "Updating", action: "Update" },
  "xpo comment": { icon: "comment", verb: "Commented on", doing: "Commenting on", action: "Comment on" },
  "xpo link": { icon: "link", verb: "Linked", doing: "Linking", action: "Link" },
  "xpo unlink": { icon: "unlink", verb: "Unlinked", doing: "Unlinking", action: "Unlink" },
  "xpo start": { icon: "start", verb: "Started", doing: "Starting", action: "Enter Work Mode on" },
  "xpo merge": { icon: "merge", verb: "Merged", doing: "Merging", action: "Merge" },
  "xpo spec": { icon: "spec", verb: "Read the spec of", doing: "Reading the spec of", action: "Read the spec of", ops: fileOps("spec") },
  "xpo walkthrough": { icon: "walkthrough", verb: "Read the walkthrough of", doing: "Reading the walkthrough of", action: "Read the walkthrough of", ops: fileOps("walkthrough") },
  "xpo artifact": {
    icon: "artifact",
    verb: "Read an artifact of",
    doing: "Reading an artifact of",
    action: "Read an artifact of",
    ops: {
      read: { verb: "Read an artifact of", doing: "Reading an artifact of", action: "Read an artifact of" },
      add: { verb: "Added an artifact to", doing: "Adding an artifact to", action: "Add an artifact to" },
      delete: { verb: "Deleted an artifact of", doing: "Deleting an artifact of", action: "Delete an artifact of" },
      list: { verb: "Listed the artifacts of", doing: "Listing the artifacts of", action: "List the artifacts of" },
    },
  },
  "xpo rationale": { icon: "search", verb: "Searched design notes for", doing: "Searching design notes for", bare: "Searched design notes", bareDoing: "Searching design notes", action: "Search design notes for", bareAction: "Search design notes" },
  "xpo git_status": { icon: "git_status", verb: "Checked the git status of", doing: "Checking the git status of", action: "Check the git status of" },
  "xpo git_diff": { icon: "git_diff", verb: "Read the diff of", doing: "Reading the diff of", action: "Read the diff of" },
  "xpo git_log": { icon: "read", verb: "Listed commits of", doing: "Listing commits of", bare: "Listed commits", bareDoing: "Listing commits", action: "List commits of", bareAction: "List commits" },
  "xpo git_show": { icon: "read", verb: "Read commit", doing: "Reading commit", action: "Read commit" },
  "xpo code_history": { icon: "read", verb: "Traced the history of", doing: "Tracing the history of", action: "Trace the history of" },
};

/** The detail shown next to a tool's verb: file names, not paths; short SHAs. */
function toolDetail(key: string, arg: string): string {
  let detail = arg.replace(/\s+/g, " ").trim();
  if ((key === "Read" || key === "Edit" || key === "Write" || key === "LSP") && detail) detail = detail.split("/").pop() ?? detail;
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

/** A call that ran, in words: "Read board.go". A running call reads "Reading board.go". */
export function toolSummary(name: string, arg: string, running = false): { text: string; detail: string } {
  const key = toolLabel(name);
  const call = callVerbs(key, arg);
  const detail = toolDetail(key, call.arg);
  const known = call.known;
  if (!known) return { text: key, detail };
  if (running) return { text: detail || !known.bareDoing ? known.doing : known.bareDoing, detail };
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
  answer: "Answered",
  cancelled: "Cancelled",
  expired: "Expired",
};

/** How an approval card ended, in words. A question card the user denied went to "Chat about this". */
export function approvalOutcomeLabel(decision: NonNullable<StepApproval["decision"]>, question = false): string {
  return question && decision === "deny" ? "Chat instead" : APPROVAL_OUTCOMES[decision];
}

/** The step is an AskUserQuestion call: its card asks questions, not for approval. */
export function isQuestion(step: ToolStep): boolean {
  return step.name === "AskUserQuestion";
}

/** The card that takes focus, approval or question: the focused card, else the oldest. */
export function activeCardId(pending: (ToolStep & { approval: StepApproval })[], focusedId: string | null): string | null {
  return (pending.find((s) => s.approval.id === focusedId) ?? pending[0])?.approval.id ?? null;
}

/** A text field, or a rich-text editor such as the composer, with something in it: the user is typing there. */
export function typing(el: Element | null): boolean {
  if (el instanceof HTMLTextAreaElement || el instanceof HTMLInputElement) return el.value.trim() !== "";
  return el?.getAttribute("contenteditable") === "true" && (el.textContent ?? "").trim() !== "";
}

/**
 * The decision a key makes on an approval card, as in a form: Enter allows
 * once and Esc denies. Only with focus on the card itself, so a focused
 * button keeps its own Enter.
 */
export function approvalKeyDecision(e: { key: string; onCard: boolean; composing: boolean; modified: boolean }): "allow" | "deny" | null {
  if (!e.onCard || e.composing || e.modified) return null;
  return e.key === "Enter" ? "allow" : e.key === "Escape" ? "deny" : null;
}

/**
 * Esc in the composer gives up focus, so global shortcuts work again. The
 * draft stays. Not during IME input, and not when another handler took Esc.
 */
export function composerEscapeBlurs(e: { key: string; enabled: boolean; composing: boolean; handled: boolean }): boolean {
  return e.enabled && e.key === "Escape" && !e.composing && !e.handled;
}

/**
 * What ↑ does in an empty composer: pull the last queued message back into
 * it, or else load the last message to edit. With text in the composer, ↑
 * moves the caret as usual.
 */
export function composerArrowUp(e: { key: string; empty: boolean; composing: boolean; modified: boolean; queued: boolean; canEdit: boolean }): "pull" | "edit" | null {
  if (e.key !== "ArrowUp" || !e.empty || e.composing || e.modified) return null;
  if (e.queued) return "pull";
  return e.canEdit ? "edit" : null;
}

/** The way back into the composer after Esc: `/`, as for search fields. */
export function focusComposerShortcut(run: () => void): ShortcutBinding {
  return { id: "agent.focus-composer", key: "/", label: "Focus composer", group: "Agent", run };
}

/** The last open card went away while it held focus, so focus fell to the page. */
export function cardsDone(before: number, after: number, focusOnPage: boolean): boolean {
  return before > 0 && after === 0 && focusOnPage;
}

export interface QuestionOption {
  label: string;
  description: string;
}

/** One question of an AskUserQuestion call. */
export interface Question {
  question: string;
  header: string;
  options: QuestionOption[];
  multiSelect: boolean;
}

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}

/** The questions of an AskUserQuestion input. Questions without text are left out. */
export function parseQuestions(input: string | undefined): Question[] {
  let raw: unknown;
  try {
    raw = (JSON.parse(input ?? "") as { questions?: unknown })?.questions;
  } catch {
    return [];
  }
  if (!Array.isArray(raw)) return [];
  const out: Question[] = [];
  for (const q of raw as Record<string, unknown>[]) {
    const question = str(q?.question);
    if (!question) continue;
    const options = Array.isArray(q.options) ? (q.options as Record<string, unknown>[]).map((o) => ({ label: str(o?.label), description: str(o?.description) })).filter((o) => o.label) : [];
    out.push({ question, header: str(q.header), options, multiSelect: q.multiSelect === true });
  }
  return out;
}

/**
 * What the user picked for one question: option labels, and the free text.
 * other is null while the field is closed; it opens on "Other…", and at
 * once for a question without options.
 */
export interface QuestionPick {
  picked: string[];
  other: string | null;
}

export function emptyPicks(questions: Question[]): QuestionPick[] {
  return questions.map((q) => ({ picked: [], other: q.options.length === 0 ? "" : null }));
}

function updatePick(picks: QuestionPick[], i: number, fn: (p: QuestionPick) => QuestionPick): QuestionPick[] {
  return picks.map((p, j) => (j === i ? fn(p) : p));
}

/** Picks an option: the only one of a single-select question, which closes Other; toggled for a multi-select one. */
export function pickOption(picks: QuestionPick[], questions: Question[], i: number, label: string): QuestionPick[] {
  return updatePick(picks, i, (p) => {
    if (!questions[i].multiSelect) return { picked: [label], other: null };
    return { ...p, picked: p.picked.includes(label) ? p.picked.filter((l) => l !== label) : [...p.picked, label] };
  });
}

/** Opens or closes the free-text field. For a single-select question it replaces the option. */
export function toggleOther(picks: QuestionPick[], questions: Question[], i: number): QuestionPick[] {
  return updatePick(picks, i, (p) => {
    if (p.other !== null) return { ...p, other: null };
    return { picked: questions[i].multiSelect ? p.picked : [], other: "" };
  });
}

export function setOtherText(picks: QuestionPick[], i: number, text: string): QuestionPick[] {
  return updatePick(picks, i, (p) => ({ ...p, other: text }));
}

/**
 * The answers by question text, as Claude Code's own picker writes them: the
 * picked labels in option order, then the free text, joined with ", ". Null
 * until every question has an answer, and without questions.
 */
export function questionAnswers(questions: Question[], picks: QuestionPick[]): Record<string, string> | null {
  if (questions.length === 0) return null;
  const answers: Record<string, string> = {};
  for (const [i, q] of questions.entries()) {
    const answer = questionAnswerText(q, picks[i]);
    if (!answer) return null;
    answers[q.question] = answer;
  }
  return answers;
}

/** One question's answer: the picked labels in option order, then the free text. Empty without an answer. */
export function questionAnswerText(q: Question, p: QuestionPick | undefined): string {
  const labels = q.options.map((o) => o.label).filter((l) => p?.picked.includes(l));
  const other = p?.other?.trim();
  return [...labels, ...(other ? [other] : [])].join(", ");
}

export function questionAnswered(q: Question, p: QuestionPick | undefined): boolean {
  return questionAnswerText(q, p) !== "";
}

/** A question's tab: its header, or Q and its number. */
export function questionTabLabel(q: Question, i: number): string {
  return q.header || `Q${i + 1}`;
}

/**
 * The card's steps are its questions, then a review step. The review step's
 * index, or null: a single question has no review step.
 */
export function reviewStep(questions: Question[]): number | null {
  return questions.length > 1 ? questions.length : null;
}

/** The index of the card's last step: the review step, or the only question. */
function lastStep(questions: Question[]): number {
  return reviewStep(questions) ?? questions.length - 1;
}

export type QuestionMove = { kind: "send" } | { kind: "go"; index: number };

/** Which way the question card last moved between steps. Null until the first move. */
export type StepDirection = "forward" | "back" | null;

/**
 * The class that slides a new step in from its direction: from the right
 * going forward, from the left going back. index.css turns it into a fade
 * under reduced motion and drops it when motion is off.
 */
export function stepInClass(direction: StepDirection): string {
  if (direction === "forward") return "animate-step-in-forward";
  if (direction === "back") return "animate-step-in-back";
  return "";
}

/**
 * What the primary button (or Enter) does on step `current`: Next moves to
 * the next step, and from the last question to review. On the last step it
 * sends once every question has an answer, and else moves to the first
 * question without one.
 */
export function questionAdvance(questions: Question[], picks: QuestionPick[], current: number): QuestionMove {
  if (current < lastStep(questions)) return { kind: "go", index: current + 1 };
  if (questionAnswers(questions, picks)) return { kind: "send" };
  const open = questions.findIndex((q, i) => !questionAnswered(q, picks[i]));
  return { kind: "go", index: Math.max(open, 0) };
}

/**
 * Where a pick on question `current` moves to on its own: the next step,
 * after a single-select option. A single question stays, so nothing sends
 * on its own.
 */
export function autoAdvanceTo(questions: Question[], current: number): number | null {
  const q = questions[current];
  if (!q || q.multiSelect || current >= lastStep(questions)) return null;
  return current + 1;
}

/**
 * The card's keys: ←/→ move between steps, 1–9 pick an option of the current
 * question. Enter presses the primary button (Next or "Submit answers"), only
 * with focus on the card itself, so a focused button keeps its own Enter.
 */
export function questionKeyAction(key: string, questions: Question[], current: number, onCard = false): QuestionMove | { kind: "pick"; label: string } | { kind: "primary" } | null {
  if (key === "Enter") return onCard ? { kind: "primary" } : null;
  if (key === "ArrowLeft") return current > 0 ? { kind: "go", index: current - 1 } : null;
  if (key === "ArrowRight") return current < lastStep(questions) ? { kind: "go", index: current + 1 } : null;
  if (/^[1-9]$/.test(key)) {
    const option = questions[current]?.options[Number(key) - 1];
    return option ? { kind: "pick", label: option.label } : null;
  }
  return null;
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
 * failed) reads as what was asked: "Merge upe-1", not "Merged upe-1". So does
 * a call waiting for approval. A running call reads "Merging upe-1", and so
 * does a `live` one: the active group's line, which is still working.
 */
export function stepSummary(step: ToolStep, live = false): { text: string; detail: string } {
  const waiting = !!step.approval && !step.approval.decision;
  const ran = !waiting && !userDenied(step) && step.status !== "error" && step.status !== "denied";
  if (ran) return toolSummary(step.name, step.arg, live || !step.done);
  const key = toolLabel(step.name);
  if (!TOOL_VERBS[key]) return { text: key, detail: toolDetail(key, step.arg) };
  const { action, detail } = approvalTitle(step.name, step.arg);
  return { text: action, detail };
}

/**
 * A work-list row: its icon, then its `verb`, then the detail. An unknown tool
 * gets the generic icon and its `name` in place of a verb. A shell call's
 * detail is its description, a sentence of its own, so it has no verb.
 * `live` keeps the progressive verb for the active group's line.
 */
export function stepIcon(step: ToolStep, live = false): { icon: ToolIconId; verb?: string; name?: string; detail: string } {
  const { text, detail } = stepSummary(step, live);
  const known = TOOL_VERBS[toolLabel(step.name)];
  if (!known) return { icon: "tool", name: text, detail };
  if (step.name === "Bash" && detail) return { icon: known.icon, detail };
  return { icon: known.icon, verb: text, detail };
}

function argText(v: unknown): string {
  if (typeof v === "string") return v;
  if (Array.isArray(v) && v.every((x) => typeof x !== "object" || x === null)) return v.join(", ");
  return typeof v === "object" ? JSON.stringify(v) : String(v);
}

/**
 * The call an approval card shows under its headline: the command of a shell
 * call, and every other non-empty argument.
 */
export function approvalArgs(name: string, input: string | undefined): { command?: string; fields: [string, string][] } {
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

/**
 * The turn whose user message the user can edit: the last one, once the chat
 * is idle and the agent can rewind its session there. Claude resumes at the
 * anchor of the turn before (the first message starts a new session); Codex
 * drops the edited turn by its own anchor. Null when there is none.
 */
export function editTarget(state: ChatThread, agent: string): number | null {
  const turns = state.turns;
  if (turns.length === 0 || turns[turns.length - 1].status === "running") return null;
  const i = lastIndexWhere(turns, (t) => !t.continued && !t.byAgent);
  if (i < 0) return null;
  if (agent === "codex") return turns[i].anchor ? i : null;
  return i === 0 || turns[i - 1].anchor ? i : null;
}

/** The indices of the turns that start with a user message: the turn rail's bars. */
export function userTurns(state: ChatThread): number[] {
  return state.turns.flatMap((t, i) => (t.continued || t.byAgent ? [] : [i]));
}

/** The user messages in view: those whose span overlaps it. Returns indices into spans. */
export function visibleTurns(spans: { top: number; bottom: number }[], viewTop: number, viewBottom: number): number[] {
  return spans.flatMap(({ top, bottom }, i) => (top < viewBottom && bottom > viewTop ? [i] : []));
}
