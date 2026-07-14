# Ability to Add Attachments from Web UI

## Problem

The Web UI can display and download text artifacts but has no way to upload or delete files. Binary files (images, PDFs) are not supported at all. The existing artifact system is text-only and git-tracked, which is correct for specs and walkthroughs but wrong for binary attachments that would bloat the repository.

## Design: Attachments as a Separate Concept

Attachments are distinct from artifacts:

| | Artifacts | Attachments |
|---|---|---|
| Content | Text (markdown) | Any file (binary or text) |
| Storage | `.xpo/artifacts/{issue-id}/` | `.xpo/attachments/{issue-id}/` |
| Git | Tracked (committed with code) | Tracked (committed directly, or via LFS — see xpo-6c8ca4) |
| Event type | `ARTIFACT` | `ATTACHMENT` |
| Examples | spec.md, walkthrough.md | screenshots, logs, PDFs |

Attachments are always committed to git. Whether they go through Git LFS is a separate concern handled at `xpo init` time (xpo-6c8ca4). This story focuses on the attachment data model, backend API, and Web UI.

## Requirements

1. **New `ATTACHMENT` event type** with metadata: `filename`, `mime_type`, `byte_size`, `sha256`, `action` (added/deleted).
2. **Storage** at `.xpo/attachments/{issue-id}/{filename}`.
3. **Upload API** — `POST /api/issues/{id}/attachments` accepts multipart form data.
4. **Delete API** — `DELETE /api/issues/{id}/attachments/{filename}`.
5. **Serve API** — `GET /api/issues/{id}/attachments/{filename}` returns raw bytes with correct `Content-Type`.
6. **Upload UI in IssueDetail** — "Attach file" button with file picker.
7. **Upload UI in NewIssueModal** — file picker queues files, uploads after issue creation.
8. **Delete UI** — trash icon on attachment rows.
9. **No inline preview** — download links only for now.

## Data Model

### Event payload (`internal/model/types.go`)

```go
type AttachmentPayload struct {
    Filename string `json:"filename"`
    MimeType string `json:"mime_type"`
    ByteSize int64  `json:"byte_size"`
    SHA256   string `json:"sha256"`
    Action   string `json:"action"` // "added" or "deleted"
}
```

### Projected summary (`internal/model/types.go`)

```go
type AttachmentSummary struct {
    Filename string    `json:"filename"`
    MimeType string    `json:"mime_type"`
    ByteSize int64     `json:"byte_size"`
    SHA256   string    `json:"sha256"`
    AddedAt  time.Time `json:"added_at"`
    AddedBy  string    `json:"added_by"`
}
```

### On the Issue struct

```go
Attachments []AttachmentSummary
```

## Implementation Outline

### Backend

#### Model (`internal/model/types.go`)

- Add `EventTypeAttachment EventType = "ATTACHMENT"`.
- Add `AttachmentPayload` and `AttachmentSummary` structs.
- Add `Attachments []AttachmentSummary` to the `Issue` struct.

#### Projection (`internal/exponential/projection.go`)

Handle `EventTypeAttachment`:
- `"added"` → append or update `AttachmentSummary` in `issue.Attachments`.
- `"deleted"` → filter it out.

#### Attachment storage (`internal/exponential/attachment.go` — new file)

Methods on `LocalTransport`:

- `AddAttachment(issueID, filename, mimeType string, data []byte) error`
  - Validate filename (reuse `validateArtifactFilename`).
  - Compute SHA-256 of `data`.
  - Write to `.xpo/attachments/{issueID}/{filename}`.
  - Append `ATTACHMENT` event with `action: "added"`.
- `ReadAttachmentBytes(issueID, filename string) ([]byte, error)` — read raw bytes.
- `DeleteAttachment(issueID, filename string) error` — remove file, append event with `action: "deleted"`.
- `ListAttachments(issueID string) ([]AttachmentSummary, error)` — return from projection.

#### Transport interface (`internal/exponential/transport.go`)

Add the four attachment methods.

#### Server responses (`internal/server/responses.go`)

```go
type AttachmentResponse struct {
    Filename string    `json:"filename"`
    MimeType string    `json:"mime_type"`
    ByteSize int64     `json:"byte_size"`
    SHA256   string    `json:"sha256"`
    AddedAt  time.Time `json:"added_at"`
    AddedBy  string    `json:"added_by"`
}
```

Add `Attachments []AttachmentResponse` to `IssueResponse`. Map in `issueToResponse`.

#### Server routes (`internal/server/router.go`)

```
GET    /api/issues/{id}/attachments/{filename}    (issue.read)
POST   /api/issues/{id}/attachments               (issue.write)
DELETE /api/issues/{id}/attachments/{filename}     (issue.write)
```

#### Server handlers (`internal/server/handlers.go`)

- **`handleGetAttachment`** — read bytes, determine `Content-Type` from stored `mime_type` (fallback `application/octet-stream`), set `Content-Disposition`, write raw bytes.
- **`handleUploadAttachment`** — parse multipart form (`file` field), read bytes, detect MIME type from file header's `Content-Type` (fallback `mime.TypeByExtension`), call `AddAttachment`, broadcast SSE `"ATTACHMENT"`, return metadata JSON.
- **`handleDeleteAttachment`** — call `DeleteAttachment`, broadcast SSE, return success.

#### SSE (`internal/server/sse.go`)

Add `"ATTACHMENT"` to the broadcast event types so the frontend receives real-time updates.

### Frontend

#### Types (`web/src/api/types.ts`)

```typescript
export interface AttachmentSummary {
  filename: string;
  mime_type: string;
  byte_size: number;
  sha256: string;
  added_at: string;
  added_by: string;
}
```

Add `attachments?: AttachmentSummary[]` to the `Issue` interface.

#### API client (`web/src/api/client.ts`)

- `uploadAttachment(issueId: string, file: File): Promise<AttachmentSummary>` — POST multipart.
- `deleteAttachment(issueId: string, filename: string): Promise<void>` — DELETE.
- `getAttachmentUrl(issueId: string, filename: string): string` — returns the direct URL.

#### AttachmentList (`web/src/components/IssueDetail/AttachmentList.tsx` — new)

- Renders attachment rows: file icon, filename, human-readable size, uploader, timestamp, download link, delete button (on hover).
- "Attach file" button triggers a hidden `<input type="file">`. Uploads via `uploadAttachment`, then triggers issue refresh.
- Shown in IssueDetail below the existing ArtifactList section.

#### NewIssueModal (`web/src/components/NewIssueModal/NewIssueModal.tsx`)

- File picker in the "More options" section.
- Queued files stored as `File[]` in state, shown as a removable list.
- After `createIssue` succeeds, upload each file via `uploadAttachment(newIssueId, file)`.
- Clear queue on successful creation.

## Acceptance Criteria

- [ ] New `ATTACHMENT` event type with `filename`, `mime_type`, `byte_size`, `sha256`, `action`.
- [ ] Attachments stored at `.xpo/attachments/{issue-id}/{filename}`.
- [ ] `POST /api/issues/{id}/attachments` accepts multipart upload and stores file.
- [ ] `DELETE /api/issues/{id}/attachments/{filename}` removes file and records event.
- [ ] `GET /api/issues/{id}/attachments/{filename}` serves raw bytes with correct Content-Type.
- [ ] IssueDetail shows attachment list with upload and delete controls.
- [ ] NewIssueModal allows queuing files that upload after issue creation.
- [ ] Binary files (images, PDFs) round-trip without corruption.
- [ ] Old issues without attachments are unaffected.
- [ ] Tests pass (`make test`).
