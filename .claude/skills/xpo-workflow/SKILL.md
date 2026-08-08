---
name: xpo-workflow
description: >-
  Issue-tracking workflow for this repository using xpo (Exponential).
  Use before planning, implementing, fixing, or completing any code
  change; when creating or updating issues, specs, or walkthroughs;
  or when the user mentions xpo, issues, epics, or the board.
metadata:
  author: exponential
  version: "1.0"
---

# xpo Development Workflow

### 1. Discover

- Call `list` on the xpo MCP server to see the board; use the `match` parameter for free-text search.
- Call `show` for full details on candidate issues.
- Before creating a new issue, always search to ensure no existing issue covers the work.

### 2. Plan

- **Features/design work:** propose the idea to the user. Only create an issue after the user approves.
- **Bugs found during implementation:** file immediately, link back to the originating issue. No approval needed.
- Always set at least one label (`bug`, `feature`, `epic`, or something more specific).
- Keep titles under 100 characters.
- Descriptions are Markdown. Use headings, lists, code blocks, bold/italic. Write literal newlines, not `\n` escape sequences.
- When a new issue belongs to an epic, set `parent` to the epic's ID.

### 3. Spec (Think Before You Build)

Before implementing, ensure the issue has an up-to-date spec. If none exists, write one using the
xpo MCP server's `spec` tool. See `references/spec-guide.md` for the full spec template and scaling rules.

The spec is a thinking tool — it collapses the design space so the user can steer decisions
instead of discovering unwanted choices after the code is written. Scale the depth to the task:
a bug fix gets a brief What/Why/How/AC spec; a feature gets the full template.

**Interactive mode** (user present — CLI, IDE plugin): after writing the spec, surface open questions
and uncertainties to the user and wait for answers before implementing. The spec is a conversation
artifact — the user should confirm or redirect before code is written.

**Non-interactive mode** (headless — xpo drive, piped prompts): implement from the spec without
waiting, BUT the spec must explicitly mark where you made judgment calls. For every open question,
document the question, the answer you chose, and your reasoning. Use a dedicated section:

- **Agent Decisions** — choices made on behalf of the user where the answer was not obvious.
  Format each as: the question, the choice made, the reasoning, and what to revisit if the
  choice was wrong.

### 4. Start

- Only issues with PLANNED status are eligible for work.
- If the issue you want to work on is in BACKLOG, ask the user if it is okay to transition it. Do not start without approval.
- Check the issue's dependencies. If any `depends_on` or `blocked_by` targets are not DONE, stop and ask the user how to proceed.
- If the issue is already in DOING and assigned to someone else, stop and ask the user before taking it over.
- Call `start` with the issue ID **before touching any file**. This transitions the issue to DOING and creates an isolated git worktree for your changes. After starting, set `assignee` to yourself via `update`.
- The `start` tool returns a `worktree_path` — **all file reads, edits, and commands must happen inside that directory**, not in the main checkout. The main checkout stays on the default branch.
- Each issue gets its own worktree branching from the default branch. This means multiple issues can be worked on in parallel without interfering with each other.
- To resume or reclaim an issue that is already in DOING, call `start` with `force: true`.

### 5. Implement & Test

- Work inside the worktree path returned by `start`. All file reads, edits, builds, and test runs must target that directory — not the main checkout.
- Follow the spec. The flow steps, decisions, and edge cases are your requirements.
- If you encounter a decision not covered by the spec, make a reasonable choice and note it in the completion comment. If the decision is significant (would surprise the user or constrain future work), stop and ask.
- If you realize the spec has a significant gap or is wrong, stop. Explain the issue, propose a spec update, and wait for approval.
- Use build commands appropriate to the language, framework, or technology of this project.
- Use test commands that ideally execute lint, style checks, and unit tests appropriate for the language, framework, or technology of this project.
- Use the `artifact` tool for supplemental files (test outputs, design diagrams, logs).

### 6. Completion Comment

When implementation is done (evidenced by passing tests), add a brief comment:

- **Summary** — what changed (use backtick code spans for file/function names)
- **Rationale** — why this approach
- **Decisions made** — any choices not in the original spec
- **Status** — tests passing, ready for review

The comment says "ready." The walkthrough says "how it works." Do not duplicate one into the other.

**Stop here and wait.** Do not write the walkthrough, commit, or merge until the user has reviewed
the changes and responded. The completion comment is a handoff — the user needs to test the changes
before anything is finalized. Proceeding without user approval risks wasting work on the walkthrough
and creating commits or merges the user will want reverted.

### 7. Review & Spec Update

When the user top-hats and requests changes, update the spec to reflect the corrections **before**
modifying code. The spec must always match the final implementation. If the user's feedback changes
the approach, acceptance criteria, or decisions, those changes belong in the spec — not just in the
code diff. Without this step, the spec drifts from reality and future agents reading it will build
the wrong thing.

### 8. Walkthrough (After User Review)

**Prerequisite:** the user has explicitly approved the changes. If the user has not responded since
your completion comment, you are NOT on this step — go back to step 6 and wait.

After the user reviews and approves (including any correction rounds), write a walkthrough using the
xpo MCP server's `walkthrough` tool.

The walkthrough is the **durable implementation record** — written from the perspective of a senior
engineer explaining the changes to a junior developer:

- What was built and why
- How the pieces fit together
- Key decisions and their rationale (including any that emerged during review)
- Anything non-obvious that a future reader would need to understand

Write the walkthrough **after** any user-requested corrections are applied, so it reflects the final state.

### 9. Complete

Do NOT transition to DONE without a walkthrough. If the user asks to close the issue, write the
walkthrough first — it is the only durable record of what was built and why. Without it, all
context about the implementation is lost when the conversation ends.

When worktrees are enabled, use `merge` to complete the issue. It merges the issue branch into
the default branch from the hub checkout, records a MERGE event, closes the issue, and cleans up
the worktree automatically.

**Gate:** commit and merge only when ALL of the following are true:
1. The user has explicitly approved the changes (not just silence after your completion comment)
2. Tests pass
3. The walkthrough is attached

If any of these are missing, do not commit or merge — go back to the missing step.

## Status Graph

`BACKLOG → PLANNED → DOING → DONE; DOING ⇄ BLOCKED`

BLOCKED is not a step on the way to DONE — it is a side-state an issue can enter and leave.

## Prioritization

When choosing what to work on next (and dependency links do not resolve the order):

1. **Blockers** — issues blocking other work
2. **Bugs** — correctness problems in existing functionality
3. **Planned features** — by dependency order, then by story points (smaller first)

## Linking

Use the `link` tool to express relationships. Supported types: `blocks`, `blocked_by`,
`depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`.
When a task spawns follow-up work, link the new issue back to the originating one.

## Issue Creation

- Always set at least one label.
- Descriptions and comments render as Markdown. Write literal newlines, not `\n` escape sequences.
- Markdown checklists (`- [ ] Title`) can sub-divide task steps.
