import type { Issue } from "../../api/types";
import { unresolvedBlockers } from "../IssueDetail/relation-utils";
import { childProgress } from "../Releases/releases-utils";

/** What the issue hover card's footer counts. */
export interface PreviewCounts {
  comments: number;
  /** Artifacts other than the spec and walkthrough. */
  attachments: number;
  spec: boolean;
  walkthrough: boolean;
  /** Sub-issues done of all; null without sub-issues. */
  children: { done: number; total: number } | null;
  commits: number;
  /** Blockers that are not DONE, as the start gate counts them. */
  blockers: number;
}

export function previewCounts(issue: Issue, issues: Issue[]): PreviewCounts {
  const types = (issue.artifacts ?? []).map(a => a.artifact_type);
  return {
    comments: issue.comments?.length ?? 0,
    attachments: types.filter(t => t !== "spec" && t !== "walkthrough").length,
    spec: types.includes("spec"),
    walkthrough: types.includes("walkthrough"),
    children: childProgress(issue, issues),
    commits: issue.branch_stats?.commits ?? 0,
    blockers: unresolvedBlockers(issue, issues).length,
  };
}
