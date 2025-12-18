# Beats Agent Instructions

This repository uses `beats`, a local JSONL-based issue tracker, to manage development tasks. As an AI agent, you should use `beats` to understand the current state of the project, plan your work, and record new findings.

## Core Philosophy: Project Memory

`beats` serves as the persistent memory for the project. 
- **Start** by reading the backlog to understand what needs to be done.
- **Update** the status of tasks you are working on.
- **Record** any new tasks or bugs you discover as new issues. Do not just fix them implicitly or leave them as TODO comments in code; create a tracked issue so it can be prioritized.
- **Persist** your planning. If a task is too big, break it down into child tasks in `beats`.

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
2. **Selection**: Pick a task (e.g., `task-a1b2c3`).
3. **Deep Dive**: Run `./beats show task-a1b2c3`.
4. **Status Update**: Run `./beats start task-a1b2c3`.
5. **Implementation**: Modify code.
6. **Task Discovery**: You realize the `User` model is missing a field.
    - Run `./beats add "Add email field to User model" -p <parent-epic-id>`.
7. **Completion**: Run `./beats done task-a1b2c3`.


## Building the project

Use the `make build` command to compile the `beats` binary.