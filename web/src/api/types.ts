export interface Issue {
  id: string;
  title: string;
  description: string;
  status: string;
  parent_id?: string;
  estimate: number;
  assignee?: string;
  labels?: string[];
  dependencies?: Dependency[];
  comments?: Comment[];
  created_at: string;
  created_by: string;
  updated_at: string;
  is_pending?: boolean;
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
  created_at: string;
}
