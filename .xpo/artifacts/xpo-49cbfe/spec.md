# handleDraft bypasses all input validation

## Problem

The `handleDraft` handler in `internal/server/handlers.go` deserializes payloads directly into `model.CreatePayload`/`model.UpdatePayload` and passes them to the service layer with zero validation. It also ignores `json.Unmarshal` errors on lines 116, 127, 137, 147.

## Constraint

The frontend sends `model.CreatePayload` format (field names: `parent_id`, `estimate`, `dependencies`) — not `inputs.AddInput` format (`parent`, `story_points`, `links`). So we can't route through `ToCreatePayload`/`ToUpdatePayload` without also changing the frontend. Validation is added inline in the handler instead.

## Fix

### All branches

- Check `json.Unmarshal` errors and respond with 400.

### CREATE branch

- Validate `title` is non-empty.
- Validate `status` against known statuses (if non-empty).
- Resolve `parent_id` via `GetIssue()` (if non-empty).
- Resolve each `dependencies[].target_id` via `GetIssue()`.
- Validate `estimate` via `config.ValidateEstimate()` (if > 0).

### UPDATE branch

- Validate `req.IssueID` is non-empty.
- Validate `status` against known statuses (if non-nil).
- Resolve `parent_id` via `GetIssue()` (if non-nil, non-empty).
- Resolve each `dependencies[].target_id` via `GetIssue()`.

### COMMENT branch

- Validate `req.IssueID` is non-empty.
- Validate `text` is non-empty.

### DELETE branch

- Validate `req.IssueID` is non-empty.

## Acceptance Criteria

- [ ] All `json.Unmarshal` errors return 400.
- [ ] CREATE with empty title returns 400.
- [ ] CREATE with invalid status returns 400.
- [ ] CREATE resolves parent_id and dependency target IDs.
- [ ] UPDATE resolves parent_id and dependency target IDs.
- [ ] COMMENT with empty text returns 400.
- [ ] Operations with empty issue_id return 400.
