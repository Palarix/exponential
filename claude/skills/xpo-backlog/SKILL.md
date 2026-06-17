---
name: xpo-backlog
description: "Review the project backlog using xpo. Lists issues by status and suggests what to work on next."
disable-model-invocation: true
---

# Review the Exponential Backlog

You are reviewing the project backlog to understand current state and identify next steps.

## Steps

1. **List in-progress work first** to see what's already underway:

```bash
xpo list --status DOING
```

2. **Check for blocked items** that may need attention:

```bash
xpo list --status BLOCKED
```

3. **Review planned work** to see what's ready to pick up:

```bash
xpo list --status PLANNED
```

4. **Scan the full backlog** for context:

```bash
xpo list --status BACKLOG
```

5. **Inspect any issue** that needs more detail:

```bash
xpo show <issue-id>
```

## What to Report

After reviewing, summarize for the user:

- **In progress**: what's actively being worked on and by whom
- **Blocked**: anything stuck and what it's waiting on
- **Ready**: planned issues that are ready to pick up, in priority order
- **Backlog highlights**: anything notable in the backlog that might be worth planning

Keep the summary concise. Link issue IDs so the user can drill in.

## Rules

- Do NOT change any issue status during a backlog review — this is read-only.
- Do NOT start working on issues — only report what you find.
- If you notice stale DOING issues (no recent activity), flag them for the user.
