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
  on_behalf_of?: string;
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
  // Agent doing the work for the assignee (principal-first assignees).
  assignee_via?: string;
  cycle_id?: string;
  effective_cycle_id?: string;
  release_id?: string;
  /** Display name of the release. */
  release?: string;
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
  // Set once the issue has a MERGE event.
  merged?: boolean;
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

export type ReleaseSlot = 'current' | 'next' | 'upcoming' | 'past';

/** A release on the board. A placeholder has a number but no ID; the first write saves it. */
export interface Release {
  id?: string;
  number: number;
  /** The label, or `#7` without one. */
  name: string;
  label?: string;
  goal?: string;
  state: 'OPEN' | 'RELEASED';
  slot?: ReleaseSlot;
  placeholder?: boolean;
  /** A Next placeholder that refuses writes until Current is saved. */
  locked?: boolean;
  sort_order?: string;
  target_date?: string;
  started_at?: string;
  released_at?: string;
  created_at?: string;
  /** Scope (leaf issues, not CANCELED/DUPLICATE) by status; a past release's snapshot. */
  counts: Record<string, number>;
  total: number;
  done: number;
}

export interface ReleasesResponse {
  current: Release;
  next: Release;
  upcoming: Release[];
  past: Release[];
}

export interface ReleaseDay {
  date: string;
  scope: number;
  open: number;
  completed: number;
}

export interface ReleaseProgressResponse {
  release: Release;
  counts: Record<string, number>;
  total: number;
  done: number;
  days: ReleaseDay[];
}

export interface Dependency {
  source_id: string;
  target_id: string;
  kind: string;
  /** Stored on the other issue and shown here as its inverse. */
  derived?: boolean;
}

export interface Comment {
  id: string;
  text: string;
  created_by: string;
  // Set when an agent wrote the comment for this person.
  on_behalf_of?: string;
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

/** Fields of the new-issue modal, saved as an issue draft (`.xpo/drafts/`). */
export interface IssueDraftPayload {
  title: string;
  description?: string;
  labels?: string[];
  additional_labels?: string[];
  status?: string;
  estimate?: number;
  parent_id?: string;
  assignee?: string;
}

export interface IssueDraft {
  id: string;
  created_at: string;
  updated_at: string;
  created_by: string;
  payload: IssueDraftPayload;
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
