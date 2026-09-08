# Spec: Fix remote mutation tests to validate request payloads

## What

Make the mock draft handler validate incoming requests so tests catch wrong event types, missing issue IDs, and malformed payloads.

## How

Replace the permissive `POST /api/draft` handler with one that:
1. Validates `IssueID` is non-empty (except for creates)
2. Validates `Type` is a known event type
3. For updates: decodes payload as `UpdatePayload`, verifies `Status` field is present
4. For comments: decodes payload as `CommentPayload`, verifies `Text` is non-empty
5. For deletes: decodes payload as `DeletePayload`, verifies `Reason` is non-empty
6. Returns 400 on validation failure

## Test changes

- `TestRemoteTransport_UpdateIssue`: verify returned message references the issue ID
- `TestRemoteTransport_AddComment`: verify no error (mock now validates text is present)
- `TestRemoteTransport_DeleteIssue`: verify no error (mock now validates reason is present)
- Add `TestRemoteTransport_UpdateIssue_WrongID`: pass a nonexistent issue ID, mock returns 404
- Add `TestRemoteTransport_AddComment_EmptyText`: verify mock rejects empty comment

## AC

- [ ] Mock validates request type, issue ID, and payload shape
- [ ] Existing tests still pass (they send correct data)
- [ ] New tests verify validation catches bad inputs
- [ ] `make test` passes
