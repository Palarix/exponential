import type { Issue, Dependency, Comment, PendingState, Event, Cycle, CyclesResponse, CycleProgressDay, CycleProgressResponse, InboxItem, InboxStatus, TimelineEntry, CommitDetail, CommitFile } from './types';

const API_BASE = '/api';

export type { Issue, Dependency, Comment, PendingState, Event, Cycle, CyclesResponse, CycleProgressDay, CycleProgressResponse, InboxItem, InboxStatus, TimelineEntry, CommitDetail, CommitFile };

export class ApiError extends Error {
  status: number;
  statusText: string;
  body: string;

  constructor(res: Response, body: string) {
    const msg = body || `${res.status} ${res.statusText}`;
    super(msg);
    this.name = 'ApiError';
    this.status = res.status;
    this.statusText = res.statusText;
    this.body = body;
  }
}

async function request<T = void>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res, body);
  }
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function fetchIssues(): Promise<Issue[]> {
  return request(`${API_BASE}/issues`);
}

export async function fetchPending(): Promise<PendingState> {
  return request(`${API_BASE}/pending`);
}

export async function addDraft(issueId: string, type: string, payload: unknown): Promise<void> {
  return request(`${API_BASE}/draft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ issue_id: issueId, type, payload }),
  });
}

export async function createIssue(payload: { title: string; description?: string; labels?: string[]; parent_id?: string; assignee?: string; sort_order?: string }): Promise<string> {
  const data = await request<{ issue_id: string }>(`${API_BASE}/draft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ issue_id: '', type: 'CREATE', payload }),
  });
  return data.issue_id;
}

export interface HistoryEvent {
  type: string;
  payload: Record<string, unknown>;
  created_at: string;
  created_by: string;
  on_behalf_of?: string;
}

export async function fetchIssueHistory(issueId: string): Promise<HistoryEvent[]> {
  return request(`${API_BASE}/issues/${issueId}/history`);
}

export async function fetchArtifactContent(issueId: string, filename: string): Promise<string> {
  const res: { content: string } = await request(`${API_BASE}/issues/${issueId}/artifacts/${encodeURIComponent(filename)}`);
  return res.content;
}

export async function saveAll(message?: string): Promise<void> {
  return request(`${API_BASE}/save`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: message || '' }),
  });
}

export interface AttentionItem {
  issue_id: string;
  title: string;
  kind: 'blocker' | 'stale_wip' | 'high_priority';
  age_days?: number;
  priority?: number;
}

export interface WorkloadEntry {
  assignee: string;
  in_progress: number;
  open_points: number;
  blocked: number;
  last_completed?: string;
}

export interface ActivityEvent {
  issue_id: string;
  issue_title: string;
  type: 'CREATE' | 'UPDATE' | 'COMMENT' | 'MERGE' | 'ARTIFACT';
  payload: Record<string, unknown>;
  created_at: string;
  created_by: string;
  on_behalf_of?: string;
}

export async function fetchActivity(): Promise<ActivityEvent[]> {
  return request(`${API_BASE}/activity`);
}

export async function fetchTimeline(limit?: number, kind?: string): Promise<TimelineEntry[]> {
  const params = new URLSearchParams();
  if (limit) params.set('limit', String(limit));
  if (kind) params.set('kind', kind);
  const qs = params.toString();
  return request(`${API_BASE}/timeline${qs ? '?' + qs : ''}`);
}

export async function fetchCommitDetail(sha: string): Promise<CommitDetail> {
  return request(`${API_BASE}/commits/${sha}`);
}

export interface EpicProgress {
  issue_id: string;
  title: string;
  children_done: number;
  children_total: number;
  points_done: number;
  points_total: number;
  points_remaining: number;
  stale: boolean;
}

export interface PulseMetrics {
  velocity: {
    current_week_points: number;
    last_7d_points: number;
    prior_7d_points: number;
    delta: number;
    weekly_buckets: { week_start: string; points: number }[];
    daily_buckets: { date: string; points: number }[];
  };
  throughput: {
    last_7d: number;
    prior_7d: number;
    delta: number;
  };
  wip: {
    total: number;
    stale: number;
    stale_threshold_days: number;
  };
  blockers: {
    total: number;
    oldest_days: number;
  };
  flow: {
    cycle_time_hrs: number;
    cycle_time_p75_hrs: number;
    cycle_time_p90_hrs: number;
    cycle_time_min_hrs: number;
    cycle_time_max_hrs: number;
    cycle_count: number;
    lead_time_hrs: number;
    lead_time_p75_hrs: number;
    lead_time_p90_hrs: number;
    lead_time_min_hrs: number;
    lead_time_max_hrs: number;
    lead_count: number;
    staleness: { under_1d: number; under_3d: number; under_7d: number; under_14d: number; under_30d: number; over_30d: number };
    staleness_total: number;
  };
  attention: AttentionItem[];
  workload: WorkloadEntry[];
  epics: EpicProgress[];
  trends: {
    weekly: { week_start: string; created: number; completed: number }[];
    median_triage_mins: number;
    triaged_count: number;
    bug_age: {
      under_24h: number;
      under_48h: number;
      under_5d: number;
      under_14d: number;
      under_1mo: number;
      over_1mo: number;
    };
  };
}

