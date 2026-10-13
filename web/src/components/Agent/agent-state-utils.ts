import type { AgentStatus, ChatContext, ChatMeta } from "../../api/chat";
import type { DropdownOption } from "../ui";
import type { Issue } from "../../api/types";
import { VIEW_BY_ID, type ViewId } from "../../views";

const PILLS_KEY = "xpo.agent.pills";
const PREFS_KEY = "xpo.agent.prefs";
const POPUP_SIZE_KEY = "xpo.agent.popupSize";

/** The browser's localStorage, or null when it is blocked. */
export function safeLocalStorage(): Storage | null {
  try { return window.localStorage; } catch { return null; }
}

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

/**
 * How often to refetch the chat list: every 2s while any chat runs, so pills
 * and issue chat icons show its state; otherwise not at all.
 */
export function chatListPollMs(chats: { state: string }[] | undefined): number | false {
  return chats?.some((c) => c.state === "running") ? 2000 : false;
}

export interface PopupSize {
  width: number;
  height: number;
}

export const DEFAULT_POPUP_SIZE: PopupSize = { width: 460, height: 600 };
const MIN_POPUP: PopupSize = { width: 360, height: 400 };
const MAX_POPUP: PopupSize = { width: 600, height: 900 };

/**
 * A popup size within 360x400 to 600x900 that also fits the viewport: 12px
 * margins at the sides, and 64px of height for the status bar and a gap.
 */
export function clampPopupSize(size: PopupSize, viewport: PopupSize): PopupSize {
  const clamp = (v: number, min: number, max: number) => Math.max(min, Math.min(v, max));
  return {
    width: clamp(size.width, MIN_POPUP.width, Math.min(MAX_POPUP.width, viewport.width - 24)),
    height: clamp(size.height, MIN_POPUP.height, Math.min(MAX_POPUP.height, viewport.height - 64)),
  };
}

/** The popup size the viewer last chose; the default without one. */
export function loadPopupSize(storage: Storage | null): PopupSize {
  const v = readJSON(storage, POPUP_SIZE_KEY) as Partial<PopupSize> | null;
  return v && typeof v.width === "number" && typeof v.height === "number" ? { width: v.width, height: v.height } : DEFAULT_POPUP_SIZE;
}

export function savePopupSize(storage: Storage | null, size: PopupSize): void {
  writeJSON(storage, POPUP_SIZE_KEY, size);
}

/**
 * The left edge of a popup of width centered over an anchor whose center is
 * at anchorCenter, kept margin px inside a viewport of viewportWidth.
 */
