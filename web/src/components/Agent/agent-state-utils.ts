import type { AgentStatus, ChatContext } from "../../api/chat";
import type { DropdownOption } from "../ui";
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

/** The provider, model and effort picked in the composer; "" means the default. */
export interface ChatPrefs {
  provider: string;
  model: string;
  effort: string;
}

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}

/**
 * The last provider, model and effort picked, used for new chats. They are
 * checked against the provider list with normalizePrefs. Prefs saved before
 * providers existed were for Claude.
 */
export function loadPrefs(storage: Storage | null): ChatPrefs {
  const value = readJSON(storage, PREFS_KEY) as Partial<Record<keyof ChatPrefs, unknown>> | null;
  if (!value || typeof value !== "object") return { provider: "", model: "", effort: "" };
  const provider = "provider" in value ? str(value.provider) : str(value.model) || str(value.effort) ? "claude" : "";
  return { provider, model: str(value.model), effort: str(value.effort) };
}

export function savePrefs(storage: Storage | null, prefs: ChatPrefs): void {
  writeJSON(storage, PREFS_KEY, prefs);
}

/** Prefs that the provider list allows: unknown values fall back to the defaults. */
export function normalizePrefs(prefs: ChatPrefs, status: AgentStatus | undefined): ChatPrefs {
  if (!status) return prefs;
  const provider = status.providers.find((p) => p.available && p.id === prefs.provider) ?? status.providers.find((p) => p.id === status.default_provider);
  if (!provider) return { provider: "", model: "", effort: "" };
  const model = provider.models.find((m) => m.id === prefs.model) ?? provider.models.find((m) => m.id === "");
  const effort = model?.efforts.includes(prefs.effort) ? prefs.effort : "";
  const next = { provider: provider.id, model: model?.id ?? "", effort };
  return next.provider === prefs.provider && next.model === prefs.model && next.effort === prefs.effort ? prefs : next;
}

/** Prefs for another provider: its default model and effort. */
export function withProvider(prefs: ChatPrefs, provider: string): ChatPrefs {
  return provider === prefs.provider ? prefs : { provider, model: "", effort: "" };
}

/** Prefs for another model, keeping the effort if the model supports it. */
export function withModel(prefs: ChatPrefs, model: string, status: AgentStatus | undefined): ChatPrefs {
  const efforts = status?.providers.find((p) => p.id === prefs.provider)?.models.find((m) => m.id === model)?.efforts ?? [""];
  return { ...prefs, model, effort: efforts.includes(prefs.effort) ? prefs.effort : "" };
}

function effortLabel(effort: string): string {
  return EFFORT_LABELS[effort] ?? effort.charAt(0).toUpperCase() + effort.slice(1);
}

/** The composer's dropdown options for the picked provider and model. */
export function composerOptions(status: AgentStatus | undefined, prefs: ChatPrefs): {
  providers: DropdownOption[];
  models: DropdownOption[];
  efforts: DropdownOption[];
} {
  const provider = status?.providers.find((p) => p.id === prefs.provider);
  const model = provider?.models.find((m) => m.id === prefs.model);
  return {
    providers: (status?.providers ?? []).filter((p) => p.available).map((p) => ({ value: p.id, label: p.name })),
    models: (provider?.models ?? []).map((m) => ({ value: m.id, label: m.label })),
    efforts: (model?.efforts ?? []).map((e) => ({ value: e, label: effortLabel(e) })),
  };
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

const EFFORT_LABELS: Record<string, string> = {
  "": "Default effort",
  low: "Low",
  medium: "Medium",
  high: "High",
  xhigh: "Extra high",
  max: "Max",
};

export const SUGGESTIONS = [
  {
    label: "File a bug",
    description: "Describe the problem and get a well-formed issue",
    prompt: "Help me file a bug. Ask me what's wrong, then create the issue.",
  },
  {
    label: "What's blocking an epic?",
    description: "Find blocked or stalled sub-issues across epics",
    prompt: "Which open epics have blocked or stalled sub-issues, and what is blocking them?",
  },
  {
    label: "Summarize my open issues",
    description: "Group your issues by status and spot stale work",
    prompt: "Summarize my open issues by status, and point out anything that looks stale.",
  },
] as const;
