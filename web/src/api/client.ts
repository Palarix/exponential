import type { Issue, Dependency, Comment, PendingState, Event } from './types';

const API_BASE = '/api';

export type { Issue, Dependency, Comment, PendingState, Event };

export async function fetchIssues(): Promise<Issue[]> {
  const res = await fetch(`${API_BASE}/issues`);
  if (!res.ok) throw new Error('Failed to fetch issues');
  return res.json();
}

export async function fetchPending(): Promise<PendingState> {
  const res = await fetch(`${API_BASE}/pending`);
  if (!res.ok) throw new Error('Failed to fetch pending state');
  return res.json();
}

export async function addDraft(issueId: string, type: string, payload: unknown): Promise<void> {
  const res = await fetch(`${API_BASE}/draft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ issue_id: issueId, type, payload }),
  });
  if (!res.ok) throw new Error('Failed to add draft');
}

export async function createIssue(payload: { title: string; description?: string; labels?: string[]; parent_id?: string }): Promise<string> {
  const res = await fetch(`${API_BASE}/draft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ issue_id: '', type: 'CREATE', payload }),
  });
  if (!res.ok) throw new Error('Failed to create issue');
  const data = await res.json();
  return data.issue_id;
}

export interface HistoryEvent {
  type: string;
  payload: Record<string, unknown>;
  created_at: string;
  created_by: string;
}

export async function fetchIssueHistory(issueId: string): Promise<HistoryEvent[]> {
  const res = await fetch(`${API_BASE}/issues/${issueId}/history`);
  if (!res.ok) throw new Error('Failed to fetch history');
  return res.json();
}

export async function saveAll(message?: string): Promise<void> {
  const res = await fetch(`${API_BASE}/save`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: message || '' }),
  });
  if (!res.ok) throw new Error('Failed to save');
}

export async function fetchConfig(): Promise<{ auto_commit: boolean; prefix: string; version: string; labels: Record<string, string>; name: string; hide_default_labels: boolean }> {
  const res = await fetch(`${API_BASE}/config`);
  if (!res.ok) throw new Error('Failed to fetch config');
  return res.json();
}

export async function addConfigLabel(name: string, color: string): Promise<void> {
  const res = await fetch(`${API_BASE}/config/labels`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, color }),
  });
  if (!res.ok) throw new Error('Failed to add label');
}

export async function updateConfigLabel(oldName: string, newName: string, color: string): Promise<void> {
  const res = await fetch(`${API_BASE}/config/labels`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ old_name: oldName, new_name: newName, color }),
  });
  if (!res.ok) throw new Error('Failed to update label');
}

export async function deleteConfigLabel(name: string): Promise<void> {
  const res = await fetch(`${API_BASE}/config/labels`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) throw new Error('Failed to delete label');
}

export async function discardAll(): Promise<void> {
  const res = await fetch(`${API_BASE}/pending`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to discard');
}
