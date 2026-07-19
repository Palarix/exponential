# Spec: Minor validation gaps across MCP and HTTP handlers

## 1. Negative estimates in MCP `add`

**Current state:** `ValidateCreatePayload` already rejects negative estimates. The issue description is stale — this was fixed in a prior commit.

**Action:** No change needed. Already validated.

## 2. Assignee format validation

**Current state:** Only max-length checked (200 chars). No format validation.

**Fix:** Add a lightweight format check in `ValidateCreatePayload` and `ValidateUpdatePayload`. Accept either:
- Empty string (no assignee)
- A string matching `Name <email>` pattern (contains `<` and `>` wrapping an `@`)
- A plain email address

Use a simple heuristic rather than a full RFC 5322 parser: if the string contains `<` and `>`, ensure the angle-bracket content has an `@`. If no angle brackets, accept as-is (plain name or email).

**Decision:** Actually, being overly strict here could break existing data. A more pragmatic approach: validate that if angle brackets are present, they're well-formed. Reject strings that are clearly garbage (empty after trim, only whitespace). Keep it light.

## 3. Label color format (hex validation)

**Current state:** HTTP handlers `handleAddLabel` and `handleUpdateLabel` accept any non-empty string as color.

**Fix:** Validate color is a 6-digit hex string (with or without `#` prefix). Normalize to include `#`. Regex: `^#?[0-9a-fA-F]{6}$`.

Add a `ValidateHexColor` helper and call it in both handlers.

## 4. Priority bounds

**Current state:** No bounds checking anywhere. Model uses int with convention 0=None, 1=Urgent, 2=High, 3=Medium, 4=Low.

**Fix:** Add priority validation in `ValidateCreatePayload` and `ValidateUpdatePayload`: must be in range [0, 4]. Add to both.

## 5. MCP `list` status filter

**Current state:** Invalid status strings silently match nothing.

**Fix:** Validate each status in the filter against the known set before querying. Return an error naming the invalid value(s).

## 6. MCP `merge` strategy

**Current state:** Invalid strings silently default to squash.

**Fix:** Add a `default` case that returns an error when the strategy is non-empty and not one of `"squash"`, `"merge"`, `"ff"`. Empty string still defaults to squash (the normal case when unspecified).

## 7. HTTP `handleMergeIssue` JSON decode + strategy

**Current state:** JSON decode errors are swallowed; invalid strategies default to squash.

**Fix:**
- Return 400 on JSON decode error (same pattern as other handlers).
- Add strategy validation identical to fix #6.

## Acceptance Criteria

- All seven validation gaps are addressed.
- Existing tests pass.
- New validation errors return clear messages naming the invalid value.
