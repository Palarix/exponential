export interface BranchStats {
  branch: string;
  head_sha: string;
  commits: number;
  files_changed: number;
  insertions: number;
  deletions: number;
  has_uncommitted: boolean;
}

export interface ArtifactSummary {
  artifact_type: string;
  filename: string;
  updated_at: string;
  updated_by: string;
}

export interface Issue {
  id: string;
  title: string;
  description: string;
  status: string;
  parent_id?: string;
  estimate: number;
  priority: number;
  sort_order: string;
  assignee?: string;
  cycle_id?: string;
  effective_cycle_id?: string;
  labels?: string[];
  dependencies?: Dependency[];
  artifacts?: ArtifactSummary[];
  comments?: Comment[];
  created_at: string;
  created_by: string;
  updated_at: string;
  is_inferred?: boolean;
  is_pending?: boolean;
  branch_stats?: BranchStats;
}

export interface Cycle {
  id: string;
  number: number;
  start: string;
  end: string;
  status: 'completed' | 'current' | 'upcoming' | 'planned';
  done: number;
  total: number;
}

export interface CyclesResponse {
  enabled: boolean;
  cycles: Cycle[];
}

export interface CycleProgressDay {
  date: string;
  scope: number;
  started: number;
  completed: number;
}

export interface CycleProgressResponse {
  days: CycleProgressDay[];
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
  has_pending: boolean;
  events: Event[];
  issue_ids: string[];
}

export interface Event {
  issue_id: string;
  type: string;
  payload?: Record<string, unknown>;
  created_at: string;
}

export interface InboxItem {
  issue_id: string;
  issue_title: string;
  type: 'CREATE' | 'UPDATE' | 'COMMENT' | 'MERGE' | 'DELETE' | 'ARTIFACT';
  payload: Record<string, unknown>;
  created_at: string;
  created_by: string;
  on_behalf_of?: string;
}

export interface InboxStatus {
  last_read: string;
  unread: number;
}

export interface CommitDetail {
  sha: string;
  subject: string;
  body?: string;
  author: string;
  date: string;
  files: CommitFile[];
}

export interface CommitFile {
  path: string;
  additions: number;
  deletions: number;
}

export interface TimelineEntry {
  kind: 'issue_event' | 'commit';
  timestamp: string;
  issue_id: string;
  issue_title?: string;
  event_type?: string;
  payload?: Record<string, unknown>;
  created_by?: string;
  on_behalf_of?: string;
  source?: string;
  sha?: string;
  message?: string;
  author?: string;
  branch?: string;
}