export function popupLeft(anchorCenter: number, width: number, viewportWidth: number, margin = 12): number {
  const max = viewportWidth - width - margin;
  return Math.max(margin, Math.min(anchorCenter - width / 2, max));
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

/**
 * The prefs for a chat's composer: a pick made in this chat, else what the
 * chat's last message used (saved in its meta), else the remembered prefs.
 * A new chat, or one with no message yet, uses the remembered prefs.
 */
export function chatPrefs(meta: ChatMeta | undefined, picked: ChatPrefs | undefined, saved: ChatPrefs): ChatPrefs {
  if (picked) return picked;
  if (!meta?.agent) return saved;
  return { provider: meta.agent, model: meta.model ?? "", effort: meta.effort ?? "" };
}

/** Prefs that the provider list allows: unknown values fall back to the defaults. */
export function normalizePrefs(prefs: ChatPrefs, status: AgentStatus | undefined): ChatPrefs {
  if (!status) return prefs;
  const provider = status.providers.find((p) => p.available && p.id === prefs.provider) ?? status.providers.find((p) => p.id === status.default_provider);
  if (!provider) return { provider: "", model: "", effort: "" };
  const model = provider.models.find((m) => m.id === prefs.model) ?? provider.models.find((m) => m.id === "");
  const effort = pickEffort(prefs.effort, model?.efforts ?? []);
  const next = { provider: provider.id, model: model?.id ?? "", effort };
  return next.provider === prefs.provider && next.model === prefs.model && next.effort === prefs.effort ? prefs : next;
}

/** The effort to use from a model's levels: the wanted one if offered, else Medium, else the first. */
function pickEffort(wanted: string, efforts: string[]): string {
  if (efforts.includes(wanted)) return wanted;
  return efforts.includes("medium") ? "medium" : (efforts[0] ?? "");
}

/** Prefs for another provider: its default model. withModel then checks the effort. */
export function withProvider(prefs: ChatPrefs, provider: string): ChatPrefs {
  return provider === prefs.provider ? prefs : { ...prefs, provider, model: "" };
}

/** Prefs for another model, keeping the effort if the model supports it. */
export function withModel(prefs: ChatPrefs, model: string, status: AgentStatus | undefined): ChatPrefs {
  const efforts = status?.providers.find((p) => p.id === prefs.provider)?.models.find((m) => m.id === model)?.efforts ?? [];
  return { ...prefs, model, effort: pickEffort(prefs.effort, efforts) };
}

function effortLabel(effort: string): string {
  return EFFORT_LABELS[effort] ?? effort.charAt(0).toUpperCase() + effort.slice(1);
}

/** One provider's models in the model menu. */
export interface ModelGroup {
  provider: string;
  label: string;
  /** Another provider's group, once the chat has started. */
  disabled: boolean;
  options: DropdownOption[];
}

/** The model menu's value for a provider's model. */
export function modelKey(provider: string, model: string): string {
  return `${provider}/${model}`;
}

/** Prefs for a model picked from the menu, which may belong to another provider. */
export function pickModel(prefs: ChatPrefs, key: string, status: AgentStatus | undefined): ChatPrefs {
  const i = key.indexOf("/");
  const provider = key.slice(0, i);
  return withModel(withProvider(prefs, provider), key.slice(i + 1), status);
}

/** A context window in tokens as "200K" or "1M"; "" when unknown. */
export function formatContextWindow(tokens: number | undefined): string {
  if (!tokens) return "";
  if (tokens >= 1_000_000) return `${Math.round(tokens / 100_000) / 10}M`;
  return `${Math.round(tokens / 1000)}K`;
}

/** The composer's picker options and trigger labels for the picked provider and model. */
export function composerOptions(status: AgentStatus | undefined, prefs: ChatPrefs, providerLocked: boolean): {
  groups: ModelGroup[];
  model: string;
  modelLabel: string;
  efforts: DropdownOption[];
  effortLabel: string;
} {
  const provider = status?.providers.find((p) => p.id === prefs.provider);
  const model = provider?.models.find((m) => m.id === prefs.model);
  const context = formatContextWindow(model?.context_window);
  return {
    groups: (status?.providers ?? []).filter((p) => p.available).map((p) => ({
      provider: p.id,
      label: p.name,
      disabled: providerLocked && p.id !== prefs.provider,
      options: p.models.map((m) => ({
        value: modelKey(p.id, m.id),
        label: m.id === "" && m.name ? `${m.name} (Default)` : m.name || m.label,
      })),
    })),
    model: modelKey(prefs.provider, prefs.model),
    modelLabel: model ? model.name || model.label : "Model",
    efforts: (model?.efforts ?? []).map((e) => ({ value: e, label: effortLabel(e) })),
    effortLabel: !prefs.effort ? "Effort" : context ? `${effortLabel(prefs.effort)} · ${context}` : effortLabel(prefs.effort),
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

/** The last issue the user opened; seq counts the times they opened one. */
export interface ViewedIssue {
  id?: string;
  seq: number;
}

/** Opening a different issue counts; views without an issue keep the last one. */
export function nextViewed(v: ViewedIssue, issueId: string | undefined): ViewedIssue {
  return issueId && issueId !== v.id ? { id: issueId, seq: v.seq + 1 } : v;
}

/**
 * The issue a chat's next message carries as context, shown as the
 * composer's chip: the last opened issue, until the chat sends or removes
 * it (consumed is the seq it did so at). A Work Mode chat never gets its
 * bound issue: the mode names it already.
 */
export function pendingContextIssue(viewed: ViewedIssue, consumed: number | undefined, workIssue: string | undefined): string | undefined {
  return viewed.id && viewed.seq > (consumed ?? 0) && viewed.id !== workIssue ? viewed.id : undefined;
}

// The new chat, which has no ID yet, keeps its record under this key.
const NEW_CHAT = "";

/** By chat ID, the viewed issue seq up to which a chat sent or removed its context. */
export type ConsumedContext = Record<string, number>;

export function consumedFor(c: ConsumedContext, chatId: string | null): number | undefined {
  return c[chatId ?? NEW_CHAT];
}

export function consumeContext(c: ConsumedContext, chatId: string | null, seq: number): ConsumedContext {
  return { ...c, [chatId ?? NEW_CHAT]: seq };
}

/** Moves the new chat's record to the chat once it exists. */
export function adoptNewChatContext(c: ConsumedContext, id: string): ConsumedContext {
  if (!(NEW_CHAT in c)) return c;
  const { [NEW_CHAT]: seq, ...rest } = c;
  return { ...rest, [id]: seq };
}

const EFFORT_LABELS: Record<string, string> = {
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
