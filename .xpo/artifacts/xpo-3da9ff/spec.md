# Spec: Enforce PLANNED-only start in StartWork

## What

`StartWork` should only accept issues in PLANNED status. BACKLOG issues must be rejected.

## How

Add a case in the status switch (start.go lines 22-36) for `model.StatusBacklog` that returns an error: "issue %s is in BACKLOG — move it to PLANNED before starting".

## Tests

1. **TestStartWork_BacklogIssue_Rejected** — BACKLOG issue without force, returns error mentioning BACKLOG and PLANNED
2. Verify existing `TestStartWork_NonGitRepo` (which uses a PLANNED issue) still passes

## AC

- [ ] BACKLOG → DOING is rejected with descriptive error
- [ ] PLANNED → DOING still works
- [ ] Terminal, BLOCKED, already-DOING guards unchanged
- [ ] `make test` passes
