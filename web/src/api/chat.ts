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
  status?: string;
  duration_ms?: number;
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  message?: string;
}

export interface ChatMeta {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
  agent: string;
  model?: string;
  effort?: string;
  state: "idle" | "running" | "interrupted";
}

export interface AgentStatus {
  available: boolean;
  agent: string;
  model: string;
  reason?: string;
}

export const CHAT_MODELS = ["", "haiku", "sonnet", "opus"] as const;
export const CHAT_EFFORTS = ["", "low", "medium", "high", "xhigh", "max"] as const;
export type ChatModel = (typeof CHAT_MODELS)[number];
export type ChatEffort = (typeof CHAT_EFFORTS)[number];

export interface SendChatMessage {
  text: string;
  context?: ChatContext;
  model: ChatModel;
  effort: ChatEffort;
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
