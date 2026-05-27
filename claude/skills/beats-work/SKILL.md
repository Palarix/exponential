---
name: beats-work
description: "Pick up a planned beats issue and work through it following the full beats discipline: discover, start, implement, test, document, and complete."
disable-model-invocation: true
---

# Work on a Beats Issue

You are following the beats-driven development workflow. This is a disciplined process — every step matters.

## Workflow

### 1. Discover

If no issue ID was provided, find the next task to work on:

```bash
beats list --status PLANNED
```

Pick the highest-priority planned issue, or ask the user which one to work on. Then read its full details:

```bash
beats show <issue-id>
```

Read the description, comments, and any linked issues to fully understand the task before writing code.

**Important**: Only work on issues with status `PLANNED`. Do not pick up `BACKLOG` issues.

### 2. Start

Mark the issue as in-progress BEFORE making any code changes:

```bash
beats start <issue-id>
```

### 3. Implement

Make the code changes required by the issue. While working:

- If you discover bugs or necessary work outside the scope of this issue, file them immediately:

```bash
beats add "Discovered issue title" --label "bug" --desc "What you found"
beats link <new-id> <current-id> -t "relates_to"
```

- Do NOT fix unrelated issues silently — file them so they can be prioritized.

### 4. Test

Verify your changes work:

```bash
go test ./...
go build ./...
```

Or use whatever test/build commands are appropriate for the project.

### 5. Document

Add a summary comment to the issue explaining what you changed and why. Use markdown formatting:

```bash
beats comment <issue-id> "## Summary

- Changed X in \`path/to/file.go\` to fix Y
- Added tests for Z

**Rationale:** Explanation of the approach chosen."
```

### 6. Complete

Mark the issue as done only AFTER the user has reviewed and approved your changes:

```bash
beats done <issue-id>
```

## Rules

- Never skip the `beats start` step — the tracker must reflect what you're doing.
- Never mark `beats done` without adding a summary comment first.
- Never mark `beats done` without user approval of the changes.
- File new issues for anything you find that's outside scope — don't fix silently, don't leave TODOs.
