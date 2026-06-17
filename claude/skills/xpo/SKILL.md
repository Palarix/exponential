---
name: xpo
description: "Exponential issue tracker workflow rules and command reference. Auto-loads when the agent is doing development work in a project that uses xpo for issue tracking."
---

# Exponential Issue Tracker — Agent Workflow & Reference

Exponential is a JSONL-based issue tracker committed directly to Git alongside your project. It serves as persistent project memory for AI agents and human developers alike.

## Workflow Rules

1. **No ghost work** — every code change MUST be backed by a xpo issue.
2. **Only pick up PLANNED work** — do not start work on BACKLOG issues.
3. **Create before you code** — if no issue exists, create it first (after user approval).
4. **Start before you edit** — run `xpo start <id>` before touching any code.
5. **Comment before you close** — add a markdown summary of changes before `xpo done`.
6. **File what you find** — if you discover bugs or needed work, file them as new issues rather than fixing them silently or leaving TODOs.

## Issue Statuses

| Status    | Meaning                                 |
|-----------|-----------------------------------------|
| `BACKLOG` | Captured but not yet prioritized        |
| `PLANNED` | Prioritized and ready to be worked on   |
| `DOING`   | Actively being worked on                |
| `BLOCKED` | Waiting on something external           |
| `DONE`    | Completed                               |

## Command Reference

### Discovery

```bash
xpo list                               # list all issues (alias: xpo ls)
xpo list --status PLANNED              # filter by status
xpo list --label bug                   # filter by label
xpo list --assignee "name"             # filter by assignee
xpo list --mine                        # show only my issues
xpo list --parent <epic-id>            # show children of an epic
xpo list --since 1w                    # updated in the last week
xpo list --match "search term"         # full-text search
xpo list --all                         # include old DONE issues
xpo show <issue-id>                    # full issue details
xpo comments <issue-id>               # list comments on an issue
xpo history <issue-id>                 # audit trail
```

### Creating Issues

```bash
xpo add "Title" --label "task" --desc "Description in markdown"
xpo add "Title" --label "task" -p <parent-id> --desc "Description"
xpo add "Title" --label "feature" --sp 3 --assignee "name"
xpo add "Title" --label "epic" --desc "High-level goal"
xpo add "Title" --label "bug" --desc "Steps to reproduce..."
xpo add "Title" --label "task" --force   # skip duplicate check
```

### Status Transitions

```bash
xpo planned <issue-id>    # BACKLOG → PLANNED
xpo start <issue-id>      # → DOING
xpo blocked <issue-id>    # → BLOCKED
xpo done <issue-id>       # → DONE
```

### Updating Issues

```bash
xpo update <issue-id> --desc "New description"
xpo update <issue-id> --label "bug" --label "critical"
xpo update <issue-id> --assignee "name"
xpo update <issue-id> --sp 5
xpo update <issue-id> --parent <epic-id>
xpo update <issue-id> --status PLANNED
```

### Comments

```bash
xpo comment <issue-id> "Comment text in **markdown**"
echo "Comment text" | xpo comment <issue-id>
```

### Relationships

```bash
# Types: blocks, blocked_by, depends_on, dependency_of,
#        duplicates, duplicated_by, relates_to
xpo link <source-id> <target-id> -t "blocks"
```

### Estimation

```bash
xpo estimate <issue-id> <story-points>
```

### Maintenance

```bash
xpo archive    # archive completed and deleted issues
xpo doctor     # diagnose and fix configuration issues
xpo config     # show current configuration
xpo init       # initialize .xpo directory in a new project
xpo migrate    # migrate data to new schema version
```

## Writing Good Descriptions and Comments

Descriptions and comments render as **Markdown** in the xpo web UI. Always use:

- Headings for sections
- Bullet/numbered lists for changes
- Backtick code spans for file/function names
- Code blocks for examples
- Double newlines between paragraphs
