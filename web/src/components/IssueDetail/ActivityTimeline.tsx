import { useState, useMemo } from "react";
import type { Issue } from "../../api/client";
import { useIssueHistory } from "../../api/queries";
import { ActorName, Avatar, Text } from "../ui";
import { ChevronDown } from "lucide-react";
import {
  formatRelativeTime,
  activityActor,
  commentAuthor,
} from "../../utils/format";
import { timelineConnectors } from "./activity-timeline-utils";
import { describeEvent } from "./activity-describe";
import { ActivityValue } from "./activity-chips";
import { IssueMarkdown } from "../IssueRef/IssueMarkdown";

type ActivityEntry =
  | {
      kind: "system";
      actor: string;
      author: string;
      via?: string;
      content: React.ReactNode;
      time: string;
    }
  | {
      kind: "comment";
      actor: string;
      author: string;
      via?: string;
      text: string;
      time: string;
    };

// Timeline rail geometry (px). Spacing between entries matches the old
// space-y-3 / mt-4 layout; the avatar sits 8px into a py-1.5 text-sm row.
const SYSTEM_GAP = 12;
const COMMENT_GAP = 16;
const AVATAR_TOP = 8;
const AVATAR_BOTTOM = 24;
const RAIL_INSET = 2;

// Centered on the avatar column: 1px card border + 16px padding + 8px
// (half the sm avatar); system rows use pl-[17px] to line up.
function RailSegment({ style }: { style: React.CSSProperties }) {
  return (
    <div
      aria-hidden
      className="absolute left-[25px] w-px -translate-x-1/2 bg-[var(--color-border-default)]"
      style={style}
    />
  );
}

export default function ActivityTimeline({
  issue,
  newComment,
  onNewCommentChange,
  onAddComment,
  saving,
  commentRef,
}: {
  issue: Issue;
  newComment: string;
  onNewCommentChange: (v: string) => void;
  onAddComment: () => void;
  saving: boolean;
  commentRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  const history = useIssueHistory(issue);
  const [sortNewest, setSortNewest] = useState(true);

  const entries = useMemo(() => {
    const items: ActivityEntry[] = [];

    for (const evt of history) {
      const actor = activityActor(evt.created_by, evt.on_behalf_of);
      if (evt.type === "COMMENT") {
        const p = evt.payload || {};
        const author = commentAuthor(evt.created_by, evt.on_behalf_of);
        items.push({
          kind: "comment",
          actor: author.identity,
          author: author.name,
          via: author.via,
          text: String(p.text || ""),
          time: evt.created_at,
        });
      } else {
        const desc = describeEvent(evt);
        if (desc) {
          items.push({
            kind: "system",
            actor: actor.identity,
            author: actor.name,
            via: actor.via,
            content: desc,
            time: evt.created_at,
          });
        }
      }
    }

    const dir = sortNewest ? -1 : 1;
    items.sort(
      (a, b) => dir * (new Date(a.time).getTime() - new Date(b.time).getTime()),
    );
    return items;
  }, [history, sortNewest]);

  return (
    <div className="mt-8 pt-6 border-t border-[var(--color-border-subtle)]">
      <div className="flex items-center justify-between mb-5">
        <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
          Activity
        </h3>
        <button
          onClick={() => setSortNewest(!sortNewest)}
          className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          title={sortNewest ? "Showing newest first" : "Showing oldest first"}
        >
          {sortNewest ? "Newest" : "Oldest"}
          <ChevronDown
            className={`w-3 h-3 transition-transform ${sortNewest ? "" : "rotate-180"}`}
          />
        </button>
      </div>

      <div>
        {entries.map((entry, i) => {
          const { above, below } = timelineConnectors(i, entries.length);

          if (entry.kind === "system") {
            const gap = i > 0 ? SYSTEM_GAP : 0;
            return (
              <div key={`sys-${i}`} className="relative" style={{ paddingTop: gap }}>
                {above && (
                  <RailSegment
                    style={{ top: 0, height: gap + AVATAR_TOP - RAIL_INSET }}
                  />
                )}
                {below && (
                  <RailSegment
                    style={{ top: gap + AVATAR_BOTTOM + RAIL_INSET, bottom: 0 }}
                  />
                )}
                <div className="flex items-start gap-2 pl-[17px] pr-4 py-1.5 text-sm text-[var(--color-text-muted)]">
                  <span className="flex h-5 items-center">
                    <Avatar name={entry.actor} size="sm" />
                  </span>
                  {/* One run of text, so parts are separated by word spaces and long rows wrap like a sentence. */}
                  <div data-activity="system" className="min-w-0 leading-5">
                    <ActivityValue>
                      <ActorName name={entry.author} via={entry.via} />
                    </ActivityValue>{" "}
                    {entry.content} · {formatRelativeTime(entry.time)}
                  </div>
                </div>
              </div>
            );
          }

          return (
            <div key={`cmt-${i}`} className="relative" style={{ paddingTop: COMMENT_GAP }}>
              {above && <RailSegment style={{ top: 0, height: COMMENT_GAP }} />}
              <div className="rounded-[var(--radius-lg)] bg-[var(--color-surface-1)] py-3 px-4 border border-[var(--color-border-subtle)]">
                <div className="flex items-center gap-3 mb-2">
                  <span className="flex items-center gap-3">
                    <Avatar name={entry.actor} size="sm" />
                    <Text weight="medium" color="primary">
                      <ActorName name={entry.author} via={entry.via} />
                    </Text>
                  </span>
                  <Text>
                    {formatRelativeTime(entry.time)}
                  </Text>
                </div>
                <div className="prose-exponential prose-activity">
                  <IssueMarkdown>{entry.text}</IssueMarkdown>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Comment input */}
      <div className="mt-5 rounded-[var(--radius-lg)] bg-[var(--color-surface-1)] border border-[var(--color-border-subtle)] overflow-hidden transition-colors focus-within:border-[var(--color-border-focus)]">
        <textarea
          ref={commentRef}
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder="Leave a comment..."
          rows={1}
          className="w-full text-[0.875rem] bg-transparent text-[var(--color-text-primary)] px-4 py-3 outline-none placeholder:text-[var(--color-text-muted)] resize-none"
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onAddComment();
          }}
        />
        <div className="flex items-center justify-end gap-2 px-3 py-2">
          <button
            onClick={onAddComment}
            disabled={!newComment.trim() || saving}
            className="w-7 h-7 flex items-center justify-center rounded-full bg-[var(--color-accent-primary)] text-white disabled:opacity-20 hover:bg-[var(--color-accent-primary-hover)] transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4.5 10.5L12 3m0 0l7.5 7.5M12 3v18"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
