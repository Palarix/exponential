# Beats Agent Instructions

This repository uses `beats`, a local JSONL-based issue tracker, to manage development tasks. As an AI agent, you should use `beats` to understand the current state of the project, plan your work, and record new findings.

## Development Workflow

```bash
# 1. Check for existing tasks
beats ls

# 2a. Check task details if existing entry
beats show <id>

# 2b. Create new entry if not exists
beats add <title>

# 3. Set issue status to in progress
beats start <id>

# 4. Make changes

# 5. Test changes
go test ./...
go build ./...

# 6. Set issue status to done
beats done <id>
```

## Project Memory

`beats` serves as the persistent memory for the project.
- **Start** by reading the backlog to understand what needs to be done.
- **Update** the status of tasks you are working on.
- **Record** any new tasks or bugs you discover as new issues. Do not just fix them implicitly or leave them as TODO comments in code; create a tracked issue so it can be prioritized.
- **Persist** your planning. If a task is too big, break it down into child tasks in `beats`.

## Strict Workflow Rules

1. **No "Ghost" Work**: Any work done by an agent MUST be backed by a beats task/bug/epic.
2. **Missing Tasks**: If no such task exists for your current objective, you must create it.
   - **Timing**: Create the task *after* the user approves your initial design/plan.
3. **In-Progress**: Before starting any code work (editing files), you MUST set the corresponding beats task to `DOING` using `beats start`.
4. **Completion**: You MUST set the beats task to `DONE` using `beats done` *only after* the user approves the final review/walkthrough.

## Agent Identity
When performing actions that modify the tracker (add, update), ensure you are identified as an agent if possible, or use the execution environment's git config.

## Usage Guide

### 1. Discovery (Reading the State)

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

**Create an Epic (High-level goal):**
```bash
./beats add "Refactor Database Layer" --epic --desc "Move from SQLite to Postgres"
```

**Create a Task (Actionable item):**
```bash
./beats add "Create Migration Script" -p <status-id-of-epic> --desc "Write SQL migration"
```

**Filing Bugs/Findings:**
If you encounter a bug or necessary refactor while working on something else, file it immediately so it isn't lost.
```bash
./beats add "Bug: Race condition in login" --desc "Observed when..."
```

### 3. Execution (Updating Status)

**Start a task:**
```bash
./beats start <issue-id>
```

**Mark as Done:**
```bash
./beats done <issue-id>
```

**Update details:**
```bash
./beats update <issue-id> --desc "Updated description with new findings..."
```

## Workflow Example for Agents

1. **Context Check**: Run `./beats list` to see what is PLANNED or DOING.
2. **Backing Task**: Ensure a task exists for your work.
   - If yes: `beats start <id>`.
   - If no: Plan your work, get approval, then `beats add "..."`, then `beats start <id>`.
3. **Implementation**: Modify code, tests, docs.
4. **Review**: Present walkthrough/results to user.
5. **Completion**: On approval, `beats done <id>`.


## Building the project

Use the `make build` command to compile the `beats` binary.