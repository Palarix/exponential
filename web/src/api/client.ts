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

export async function saveAll(): Promise<void> {
  const res = await fetch(`${API_BASE}/save`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to save');
}

export async function discardAll(): Promise<void> {
  const res = await fetch(`${API_BASE}/discard`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to discard');
}
