# Web UI Artifact Rendering — Walkthrough

## How artifacts flow from backend to UI

1. **Projection** — `ProjectIssues` replays ARTIFACT events, building `Issue.Artifacts []ArtifactSummary` with type, filename, timestamp, and author.

2. **API response** — `issueToResponse` in `responses.go` maps `issue.Artifacts` into `ArtifactResponse` structs, included in the JSON payload alongside dependencies and comments.

3. **Frontend types** — `ArtifactSummary` interface in `types.ts` mirrors the backend shape. The `Issue` interface carries an optional `artifacts` array.

## Where artifacts appear

### Issue detail page

`ArtifactList.tsx` renders between SubIssuesTable and ActivityTimeline. Each artifact gets a type-specific icon:

- **FileText** — spec
- **BookOpen** — walkthrough
- **Paperclip** — generic attachments

Rows show the label (or filename for generics), the author short name, and a relative timestamp.

### Activity timeline

ARTIFACT events in `describeEvent` render as system entries: "{action} {filename}" — e.g. "created spec.md", "updated walkthrough.md", "deleted notes.txt".

### Dashboard activity feed

`describeActivity` handles the ARTIFACT case with a Paperclip icon and renders "{who} {action} {filename} on {issue title}".

### Inbox

`buildChangeSummary` includes artifact events as "{filename} {action}" chips in the notification card summary line.

### Board cards and backlog rows

Three indicator icons appear in the bottom row / after the branch badge when the issue has artifacts of the corresponding type. They use `<span title="...">` wrappers for hover tooltips since lucide icons don't accept a `title` prop directly.

## Event filter changes

Both `IsMeaningfulActivityEvent` (timeline.go) and `isMeaningfulInboxEvent` (inbox.go) now include `EventTypeArtifact` in their allow-lists, so artifact events are no longer silently dropped.
