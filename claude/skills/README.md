# Exponential Skills for Claude Code

These skills teach Claude Code how to use the `xpo` issue tracker in your project.

## Installation

Copy the skill folders into your project's `.claude/skills/` directory:

```bash
# From your project root
cp -r path/to/xpo-cli/.claude/skills/xpo* .claude/skills/
```

## Available Skills

### Agent-invoked (automatic)

| Skill | Description |
|-------|-------------|
| `xpo/` | Core workflow rules and command reference. Auto-loads when Claude is doing development work. |

### User-invoked (slash commands)

| Skill | Command | Description |
|-------|---------|-------------|
| `xpo-backlog/` | `/xpo-backlog` | Review project state: in-progress, blocked, planned, and backlog |
| `xpo-work/` | `/xpo-work` | Full workflow: discover, start, implement, test, document, complete |
| `xpo-file/` | `/xpo-file` | File a new bug, task, feature, or epic with duplicate checking |

## How It Works

- **`xpo/`** has no `disable-model-invocation` flag, so Claude auto-loads it whenever its description matches the current context (i.e., any development work in a xpo-tracked project). This gives the agent the workflow rules and command reference without the user having to ask.

- **`xpo-backlog/`**, **`xpo-work/`**, and **`xpo-file/`** set `disable-model-invocation: true`, making them explicit user commands. The user types `/xpo-work` to kick off the disciplined task workflow, `/xpo-backlog` to review the board, etc.

## Pairing with CLAUDE.md

For projects that want always-on behavioral rules beyond what the auto-invoked skill provides, add a section to your project's `CLAUDE.md`:

---
This project uses `xpo` for issue tracking. Before making code changes:

1. Ensure a xpo issue exists for the work (`xpo ls`)
2. Mark it as in-progress (`xpo start <id>`)
3. When done, add a summary comment and mark complete (`xpo done <id>`)

### Story Format
For xpo stories we will use the following format that fosters human-agent collaboration. Fill in this template, don't blindly copy the example below.

```markdown
### User story
As a **[type of user]**, I want **[capability]**, so that **[outcome / value]**.

### Why now
One or two sentences. What's the trigger? Customer ask, metric, dependency unblocked, regulatory deadline? Helps future-you remember the priority rationale.

### Acceptance criteria (behavioural)
Written for humans to agree on intent. Keep these outcome-focused, not implementation-focused. The agent-section translates these into testable specs.

### Out of scope
Explicit non-goals. Cheaper to state here than to argue about in review.

### Open questions
Anything unresolved that blocks handoff to an agent. An item should not move to "ready" with open questions in this section.

### Context pointers
**Primary files / modules:**
- `path/to/file.ext` — [what this is, why it matters]

**Related code worth reading first:**
- `path/to/related.ext` — [pattern to follow / similar feature]

**Relevant docs:**
- [link or path] — [what to look for]

**Data model touched:**
- [tables, schemas, types, or "none"]

### Architectural constraints
- **Patterns to follow:** [e.g., repository pattern used in `src/services/*`, error handling via `Result<T, E>`]
- **Libraries to use / avoid:** [e.g., use existing `httpClient`, do not add new HTTP library]
- **Naming conventions:** [link to style guide or call out specifics]
- **Where new code belongs:** [directory, module, layer]

### Hard constraints (do not violate)
Things that will cause a rejected PR regardless of whether tests pass.

- Do not modify:
- Do not introduce dependencies on:
- Must remain backward-compatible with:
- Must not change public API of:

### Acceptance criteria (testable)
Translate Part 1's behavioural criteria into checkable specs.
Prefer concrete examples and named tests over prose.

**Behavioural tests to add/pass:**
- `test_name_describing_behaviour` — given X, when Y, then Z

**Example inputs and expected outputs:**
```
input:  …
output: …
```

**Edge cases that must be handled:**

### Non-functional requirements
The stuff agents silently default if you don't say.
- **Performance:** [e.g., p95 < 200ms for endpoint X, or "no regression vs baseline"]
- **Error handling:** [expected failure modes and how they should surface]
- **Logging / observability:** [what events to emit, at what level]
- **Security:** [auth requirements, input validation, data handling]
- **Accessibility / i18n:** [if UI]

### Definition of done (machine-checkable)
- [ ] All new and existing tests pass
- [ ] Lint and type checks pass
- [ ] [Project-specific: e.g., `make verify` succeeds]
- [ ] Acceptance tests above are present and green
- [ ] No new dependencies added (or: new deps documented in PR description)
- [ ] Public API changes documented in `…`

### Suggested approach (optional)
A nudge, not a mandate. Use when you have a strong opinion about implementation strategy, or when the obvious path is wrong. Leave blank to let the agent plan.

### Handoff notes for the agent
Anything that doesn't fit above. Known gotchas, things that look wrong but aren't, historical context. "The retry logic in X looks redundant but is load-bearing for the Y integration — don't remove it."

```
---