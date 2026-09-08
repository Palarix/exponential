# Walkthrough: Enforce PLANNED-only start

## What changed

Added a `model.StatusBacklog` case to the status switch in `StartWork` (start.go line 24) that returns an error directing the user to plan the issue first. The guard sits between the terminal-status check and the BLOCKED check, matching the status graph: BACKLOG → PLANNED → DOING.

## Test

`TestStartWork_BacklogIssue_Rejected` — creates a BACKLOG issue, calls `StartWork`, verifies the error mentions both "BACKLOG" and "PLANNED".
