# Walkthrough: Fix remote mutation tests to validate payloads

## What changed

### Mock handler rewrite

The `POST /api/draft` handler in `newTestServer` now validates every request:

- **Type routing** — switches on `req.Type`; unknown types return 400
- **Issue ID** — update, comment, and delete require a non-empty `IssueID` that exists in the mock's issue set; missing/unknown returns 404
- **Payload validation:**
  - Updates: decoded as `UpdatePayload`, at least one field must be set
  - Comments: decoded as `CommentPayload`, `Text` must be non-empty
  - Deletes: decoded as `DeletePayload`, `Reason` must be non-empty
- Creates work as before (generate `test-new123`)

### Test changes

**Existing tests strengthened:**
- `TestRemoteTransport_UpdateIssue` — now verifies the returned message references the issue ID

**New validation tests:**
- `TestRemoteTransport_UpdateIssue_NotFound` — nonexistent issue ID returns error
- `TestRemoteTransport_AddComment_EmptyText` — empty comment text rejected
- `TestRemoteTransport_AddComment_NotFound` — nonexistent issue ID returns error
- `TestRemoteTransport_DeleteIssue_EmptyReason` — empty reason rejected
- `TestRemoteTransport_DeleteIssue_NotFound` — nonexistent issue ID returns error

These tests would now fail if the client sent the wrong event type, wrong issue ID, or an empty payload — the core deficiency the audit identified.
