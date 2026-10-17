# Walkthrough: search the board and prior rationale before filing an issue

## What was built
Agents now search before they file an issue. The change is in two places:
- The chat system prompt, so that the agent loads the `xpo` skill when it files an issue.
- Step 1 of the skill, so that the skill asks for the searches.

## Why
xpo-31520e restated a case that xpo-35943d had already solved. That happened for two reasons:
1. The Chat Mode prompt told the agent to load the skill only "before you plan, spec or start work". When the agent filed an issue in chat, it did not load the skill.
2. Step 1 of the skill asked only for a `list` search. The `rationale` search came at step 3, spec time, after the issue already existed.

## Changes
- `internal/chat/prompt.md`: the Chat Mode bullet now says "Before you file, plan, spec or start work on an issue, load the `xpo` skill."
- `internal/exponential/agents.go`, `generateWorkflowBody()`: step 1 now says:
  > Before any `add`, search twice: `list` with `match` for issues that already cover the work, and `rationale` for specs and walkthroughs of prior work in the same area. When you propose the issue, tell the user what the searches found, or that they found nothing.

  The sentence asks for a report because a report lets the user check that the agent did the searches. It also shows the user the related work before the user approves the issue.
- Step 3's "Check prior rationale" paragraph stays. It applies at spec time, which can be much later than filing, for example when work starts on an old BACKLOG issue.

## Generated copies
- `.codex/skills/xpo/SKILL.md` and `.opencode/skills/xpo/SKILL.md` were regenerated with `WriteAgentSkill`, so their marker hashes match the new template.
- `.claude/skills/xpo/` has intentional local edits: the STE writing-style experiment in `SKILL.md` and `references/spec-guide.md`. `WriteAgentSkill` reports these files as edited and skips them. We did not force the overwrite, because the experiment would be lost. Step 1 in `.claude/skills/xpo/SKILL.md` was patched by hand. This copy still reports as edited, as it did before.
- `GEMINI.md` is hand-written and has no marker. Its "Discovery" bullet was edited by hand to say the same thing.
- The skill frontmatter `version: "3.1"` did not change. Staleness detection uses the content hash, so installs in other projects show as stale after an upgrade.

## Tests
- `internal/chat/args_test.go`, `TestSystemPromptFor`: expects the new trigger sentence.
- `internal/exponential/agents_test.go`, `TestSkillStep1SearchesBoardAndRationaleBeforeAdd`: cuts step 1 out of `GenerateSkillMD()` and checks for "Before any `add`", "`list` with `match`", "`rationale`" and the reporting sentence. Before the change, both tests failed. After the change, they pass.

## Acceptance criteria
- [x] The Chat Mode prompt says "Before you file, plan, spec or start work on an issue, load the `xpo` skill." Evidence: `internal/chat/prompt.md` line 6, `TestSystemPromptFor` passes.
- [x] Skill step 1 asks for `list` (`match`) and `rationale` before any `add`, and for a report of the results to the user. Evidence: `TestSkillStep1SearchesBoardAndRationaleBeforeAdd` passes.
- [x] The `.codex` and `.opencode` SKILL.md copies match the template. The `.claude` copy keeps its local edits and has the new step 1. `GEMINI.md` says the same. Evidence: `WriteAgentSkill` reported `edited=0` for Codex and OpenCode. The diff shows the new step 1 in `.claude/skills/xpo/SKILL.md` and `GEMINI.md`.
- [x] `make test` passes. Evidence: lint, 1226 frontend tests and all Go packages pass.
