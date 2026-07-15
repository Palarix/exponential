# MCP update tool missing estimate validation

## Problem

`ValidateUpdatePayload` on `Client` does not validate the estimate field. `ValidateCreatePayload` calls `config.ValidateEstimate()` for estimates > 0, but the update path accepts any integer — including negative values or values outside the configured estimation system.

## Fix

Add estimate validation to `ValidateUpdatePayload` in `internal/exponential/client.go`. The `UpdatePayload` uses `*int` for estimate, so check when non-nil and > 0.

## Acceptance Criteria

- [ ] Updating an issue with an invalid estimate returns an error.
- [ ] Negative estimates are rejected.
- [ ] Zero estimate (clearing the field) is accepted.
- [ ] Tests pass.
