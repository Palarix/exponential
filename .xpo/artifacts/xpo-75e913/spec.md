# Web UI: Render Artifacts on Issue Detail View

**Issue:** xpo-75e913

## Scope

Surface artifact metadata across the web UI: issue detail page, activity timeline, inbox, dashboard activity feed, and board/backlog card indicators.

## Changes

### 1. Backend: Event Filters

**`internal/exponential/timeline.go`** — Add `model.EventTypeArtifact` to `IsMeaningfulActivityEvent` so ARTIFACT events appear in the dashboard activity feed.

**`internal/exponential/inbox.go`** — Add `model.EventTypeArtifact` to `isMeaningfulInboxEvent` so ARTIFACT events appear in the inbox. Add an `EventTypeArtifact` case to `FormatInboxItem` for CLI inbox rendering.

### 2. Backend: API Response

**`internal/server/responses.go`** — Add `Artifacts` field to `IssueResponse` and a new `ArtifactResponse` struct. Populate from `issue.Artifacts` in `issueToResponse`.

### 3. Frontend: Types

**`web/src/api/types.ts`** — Add `ArtifactSummary` interface and `artifacts?: ArtifactSummary[]` to `Issue`. Add `'ARTIFACT'` to `InboxItem.type` union.

### 4. Frontend: Issue Detail — Artifacts Section

**`web/src/components/IssueDetail/ArtifactList.tsx`** — New component rendering artifacts between SubIssuesTable and ActivityTimeline. Shows a collapsible list with icons per type (FileText for spec, BookOpen for walkthrough, Paperclip for generic). Each row shows filename, type badge, relative timestamp, and author. Clicking a row expands to show content fetched via the MCP read operation (or we render inline from the show API if content is available).

Since artifacts are metadata-only in the issue response (no content inlined), the section shows the list with metadata. Content viewing is deferred to future work or linked to CLI.

### 5. Frontend: Activity Timeline

**`web/src/components/IssueDetail/ActivityTimeline.tsx`** — Add `ARTIFACT` case to `describeEvent` that renders "added spec", "updated walkthrough", "deleted notes.md" etc. based on the `artifact_type`, `filename`, and `action` fields in the payload.

### 6. Frontend: Dashboard Activity Feed

**`web/src/components/Dashboard/ActivityFeed.tsx`** — Add `artifact` to `ActIconKey` union, add icon mapping (Paperclip), add `ARTIFACT` case to `describeActivity`.

### 7. Frontend: Inbox

**`web/src/components/Inbox/Inbox.tsx`** — Add `ARTIFACT` case to `buildChangeSummary` so artifact events show as "spec added", "walkthrough updated", etc.

### 8. Frontend: Board Cards

**`web/src/components/Board/BoardCard.tsx`** — Add artifact indicator icons in the bottom row. Three possible icons: FileText (spec), BookOpen (walkthrough), Paperclip (attachments). Only shown when the issue has the corresponding artifact type.

### 9. Frontend: Backlog Rows

**`web/src/components/Backlog/Backlog.tsx`** — Add the same artifact indicator icons after the branch badge, before the flex spacer.
