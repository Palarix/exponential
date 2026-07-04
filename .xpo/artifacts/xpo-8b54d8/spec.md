# Improve timeline and activity view precision

## Goal

Make event rendering in the timeline and issue activity views accurate, consistent, and principal-first.

## Changes

### 1. Remove "and closed this issue" from merge activity

**Where:** Web frontend — merge event rendering in issue activity view.

**Current:** "claude merged xpo-d31711-make-global-command-dialog-wi… via squash and closed this issue"

**Target:** "Nicolas Bettenburg merged xpo-d31711-make-global-command-dialog-wi… via squash"

The DONE transition is now a separate event (with the green checkmark) that follows the MERGE event. The merge entry should only describe the merge.

### 2. Use branch name in timeline commit entries

**Where:** Timeline view — commit event rendering.

**Current:** "Nicolas Bettenburg committed 78979f9 on Make global command dialog wider" (issue title)

**Target:** "Nicolas Bettenburg committed 78979f9 on xpo-d31711-make-global-command-dialog-wider" (branch name)

The commit happens on the branch, not the issue. Use the branch name from the merge payload or branch stats.

### 3. Principal-first actor rendering

**Where:** All event rendering (timeline + issue activity, CLI + web).

**Rule:**
- If `on_behalf_of` is set: display the principal as the primary actor, with "via {agent name}" as secondary/dimmed text.
- If `on_behalf_of` is not set: display `created_by` as the actor (unchanged behavior).

**Parsing:** Extract the display name from the `Name <email>` format for both fields. The agent name should be the short name portion of `created_by` (e.g. "claude-code/2.1.200" → "claude-code", "claude" → "claude").

**Web UI:** Consider a small bot badge or icon next to "via {agent}" to visually distinguish agent-mediated actions.

### 4. Set `on_behalf_of` in drive and merge

**Where:** `internal/exponential/drive.go`, `internal/exponential/merge.go`, and any other code paths where the CLI acts as an agent on behalf of a user.

**Current:** Only MCP-originated events set `on_behalf_of`. Events created by `drive` (walkthrough artifacts, merge events, DONE transitions) use `created_by: "claude <agent@host>"` with no `on_behalf_of`.

**Target:** When `drive` or `merge` is invoked, the git user identity (from `git config user.name` / `git config user.email`) should be set as `on_behalf_of` on all events created during the operation. The `created_by` remains the agent identity.

## Acceptance criteria

- [ ] Merge activity entry does not say "and closed this issue"
- [ ] Timeline commit entries show branch name, not issue title
- [ ] Events with `on_behalf_of` render the principal as primary actor with "via {agent}" attribution
- [ ] Events without `on_behalf_of` render unchanged
- [ ] `xpo drive` sets `on_behalf_of` on all events it creates (walkthrough, merge, DONE)
- [ ] `xpo merge` sets `on_behalf_of` when invoked by an agent context
- [ ] Existing tests pass; new tests cover principal-first rendering logic
