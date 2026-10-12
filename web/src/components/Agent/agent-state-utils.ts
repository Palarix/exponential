import { CHAT_EFFORTS, CHAT_MODELS, type ChatContext, type ChatEffort, type ChatModel } from "../../api/chat";
import type { Issue } from "../../api/types";
import { VIEW_BY_ID, type ViewId } from "../../views";

const PILLS_KEY = "xpo.agent.pills";
const PREFS_KEY = "xpo.agent.prefs";

function readJSON(storage: Storage | null, key: string): unknown {
  try {
    const raw = storage?.getItem(key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function writeJSON(storage: Storage | null, key: string, value: unknown): void {
  try {
    storage?.setItem(key, JSON.stringify(value));
  } catch {
    // Storage is a convenience: blocked or full storage just forgets.
  }
}

/** Chat IDs shown as pills in the status bar, oldest first. */
export function loadPills(storage: Storage | null): string[] {
  const value = readJSON(storage, PILLS_KEY);
  return Array.isArray(value) ? value.filter((v): v is string => typeof v === "string") : [];
}

export function savePills(storage: Storage | null, pills: string[]): void {
  writeJSON(storage, PILLS_KEY, pills);
}

export function addPill(pills: string[], id: string): string[] {
  return pills.includes(id) ? pills : [...pills, id];
}

export function removePill(pills: string[], id: string): string[] {
  return pills.filter((p) => p !== id);
}

export interface ChatPrefs {
  model: ChatModel;
  effort: ChatEffort;
}

/** The last model and effort picked, used for new chats. */
export function loadPrefs(storage: Storage | null): ChatPrefs {
  const value = readJSON(storage, PREFS_KEY) as Partial<Record<keyof ChatPrefs, unknown>> | null;
  const model = CHAT_MODELS.find((m) => m === value?.model) ?? "";
  const effort = CHAT_EFFORTS.find((e) => e === value?.effort) ?? "";
  return { model, effort };
}

export function savePrefs(storage: Storage | null, prefs: ChatPrefs): void {
  writeJSON(storage, PREFS_KEY, prefs);
}

/** What the user has open, sent with each message. */
export function chatContextFor(view: ViewId, issueId: string | null, issues: Issue[]): ChatContext {
  const ctx: ChatContext = { view: VIEW_BY_ID[view].title };
  if (!issueId) return ctx;
  ctx.issue_id = issueId;
  const title = issues.find((i) => i.id === issueId)?.title;
  if (title) ctx.issue_title = title;
  return ctx;
}

export const MODEL_LABELS: Record<ChatModel, string> = { "": "Default model", haiku: "Haiku", sonnet: "Sonnet", opus: "Opus" };
export const EFFORT_LABELS: Record<ChatEffort, string> = {
  "": "Default effort",
  low: "Low",
  medium: "Medium",
  high: "High",
  xhigh: "Extra high",
  max: "Max",
};

export const SUGGESTIONS = [
  { label: "File a bug", prompt: "Help me file a bug. Ask me what's wrong, then create the issue." },
  { label: "What's blocking an epic?", prompt: "Which open epics have blocked or stalled sub-issues, and what is blocking them?" },
  { label: "Summarize my open issues", prompt: "Summarize my open issues by status, and point out anything that looks stale." },
] as const;
