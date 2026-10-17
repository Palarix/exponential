# Search the board and prior rationale before filing an issue

## What
Agents must search before they file. Two changes:
1. The chat prompt tells the agent to load the `xpo` skill before *filing* an issue too, not only before planning, specs or starting work.
2. Step 1 of the skill asks for two searches before any `add`, `list` with `match` and `rationale`. The agent then reports what the searches found when it proposes the issue.

## Why
xpo-31520e restated a case that xpo-35943d had already solved. In Chat Mode the skill is not loaded when an agent files an issue. Even when the skill is loaded, step 1 asks only for a `list` search, and `rationale` comes later, at spec time. Prior specs and walkthroughs therefore never come up while the issue is being written.

## How
- `internal/chat/prompt.md` (Chat Mode bullet): "Before you file, plan, spec or start work on an issue, load the `xpo` skill."
- `internal/exponential/agents.go`, `generateWorkflowBody()` step 1: replace the line "Before creating a new issue, search to ensure no existing issue already covers the work." with:

  > Before any `add`, search twice: `list` with `match` for issues that already cover the work, and `rationale` for specs and walkthroughs of prior work in the same area. When you propose the issue, tell the user what the searches found, or that they found nothing.

  Step 3's "Check prior rationale" paragraph stays. It covers spec time, which is a different moment.
- Regenerate `.codex/skills/xpo/SKILL.md` and `.opencode/skills/xpo/SKILL.md` from the template with `WriteAgentSkill`.
- `.claude/skills/xpo/` has intentional local edits (the STE writing-style experiment in `SKILL.md` and `references/spec-guide.md`), so `WriteAgentSkill` reports it as edited. Do not overwrite it. Patch step 1 in its `SKILL.md` by hand.
- `GEMINI.md` is hand-written and has no marker. Edit its "Discovery" bullet by hand to say the same thing.
- Tests first:
  - `internal/chat/args_test.go`: expect the new trigger sentence.
  - `internal/exponential/agents_test.go`: new test. Step 1 of `GenerateSkillMD()` names `rationale`, and it tells the agent to report what the searches found.

## Acceptance criteria
- [ ] The Chat Mode prompt says "Before you file, plan, spec or start work on an issue, load the `xpo` skill."
- [ ] Skill step 1 asks for `list` (`match`) and `rationale` before any `add`, and for a report of the results to the user.
- [ ] The `.codex` and `.opencode` SKILL.md copies match the template. The `.claude` copy keeps its local edits and has the new step 1. `GEMINI.md` says the same.
- [ ] `make test` passes.

## Assumptions
- The skill's frontmatter `version: "3.1"` stays the same. Staleness is tracked by the content hash, not by that field.
