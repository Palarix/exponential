# xpo-e5865f: Improved Agent Instructions Out of the Box

## What

Redesign the CLAUDE.md template that `xpo init` generates (via `GenerateAgentDocs()` in `internal/exponential/agents.go`) so that new projects get a complete, opinionated agent workflow encoding the core principles of deliberate software development with AI agents — without requiring the AiSE kit plugin.

## Why

The current `GenerateAgentDocs()` output is functional but minimal. This repo's hand-crafted CLAUDE.md has evolved into a mature set of guard-rails, and the AiSE kit captures deep thinking about specs-as-design-tools. The goal is "AiSE light": specs and walkthroughs as first-class thinking tools, process guard-rails that prevent ghost work and random walks through the design tree, and enough structure to make agent-driven development deliberate rather than probabilistic — all without requiring the AiSE plugin.

## Acceptance Criteria

- [ ] `GenerateAgentDocs()` produces the updated template
- [ ] Template uses only MCP tools, not CLI references
- [ ] Spec guidance scales (brief for bugs, full for features) — evidenced by the two-tier structure
- [ ] No references to AiSE-specific concepts (ADRs, patterns, learnings, promotion cycles)
- [ ] Template is self-contained — an agent reading only this file understands the full workflow
- [ ] Build/test section uses generic language-agnostic defaults, not placeholders
- [ ] All agents in `AgentRegistry` get the same content (format differences aside)

## Template Content

The following is the target output of `GenerateAgentDocs()`. This replaces the current implementation entirely.

---

```markdown
# Exponential Agent Instructions

This project uses `xpo` (Exponential) as the issue tracker and persistent project memory. As an AI agent, use `xpo` to understand project state, plan work, record findings, and document rationale.

## How to interact with xpo

An `xpo` MCP server is registered in `.mcp.json`. **Always use the MCP tools** — do not shell out to the CLI.

| Action | MCP tool |
|---|---|
| List / search issues | `mcp__xpo__list` |
| Read one issue | `mcp__xpo__show` |
| Create an issue | `mcp__xpo__add` |
| Update fields (status, labels, assignee, story_points, parent) | `mcp__xpo__update` |
| Add a comment | `mcp__xpo__comment` |
| Add a dependency link | `mcp__xpo__link` |
| View audit trail | `mcp__xpo__history` |
| Read/write/delete a spec | `mcp__xpo__spec` |
| Read/write/delete a walkthrough | `mcp__xpo__walkthrough` |
| Manage generic artifacts | `mcp__xpo__artifact` |

Status transitions: `BACKLOG` → `PLANNED` → `DOING` → `BLOCKED` → `DONE`

Transitions are done by calling `update` with the `status` field. When writing descriptions or comments, do not escape non-printing characters.

## Development Workflow

### 1. Discover

- `list` to see the board; use `match` for free-text search.
- `show` for full details on candidate issues.
- Before creating a new issue, always search to ensure no existing issue covers the work.

### 2. Plan

- **Features/design work:** propose the idea to the user. Only create an issue with `add` after the user approves.
- **Bugs found during implementation:** file immediately with `add`, link back to the originating issue via `link`. No approval needed.
- Always set at least one label (`bug`, `feature`, `epic`, or something more specific).
- Keep titles under 100 characters.
- Descriptions are Markdown. Use headings, lists, code blocks, bold/italic, links. Double newlines between paragraphs.
- When a new issue belongs to an epic, set `parent` to the epic's ID.

### 3. Spec (Think Before You Build)

Before implementing, ensure the issue has an up-to-date spec via `mcp__xpo__spec`. If none exists, write one.

**The spec is a thinking tool.** Its purpose is to force deliberation — surfacing branch points, making assumptions explicit, and pruning the design space before committing to code. A spec that merely restates the issue title has failed.

**Scaling the spec to the task:**

For a straightforward bug fix or small change, a spec can be brief:
- What is the problem / what needs to change
- Why (root cause / motivation)
- How (the fix approach, noting any alternatives considered)
- Acceptance criteria (how to verify it worked)

For a feature or complex change, the spec should cover:
- **What** — 2-3 sentence concrete description of what this delivers
- **Why** — motivation, link to parent epic/context
- **Acceptance Criteria** — observable, testable outcomes
- **Flow** — numbered implementation steps; name concrete files, functions, endpoints
- **Decisions** — choices made during design: "X, not Y" with rationale and alternatives
- **Edge Cases** — classified by risk: HIGH (developer decides), MEDIUM (agent proposes handling), LOW (agent handles silently)
- **Assumptions** — what you're taking for granted; the user can correct these
- **Open Questions** — unresolved branch points needing answers before implementation

**Key principles:**
- Draft concrete content. Never present blank templates or ask the user to fill in sections.
- Surface your assumptions explicitly — it is always easier for the user to react to a draft than to author from scratch.
- When you encounter open questions that would significantly change the implementation, ask the user before proceeding.
- The spec is the source of truth for implementation. If you later discover it's wrong or incomplete, update the spec first, then change code.

### 4. Start

- Only issues with `PLANNED` status are eligible for work.
- If the issue you want to work on is in `BACKLOG`, ask the user if it's okay to transition it. Do not start without approval.
- Check the issue's dependencies via `show`. If any `depends_on` or `blocked_by` targets are not `DONE`, flag the unresolved blockers before starting.
- Transition to `DOING` via `update` **before touching any file**.

### 5. Implement & Test

- Follow the spec. The flow steps, decisions, and edge cases are your requirements.
- If you encounter a decision not covered by the spec, make a reasonable choice and note it in the completion comment. If the decision is significant (would surprise the user or constrain future work), stop and ask.
- If you realize the spec has a significant gap or is wrong, stop. Explain the issue, propose a spec update, and wait for approval.
- Use build commands appropriate to the language, framework, or technology of this project.
- Use test commands that ideally execute lint, style checks, and unit tests appropriate for the language, framework, or technology of this project.

### 6. Completion Comment

When implementation is done (evidenced by passing tests), add a brief comment via `mcp__xpo__comment`:

- **Summary** — what changed (use backtick code spans for file/function names)
- **Rationale** — why this approach
- **Decisions made** — any choices not in the original spec
- **Status** — tests passing, ready for review

Keep it concise. This is a signal to the user that work is ready for top-hatting, not the final record.

### 7. Walkthrough (After User Review)

After the user reviews and approves (including any correction rounds), write a walkthrough via `mcp__xpo__walkthrough` with `operation: "write"`.

The walkthrough is the **durable implementation record**. It is written from the perspective of a senior engineer explaining the changes to a junior developer. It should include:

- What was built and why
- How the pieces fit together
- Key decisions and their rationale (including any that emerged during review)
- Anything non-obvious that a future reader would need to understand

Write the walkthrough **after** any user-requested corrections are applied, so it reflects the final state — not the first draft.

### 8. Complete

- Transition to `DONE` via `update` only after: the user approves, tests pass, and the walkthrough is attached.

## Prioritization

When choosing what to work on next (and dependency links don't resolve the order):

1. **Blockers** — issues blocking other work
2. **Bugs** — correctness problems in existing functionality
3. **Planned features** — by dependency order, then by story points (smaller first)

## Strict Rules

1. **No ghost work** — every code change MUST be backed by an xpo issue.
2. **Only planned work** — do not start issues in `BACKLOG` status without explicit user approval to transition.
3. **File what you find** — bugs or follow-up work discovered during a task must be filed as new issues (linked via `mcp__xpo__link`), not left as TODOs in code.
4. **Specs always exist** — scale them to the task, but never skip the thinking step.
5. **Spec is source of truth** — if behavior needs to change, update the spec first, then change code.

## Linking

Use `mcp__xpo__link` to express relationships. Supported types: `blocks`, `blocked_by`, `depends_on`, `dependency_of`, `duplicates`, `duplicated_by`, `relates_to`. When a task spawns follow-up work, link the new issue back to the originating one.

## Usage Notes

### Creating Issues

- Always set at least one label via the `labels` parameter.
- Descriptions render as **Markdown**. Use headings, lists, code blocks, bold/italic.
- Markdown checklists (`- [ ] Title`) can sub-divide task steps.
- When a new issue belongs to an Epic, set `parent` to the epic's ID.

### Specs and Walkthroughs

- Read the spec before implementing: `mcp__xpo__spec` with `operation: "read"`.
- Write the walkthrough after approval: `mcp__xpo__walkthrough` with `operation: "write"`.
- Use `mcp__xpo__artifact` for supplemental files (test outputs, design diagrams, logs).

### Completion Comments

Comments are markdown. A good completion comment is brief:
- What changed (backtick code spans for file/function names)
- Why this approach
- Ready for review

## Agent Identity

When the tracker records who made a change, identify yourself as an agent. Use the form `<Agent Name> <agent@<host>.local>` — e.g. `Claude Code <agent@macbook.local>`. The host portion helps distinguish contributions from different execution environments.
```

