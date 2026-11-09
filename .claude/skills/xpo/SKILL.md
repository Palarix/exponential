---
name: xpo
description: >-
  How the xpo (Exponential) MCP tools work: statuses, labels, links,
  milestones, specs, walkthroughs, and starting and merging work. Use
  when you create, update, start or merge issues, write specs or
  walkthroughs, or when the user mentions xpo, issues, epics,
  milestones, or the board.
---

# xpo Reference

How the xpo tools behave: statuses, labels, links, milestones, specs, walkthroughs, start and merge.
When to use them is the project's call: if the project has an `xpo-workflow` skill, load it and follow it.

## Status Graph

`BACKLOG → PLANNED → DOING → DONE; DOING ⇄ BLOCKED; any → CANCELED; any → DUPLICATE`

| Status      | Meaning                                           |
| ----------- | ------------------------------------------------- |
| `BACKLOG`   | Captured but not yet prioritized                  |
| `PLANNED`   | Prioritized and ready to be worked on             |
| `DOING`     | Actively being worked on                          |
| `BLOCKED`   | Waiting on something external                     |
| `DONE`      | Completed                                         |
| `CANCELED`  | Will not be done (obsolete, out of scope, etc.)   |
| `DUPLICATE` | Duplicate of another issue — link to the original |

## Finding Work

`list` shows issues: `match` searches ids, titles, statuses, labels and assignees, and `milestone`
narrows the list to one milestone. `show` gives one issue's description, links and comments, and
`history` its events. `rationale` searches the specs and walkthroughs of earlier work for design
decisions, and `code_history` finds the issues that touched a file or lines.
Full tool list: `references/mcp-tools.md`.

## Issue Creation

- Always set a primary label (see **Issue Labels**). Add secondary labels for area/domain when useful.
- Descriptions and comments render as Markdown. Write literal newlines, not `\n` escape sequences.
- Markdown checklists (`- [ ] Title`) can sub-divide task steps.
- Keep titles under 100 characters.
- When a new issue belongs to an epic, set `parent` to the epic's ID.

## Issue Labels

Every issue MUST have exactly one **primary label** that classifies the type of work:

| Primary label | When to use                                         |
| ------------- | --------------------------------------------------- |
| `bug`         | Something is broken or behaving incorrectly         |
| `feature`     | A new capability that doesn't exist yet             |
| `task`        | A concrete piece of work (refactor, migration, etc) |
| `epic`        | A high-level goal that will be broken into subtasks |

An issue MAY also carry **secondary labels** for area or domain (e.g. `backend`, `frontend`, `ux`, `infra`).
Keep the set small — reuse existing labels before inventing new ones.

## Linking

Use the `link` tool to express relationships. Supported types: `blocks`, `blocked_by`,
`depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`.
Links are bidirectional: `A blocks B` also shows on B as `blocked_by A`. Two issues can have at
most one link, so pick the most specific kind. Linking an already-linked pair fails; pass
`overwrite: true` to replace the existing link. `unlink` removes a link, whichever side stores it.

Moving an issue to DOING is refused while an issue that blocks it is not DONE, whichever side
stores the link. xpo treats the other kinds as informational, but a workflow may gate on them too
(for example on `depends_on`).

## Milestones

A milestone is a scope-bounded group of issues that are done together, numbered `#1`, `#2`, …
with an optional label. Current and Next are the first two open milestones, in order. The
`milestone` tool lists, creates, updates, reorders, completes and deletes them; to complete a
milestone moves its open issues to the next one. To put an issue in a milestone, set `milestone`
on `add` or `update`: a number (`#7`), a label, `@current`, `@next`, or `none`. A new child issue
copies its parent's open milestone.

## Specs

A spec is the design of an issue, written before the code with the `spec` tool. See
`references/spec-guide.md` for the template and how to scale it to the task. `rationale` finds the
specs and walkthroughs of earlier work in the same area.

## Walkthroughs

A walkthrough is the durable record of what was built, written after the implementation with the
`walkthrough` tool. Write it as a senior engineer explaining the changes to a junior developer:

- What was built and why
- How the pieces fit together
- Key decisions and their rationale (including any that emerged during review)
- Anything non-obvious that a future reader would need to understand

Write it from the final state of the code.

## Starting Work

`start` transitions an issue to DOING, assigns it to the person you are working for, and creates an
isolated git worktree on a new branch for it. Never set `assignee` to yourself — xpo records that
an agent did the work. `start` refuses issues in BACKLOG or BLOCKED, and issues that are blocked
(see **Linking**). `force: true` takes over an issue someone else already has in DOING.

All file reads, edits, builds, and test runs must happen inside the worktree path returned by
`start` — not the main checkout. If you are already working in the issue's worktree, don't call
`start` again.

## Merging

Commit everything in the issue's worktree first: `git add -A && git commit -m "<issue-id>: <issue title>"`.
Then call the `merge` tool without a `strategy` (the user may ask for one). It squash-merges the
branch into the default branch, records a MERGE event, closes the issue, and removes the worktree and
branch. It refuses to run while the worktree has uncommitted or untracked files.

If it fails because the default branch moved on (conflicts), bring the default branch into the
issue's branch from the worktree (`git rebase <default-branch>`, or `git merge <default-branch>`),
resolve any conflicts, re-run the tests, commit, and call `merge` again.

Committing on the issue's worktree is your job. Merging is `merge`'s job: never run `git push`,
or merge, rebase or commit on the default branch yourself. Rebasing the issue's branch onto the
default branch (or merging the default branch into it) from the worktree is allowed.

<!-- xpo:skill 1.5.0 sha256:7f22f47d40f9 -->
