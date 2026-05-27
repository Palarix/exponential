---
name: beats
description: "Beats issue tracker workflow rules and command reference. Auto-loads when the agent is doing development work in a project that uses beats for issue tracking."
---

# Beats Issue Tracker — Agent Workflow & Reference

Beats is a JSONL-based issue tracker committed directly to Git alongside your project. It serves as persistent project memory for AI agents and human developers alike.

## Workflow Rules

1. **No ghost work** — every code change MUST be backed by a beats issue.
2. **Only pick up PLANNED work** — do not start work on BACKLOG issues.
3. **Create before you code** — if no issue exists, create it first (after user approval).
4. **Start before you edit** — run `beats start <id>` before touching any code.
5. **Comment before you close** — add a markdown summary of changes before `beats done`.
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
beats list                               # list all issues (alias: beats ls)
beats list --status PLANNED              # filter by status
beats list --label bug                   # filter by label
beats list --assignee "name"             # filter by assignee
beats list --mine                        # show only my issues
beats list --parent <epic-id>            # show children of an epic
beats list --since 1w                    # updated in the last week
beats list --match "search term"         # full-text search
beats list --all                         # include old DONE issues
beats show <issue-id>                    # full issue details
beats comments <issue-id>               # list comments on an issue
beats history <issue-id>                 # audit trail
```

### Creating Issues

```bash
beats add "Title" --label "task" --desc "Description in markdown"
beats add "Title" --label "task" -p <parent-id> --desc "Description"
beats add "Title" --label "feature" --sp 3 --assignee "name"
beats add "Title" --label "epic" --desc "High-level goal"
beats add "Title" --label "bug" --desc "Steps to reproduce..."
beats add "Title" --label "task" --force   # skip duplicate check
```

### Status Transitions

```bash
beats planned <issue-id>    # BACKLOG → PLANNED
beats start <issue-id>      # → DOING
beats blocked <issue-id>    # → BLOCKED
beats done <issue-id>       # → DONE
```

### Updating Issues

```bash
beats update <issue-id> --desc "New description"
beats update <issue-id> --label "bug" --label "critical"
beats update <issue-id> --assignee "name"
beats update <issue-id> --sp 5
beats update <issue-id> --parent <epic-id>
beats update <issue-id> --status PLANNED
```

### Comments

```bash
beats comment <issue-id> "Comment text in **markdown**"
echo "Comment text" | beats comment <issue-id>
```

### Relationships

```bash
# Types: blocks, blocked_by, depends_on, dependency_of,
#        duplicates, duplicated_by, relates_to
beats link <source-id> <target-id> -t "blocks"
```

### Estimation

```bash
beats estimate <issue-id> <story-points>
```

### Maintenance

```bash
beats archive    # archive completed and deleted issues
beats doctor     # diagnose and fix configuration issues
beats config     # show current configuration
beats init       # initialize .beats directory in a new project
beats migrate    # migrate data to new schema version
```

## Writing Good Descriptions and Comments

Descriptions and comments render as **Markdown** in the beats web UI. Always use:

- Headings for sections
- Bullet/numbered lists for changes
- Backtick code spans for file/function names
- Code blocks for examples
- Double newlines between paragraphs
