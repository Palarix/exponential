# Agent Instructions

This repository uses the `beats` issue tracker as the persistent project memory and to manage development tasks. As an AI agent, you should use `beats` to understand the current state of the project, plan your work, record new findings, and to document the rationale and work done to implement your tasks.

## Development Workflow

```bash
# 1. Check for existing tasks and discover issue-ids
beats ls

# 2a. Check task details if existing entry
beats show <issue-id>

# 2b. Create new entry if not exists
beats add --label '<type>' <title>

# 3. Set issue status to in progress
beats start <issue-id>

# 4. Make changes
# ...

# 5. Test changes
go test ./...
go build ./...

# 6. Add a comment with a summary of the changes or walkthrough
beats comment <issue-id> "<summary>"

# 7. Set issue status to done
beats done <issue-id>
```

## Using `beats` to Organize Your Work

`beats` serves as the persistent memory and issue tracker for this project.

- **Start** by reading the backlog to understand what needs to be done.
- **Update** the status of tasks you are working on.
- **Record** any new tasks or bugs you discover as new issues. Do not just fix them implicitly or leave them as TODO comments in code; create a tracked issue so it can be prioritized.
- **Persist** your planning. If a task is too big, break it down into child tasks in `beats`.
- **Organize** your work: use issues labeled as `epic` with sub-issues (`issues` with a `parent-id` set) to organize large chunks of work.

## Strict Workflow Rules

1. **No "Ghost" Work**: Any work done by an agent MUST be backed by a beats task/bug/epic.
2. **Only pick up planned work:** Do not pick up and start work on issues with the `BACKLOG` status. Issues must be `PLANNED` to be eligible for being worked on.
3. **Missing Tasks**: If no issue exists for your current objective, you must create it first. Do this only _after_ the user approves your initial design/plan.
4. **In-Progress**: Before starting any code work (editing files), you MUST set the corresponding beats task to `DOING` using `beats start`.
5. **Completion**: You MUST set the beats task to `DONE` using `beats done` _only after_ the user approves the final review/walkthrough. You MUST add a comment to the issue first that summarized your changes.

## Agent Identity

When performing actions that modify the tracker (add, update), ensure you are identified as an agent if possible. For instance, if you are an `OpenCode Agent`, identify yourself as `OpenCode <agent@opencode.local>` - using the host machine name as part of the email address is highly encouraged so we can identify code contributed from different execution environments!

## Usage Guide

### 1. Discovery (Reading the State)

- Before creating new issues, first ensure that there are no existing issues that already cover the same work (`beats ls`)
- If an issue looks related, inspect issue details first (`beats show <id>`) to determine if its related
- Only if no related issues exist, you may create a new issue in the project

#### Examples:

**List all issues:**

```bash
./beats list
```

Use this to find your assigned task or pick the next prioritized item from the backlog.

**Read a specific issue:**

```bash
./beats show <issue-id>
```

Always read the full details of an issue before starting work. It may contain description, acceptance criteria, or context from previous agents.

### 2. Planning (Creating Issues)

- Always add a label to new issues using the `--label <name>` option
- When possible pick from the built-in labels ('bug', 'feature', 'epic') unless another label is more appropriate
- Keep the title under 100 characters
- Descriptions and comments are rendered as **Markdown** in the web UI. Always write them in proper markdown: use headings, lists, code blocks, bold/italic, and links where appropriate. Use double newlines between paragraphs. Markdown checklists (`- [ ] Title` format) can sub-divide task steps.
- When a new issue belongs conceptually to an Epic, add the corresponding epic as a parent to the new issue (`--parent <id>` option)

#### Examples:

**Create an Epic (High-level goal):**

```bash
./beats add --label "epic" "Refactor Database Layer" --desc 'Move from SQLite to Postgres'
```

**Create a Task (Actionable item):**

```bash
./beats add --label "task" "Create Migration Script" -p <epic-id> --desc 'Write SQL migration'
```

**Filing Bugs/Findings:**
If you encounter a bug or necessary refactor while working on something else, file it immediately so it isn't lost.

```bash
./beats add --label "bug" "Race condition in login" --desc 'Observed when...'
```

### 3. Execution (Updating Status)

- Mark the issue as "in progress" by calling `beats start <id>` before starting your work and making code changes
- Record any new tasks, issues or bugs discovered during your work as new issues using `beats add` (see above)
- When new tasks, issue or bugs are created this way, link them to the currently worked on issue using the `dependencies` mechanism

#### Examples:

**Start a task:**

```bash
# Mark the given issue-id as being in progress by starting work on that issue
./beats start <issue-id>
```

**Update details:**

```bash
# Add a new comment to the given issue-id
beats comment <issue-id> 'Updated description with new findings...'
```

**Add a dependency link between two issues**:

```bash
# Add a dependency of given type from some-id to other-id
./beats link <some-id> <other-id> -t "<type>"
```

### 4. Completion (Finishing Work)

- When work on your issue, task, or bug is complete, first add a summary of the changes together with your rationale for the changes as a comment using the `beats comment <id>` command.
- Comments are rendered as **Markdown** in the web UI. Use proper markdown formatting: headings for sections, bullet or numbered lists for changes, backtick code spans for file/function names. Use double newlines between paragraphs to ensure correct rendering.
- Then mark the issue as completed using the `beats done <id>` command.

#### Examples:

**Add walkthrough:**

```bash
# Add a new comment with markdown formatting
beats comment <issue-id> "## Summary

- Fixed the login race condition in \`auth/session.go\`
- Added mutex lock around session token refresh
- Verified with concurrent request test

**Root cause:** The session refresh was not atomic."
```

**Mark as Done:**

```bash
./beats done <issue-id>
```

## Building this project

- Use the `make cli` command to compile the `beats` binary.
- Use the `make frontend` command to compile the web application assets.
- Use the `make build` command to build the CLI and embed the web application assets in the Go binary.
- Use the `make test` command to execute the test suite