export async function fetchMetrics(): Promise<PulseMetrics> {
  return request(`${API_BASE}/metrics`);
}

export interface Instance {
  name: string;
  port: number;
  pid: number;
  root_dir: string;
  started_at: string;
  is_current: boolean;
}

export async function fetchInstances(): Promise<Instance[]> {
  return request(`${API_BASE}/instances`);
}

export interface User {
  name: string;
  email: string;
}

export async function fetchUser(): Promise<User> {
  return request(`${API_BASE}/user`);
}

export async function fetchCycles(): Promise<CyclesResponse> {
  return request(`${API_BASE}/cycles`);
}

export async function fetchCycleProgress(cycleId: string): Promise<CycleProgressResponse> {
  return request(`${API_BASE}/cycles/${cycleId}/progress`);
}

export async function fetchConfig(): Promise<{ auto_commit: boolean; prefix: string; version: string; labels: Record<string, string>; name: string; hide_default_labels: boolean; default_labels: string[]; contributors?: string[]; cycles?: { enabled: boolean; duration: string; start_day: string; anchor_date: string } }> {
  return request(`${API_BASE}/config`);
}

export async function addConfigLabel(name: string, color: string): Promise<void> {
  return request(`${API_BASE}/config/labels`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, color }),
  });
}

export async function updateConfigLabel(oldName: string, newName: string, color: string): Promise<void> {
  return request(`${API_BASE}/config/labels`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ old_name: oldName, new_name: newName, color }),
  });
}

export async function deleteConfigLabel(name: string): Promise<void> {
  return request(`${API_BASE}/config/labels`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
}

export async function discardAll(): Promise<void> {
  return request(`${API_BASE}/pending`, { method: 'DELETE' });
}

export async function fetchInbox(): Promise<InboxItem[]> {
  return request(`${API_BASE}/inbox`);
}

export async function fetchInboxStatus(): Promise<InboxStatus> {
  return request(`${API_BASE}/inbox/status`);
}

export async function markInboxRead(): Promise<{ last_read: string }> {
  return request(`${API_BASE}/inbox/read`, { method: 'POST' });
}

export interface StartWorkResponse {
  status: string;
  issue_id: string;
  branch: string;
  messages: string[];
}

export async function startWork(issueId: string): Promise<StartWorkResponse> {
  return request(`${API_BASE}/issues/${issueId}/start`, { method: 'POST' });
}

export interface CommitInfo {
  sha: string;
  message: string;
  author: string;
  date: string;
}

export interface FileInfo {
  status: string;
  path: string;
  insertions: number;
  deletions: number;
}

export interface MergeResponse {
  status: string;
  merge_sha: string;
  messages: string[];
}

export async function fetchIssueCommits(issueId: string): Promise<CommitInfo[]> {
  return request(`${API_BASE}/issues/${issueId}/commits`);
}

export async function fetchIssueFiles(issueId: string): Promise<FileInfo[]> {
  return request(`${API_BASE}/issues/${issueId}/files`);
}

export async function fetchIssueDiff(issueId: string): Promise<string> {
  const res = await fetch(`${API_BASE}/issues/${issueId}/diff`);
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res, body);
  }
  return res.text();
}

export interface Mergeability {
  can_merge: boolean;
  blockers: string[];
}

export async function fetchMergeability(issueId: string): Promise<Mergeability> {
  return request(`${API_BASE}/issues/${issueId}/mergeability`);
}

export async function fetchCommitDiff(issueId: string, sha: string): Promise<string> {
  const res = await fetch(`${API_BASE}/issues/${issueId}/commits/${sha}/diff`);
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res, body);
  }
  return res.text();
}

export async function mergeIssue(issueId: string, options?: { strategy?: string; commit_message?: string; delete_branch?: boolean }): Promise<MergeResponse> {
  return request(`${API_BASE}/issues/${issueId}/merge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options || {}),
  });
}
