// API client for beats server

const API_BASE = '/api';

export interface Issue {
  id: string;
  kind: string;
  title: string;
  description: string;
  status: string;
  parent_id?: string;
  estimate: number;
  logged_effort: number;
  labels: string[];
  checklist?: ChecklistItem[];
  dependencies?: Dependency[];
  comments?: Comment[];
  created_at: string;
  updated_at: string;
  is_pending?: boolean;
}

export interface ChecklistItem {
  title: string;
  state: string;
}

export interface Dependency {
  source_id: string;
  target_id: string;
  kind: string;
}

export interface Comment {
  id: string;
  text: string;
  created_by: string;
  created_at: string;
}

export interface PendingState {
  count: number;
  events?: Event[];
}

export interface Event {
  id: string;
  type: string;
  payload: unknown;
  created_at: string;
  created_by: string;
}

// Fetch all issues
export async function getIssues(): Promise<Issue[]> {
  const response = await fetch(`${API_BASE}/issues`);
  if (!response.ok) {
    throw new Error('Failed to fetch issues');
  }
  return response.json();
}

// Fetch a single issue
export async function getIssue(id: string): Promise<Issue> {
  const response = await fetch(`${API_BASE}/issues/${id}`);
  if (!response.ok) {
    throw new Error('Failed to fetch issue');
  }
  return response.json();
}

// Get pending state
export async function getPending(): Promise<PendingState> {
  const response = await fetch(`${API_BASE}/pending`);
  if (!response.ok) {
    throw new Error('Failed to fetch pending state');
  }
  return response.json();
}

// Add a draft event
export async function addDraft(
  issueId: string,
  type: string,
  payload: unknown
): Promise<{ success: boolean; event_id: string; pending: number }> {
  const response = await fetch(`${API_BASE}/draft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ issue_id: issueId, type, payload }),
  });
  if (!response.ok) {
    throw new Error('Failed to add draft');
  }
  return response.json();
}

// Save and sync pending events
export async function save(commitMessage: string): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/save`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ commit_message: commitMessage }),
  });
  if (!response.ok) {
    throw new Error('Failed to save');
  }
  return response.json();
}

// Discard pending events
export async function discardPending(): Promise<{ success: boolean; message: string }> {
  const response = await fetch(`${API_BASE}/pending`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to discard');
  }
  return response.json();
}
