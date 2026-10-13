// Agent chat API. The board runs the agent on the viewer's machine, so these
// routes live under /api/local/ (answered by xpo board, never proxied).
import { request } from "./client";

const BASE = "/api/local";

export interface ChatContext {
  view?: string;
  issue_id?: string;
  issue_title?: string;
}

export type ChatEventType =
  | "user_message"
  | "assistant_text"
  | "tool_call"
  | "turn_done"
  | "turn_stopped"
  | "error"
  | "approval_requested"
  | "approval_resolved"
  | "mode_changed"
  | "text_delta"
  | "thinking"
  | "tool_start";

/** One entry of a chat's event stream; `seq` is set on settled events only. */
export interface ChatEvent {
  seq?: number;
  type: ChatEventType;
  time: string;
  text?: string;
  context?: ChatContext;
  name?: string;
  arg?: string;
  /** tool_start, tool_call; missing in chats saved before tool results. */
  tool_id?: string;
  /** tool_call: the outcome. */
  status?: ToolCallStatus;
  duration_ms?: number;
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  /** error; tool_call: the error or denial text. */
  message?: string;
  /** approval_requested, approval_resolved. */
  approval_id?: string;
  /** approval_requested: the full tool input as JSON. */
  input?: string;
  /** approval_resolved. */
  decision?: ApprovalOutcome;
  /** mode_changed: the new mode. */
  mode?: ChatMode;
  /** mode_changed: the issue entered or left. */
  issue_id?: string;
}

/** Chat Mode reads the repo and changes issues; Work Mode implements one issue in its worktree. */
export type ChatMode = "chat" | "work";

export type ToolCallStatus = "done" | "error" | "denied";

/** What the user can answer on an approval card. */
export type ApprovalDecision = "allow" | "allow_session" | "deny";

/** How an approval card ended: a decision, or the turn ended first. */
export type ApprovalOutcome = ApprovalDecision | "cancelled";

export interface ChatMeta {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  agent: string;
  model?: string;
  effort?: string;
  state: "idle" | "running" | "interrupted";
  /** An approval card waits for the user. */
  waiting?: boolean;
  /** Missing means Chat Mode. */
  mode?: ChatMode;
  /** The issue a chat in Work Mode works on. */
  work?: { issue_id: string; worktree: string };
  /** Every issue the chat entered Work Mode on. */
  worked_issues?: string[];
}

/** A model the composer offers; "" means the provider's default. */
export interface ModelOption {
  id: string;
  label: string;
  /** The full name, such as "Claude Opus 5.5", once a turn has used the model. */
  name?: string;
  /** The context window in tokens, once a turn has used the model. */
  context_window?: number;
  /** Effort levels the model supports; "" means the default. */
  efforts: string[];
}

/** An agent CLI chat can run, such as Claude Code or Codex. */
export interface ProviderStatus {
  id: string;
  name: string;
  available: boolean;
  /** Why the provider is not available. */
  reason?: string;
  models: ModelOption[];
}

export interface AgentStatus {
  /** At least one provider is available. */
  available: boolean;
  default_provider: string;
  /** Why no provider is available. */
  reason?: string;
  providers: ProviderStatus[];
}

export interface SendChatMessage {
  text: string;
  context?: ChatContext;
  provider: string;
  model: string;
  effort: string;
}

export function fetchAgentStatus(): Promise<AgentStatus> {
  return request(`${BASE}/agent`);
}

export function fetchChats(): Promise<ChatMeta[]> {
  return request(`${BASE}/chats`);
}

export function fetchChat(id: string): Promise<{ meta: ChatMeta; events: ChatEvent[]; running: boolean }> {
  return request(`${BASE}/chats/${encodeURIComponent(id)}`);
}

export function createChat(): Promise<ChatMeta> {
  return request(`${BASE}/chats`, { method: "POST" });
}

export function sendChatMessage(id: string, msg: SendChatMessage): Promise<void> {
  return request(`${BASE}/chats/${encodeURIComponent(id)}/messages`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(msg),
  });
}

export function cancelChat(id: string): Promise<void> {
  return request(`${BASE}/chats/${encodeURIComponent(id)}/cancel`, { method: "POST" });
}

/** Answers an approval card. A 404 means the card is no longer open. */
export function decideApproval(chatId: string, approvalId: string, decision: ApprovalDecision): Promise<void> {
  return request(`${BASE}/chats/${encodeURIComponent(chatId)}/approvals/${encodeURIComponent(approvalId)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ decision }),
  });
}

export function chatStreamURL(id: string): string {
  return `${BASE}/chats/${encodeURIComponent(id)}/stream`;
}

export function renameChat(id: string, title: string): Promise<ChatMeta> {
  return request(`${BASE}/chats/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  });
}

export function deleteChat(id: string): Promise<void> {
  return request(`${BASE}/chats/${encodeURIComponent(id)}`, { method: "DELETE" });
}

/**
 * Start Work in Work Mode: opens the chat in Work Mode on the issue, or
 * starts the issue and creates one. `created` is false for an existing chat.
 */
export function startWorkChat(issueId: string): Promise<{ chat: ChatMeta; created: boolean }> {
  return request(`${BASE}/issues/${encodeURIComponent(issueId)}/work-chat`, { method: "POST" });
}

/** Puts a chat back in Chat Mode without a merge. The issue and its worktree stay. */
export function leaveWork(chatId: string): Promise<ChatMeta> {
  return request(`${BASE}/chats/${encodeURIComponent(chatId)}/leave-work`, { method: "POST" });
}