---

## Decisions

| Decision | Rationale | Alternatives considered |
|----------|-----------|----------------------|
| Single-pass spec (not interactive loop) | AiSE light — full collaborative spec authoring is for the AiSE plugin | Multi-step spec with Problem Framing → Solution Design → Decision Capture → Edge Cases |
| No ADRs/patterns/learnings infrastructure | Those are AiSE plugin concerns; xpo provides specs and walkthroughs as the knowledge artifacts | `.aise/` directory structure with promotion cycles |
| Bugs filed freely, features need approval | Reduces friction for discovered defects while preventing scope creep | Approval required for everything; nothing requires approval |
| Always generate full template | Easier to delete than to add; users who want less ceremony can trim | Tiered templates (minimal/full); interactive init questionnaire |
| Walkthrough written after top-hat | Captures corrections and final state, not first-draft state | Walkthrough at completion-comment time; no walkthrough (just comments) |
| BACKLOG items need explicit approval to transition | Prevents agents from self-selecting work the user hasn't prioritized | Agents can grab trivial BACKLOG items |
| Generic build/test defaults | `xpo init` has no project introspection; generic language is more useful than blank placeholders the user forgets to fill | Auto-detection of Makefile/package.json; `{{placeholder}}` syntax; commented-out examples |

## Implementation Notes

- The change is entirely in `internal/exponential/agents.go` — replace the body of `GenerateAgentDocs()`.
- The `prefix` parameter is currently unused (the function ignores it). It can remain unused or be removed in a follow-up.
- The `AgentRegistry` loop in `AppendAgentInstructions` handles the file-writing; only the content generation changes.
- Tests in `agents_test.go` will need updating to match the new output.
