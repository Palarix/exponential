# Walkthrough: handleDraft bypasses all input validation

## What changed

One file: `internal/server/handlers.go`, the `handleDraft` method.

## Before

The handler deserialized payloads with `json.Unmarshal` and ignored errors. The raw `model.CreatePayload` / `model.UpdatePayload` went directly to the service layer without validation. Any string was accepted for status, parent_id was stored as-is from user input, and dependency target IDs were not resolved.

## After

Each branch now validates its inputs before calling the service layer:

### CREATE branch

1. `json.Unmarshal` error → 400 "invalid CREATE payload"
2. Empty `title` → 400 "title is required"
3. Non-empty `status` validated via `inputs.ValidateStatus()` → 400 on invalid
4. Non-empty `parent_id` resolved via `client.GetIssue()` → 400 if not found, canonical ID stored
5. Each `dependencies[].target_id` resolved via `client.GetIssue()` → 400 if not found
6. `estimate > 0` validated via `config.ValidateEstimate()` → 400 on invalid

### UPDATE branch

1. Empty `issue_id` → 400
2. `json.Unmarshal` error → 400
3. Non-nil `status` validated via `inputs.ValidateStatus()`
4. Non-nil, non-empty `parent_id` resolved via `client.GetIssue()`
5. Each `dependencies[].target_id` resolved via `client.GetIssue()`

### COMMENT branch

1. Empty `issue_id` → 400
2. `json.Unmarshal` error → 400
3. Empty `text` → 400

### DELETE branch

1. Empty `issue_id` → 400
2. `json.Unmarshal` error → 400

## Design decisions

The frontend sends `model.CreatePayload` format (field names `parent_id`, `estimate`, `dependencies`) rather than `inputs.AddInput` format (`parent`, `story_points`, `links`). Routing through `ToCreatePayload`/`ToUpdatePayload` would require changing the frontend payload format. Instead, equivalent validation was added inline in the handler, using the same `inputs.ValidateStatus()` and `config.ValidateEstimate()` functions as the MCP tools.

The `inputs` package was added to the imports.
