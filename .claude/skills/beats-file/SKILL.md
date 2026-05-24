---
name: beats-file
description: "File a new beats issue (bug, task, feature, or epic). Handles duplicate checking, proper labeling, and linking to related issues."
disable-model-invocation: true
---

# File a New Beats Issue

You are creating a new issue in the beats tracker. Follow this process to ensure quality and avoid duplicates.

## Steps

### 1. Check for Duplicates

Before creating anything, search for existing issues that might already cover this:

```bash
beats list --match "relevant search term"
```

If a match looks related, inspect it:

```bash
beats show <issue-id>
```

If an existing issue already covers the work, tell the user instead of creating a duplicate.

### 2. Determine the Type

Choose the appropriate label:

| Label       | When to use                                         |
|-------------|-----------------------------------------------------|
| `bug`       | Something is broken or behaving incorrectly         |
| `feature`   | A new capability that doesn't exist yet             |
| `task`      | A concrete piece of work (refactor, migration, etc) |
| `epic`      | A high-level goal that will be broken into subtasks  |

Use other labels when a more specific classification fits better.

### 3. Create the Issue

```bash
beats add "Title under 100 chars" \
  --label "<type>" \
  --desc "Markdown description with context, steps to reproduce, or acceptance criteria"
```

**If it belongs to an epic:**

```bash
beats add "Title" --label "<type>" -p <epic-id> --desc "Description"
```

**If it relates to another issue:**

```bash
beats link <new-id> <related-id> -t "relates_to"
```

### 4. Confirm

Show the user the created issue ID and a brief summary of what was filed.

## Writing Good Issues

- **Title**: concise, under 100 characters, describes the what
- **Description**: markdown-formatted, includes:
  - Context: why this matters
  - For bugs: steps to reproduce, expected vs actual behavior
  - For features/tasks: acceptance criteria or definition of done
  - For epics: high-level goal and scope

## Rules

- Always check for duplicates first.
- Always include a label.
- Always write descriptions in markdown.
- New issues start in `BACKLOG` status by default — do not change this.
