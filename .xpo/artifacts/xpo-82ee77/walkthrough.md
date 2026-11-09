# Walkthrough: Factory workflow template

## What was built and why

`xpo init` now offers a second workflow template, **Factory** (`--workflow factory`). The guided workflow stops for the human on every issue. The factory workflow delivers a whole milestone and stops only at its boundaries. The human approves the milestone backlog at the start and tophats the milestone at the end. Between those two points, one sub-agent answers the open questions in a spec, a second sub-agent reviews each change, and the agent merges approved work.

The template ships two sub-agent briefs as reference files: `references/open-questions.md` and `references/code-review.md`. It needs an agent that can start a sub-agent with a fresh context. The init prompt says so, and the skill opens with a **Requirements** paragraph.

## How the pieces fit together

### Templates can carry reference files

- `WorkflowTemplate` has a new field, `References func() map[string]string` (file name to content). Guided leaves it nil.
- `workflowFiles(agent, tpl)` in `workflow_install.go` returns every file that a template writes: `SKILL.md` plus `references/<name>`. `InstallWorkflow` (first init), `SetupWorkflow` (`xpo init --workflow`) and `ComputeInitPlan` all use it. So the change plan lists exactly the files that init writes.
- The inline workflow for agents without skills carries the briefs itself, as `### Brief: …` subsections after Prioritization.

### The factory template (`internal/exponential/workflow_factory.go`)

- `factoryWorkflowSkill()` and `factoryWorkflowInline()` wrap `factoryWorkflowBody(ref, oq, cr)`. `ref` names where the tool mechanics are documented ("the `xpo` skill" or "the xpo section above"). `oq` and `cr` name where the briefs are (`references/…` or "the brief below").
- `publicByDefault()` moved out of the guided body. Both templates call it.
- `factoryRules()`: the human's approval of a milestone backlog covers planning and starting its issues. A BACKLOG issue outside the started milestone still needs the human.
- `openQuestionsBrief()` and `codeReviewBrief()` return the brief text without a heading. The reference file adds a `#` heading, and the inline version adds a `###` heading. The prompt sits in a fenced `text` block, so the inline version has no `---` rules.

### Planning model

- Milestones plan targets: a label, and a one-line goal that includes the **Done when**. The workflow has no "milestone epic".
- Stories (`feature`, `bug`, `task`) have 5 story points or fewer. A bigger story becomes an `epic` with child issues in the same milestone. The agent never starts an epic. It sets the epic DONE when the last child merges.
- The board holds the state:
  - A milestone has not started while all its stories are BACKLOG.
  - On backlog approval, the agent sets every BACKLOG issue in the milestone to PLANNED.
  - A started milestone with no PLANNED or DOING story waits for its tophat, or for BLOCKED issues.
- Cuts to pay back later become BACKLOG issues in no milestone. Traps become comments on the issues they affect.

### The review loop (step 6 and `code-review.md`)

This part was tuned after the first draft. A reviewer that hears only that "a problem you miss ships" always reports something. A fresh reviewer in each round then finds new corner cases, and the loop runs to its limit. So:

- The brief states the cost of a false alarm. It says that "PASS with no findings" is a normal result.
- The mechanical checks come first and keep a hard bar: AC coverage, tests that cannot fail, decision records and project rules, test quality.
- Any other blocker or major must name a realistic trigger: who does what, in normal use of the product. A problem that needs several unlikely conditions together is minor, or the reviewer drops it.
- **Strict areas** keep a stricter bar: authentication, authorization and permissions, secrets and credentials, and data loss. There, an unlikely path that an attacker can cause can be a blocker.
- Only a blocker starts another round. Round 2 and later check only the previous findings, the fix diff, and the code that the fixes touched.
- The implementer may reject a finding when the trigger is not realistic, except in the strict areas. Each rejection goes in the completion comment and in the tophat **Deferred** list.

## Key decisions

- **Name: `factory`.** The skill folder stays `xpo-workflow`, like every template.
- **Nothing project-specific.** The text says "the project's test and lint commands", "the project's instructions" and "the project's decision records". It names no sub-agent types and no products. If a project keeps no decision records, decisions go into the spec Decisions and the tophat report.
- **Approving a tophat does not approve the next milestone.** The next milestone gets its own backlog approval in step 2.
- **Specs always record Agent Decisions.** Inside a milestone, the agent does not wait for spec approval.
- **Merge mechanics stay in the reference.** Step 8 commits, calls `merge`, and points to **Merging**.

## Non-obvious things

- Epics do not count for the milestone state. An approved backlog sets them to PLANNED too, but the agent never starts them. So "no story is PLANNED or DOING" is the tophat condition, not "no issue".
- The project's test and lint commands still run on the whole change in every round. Only the review narrows to the fixes.
- The template tests for US spelling, Public by default and the `xpo:skill` marker now loop over `WorkflowTemplates` and their reference files. A third template gets these checks automatically.

## Acceptance Criteria

### Registry and init
- [x] Second entry `factory` / "Factory"; guided stays first. Evidence: `TestFactory_Registered`.
- [x] The init prompt description says it needs sub-agents. Evidence: `TestFactory_Registered`.
- [x] `--workflow factory` works. Evidence: manual run of `xpo init --workflow factory --yes` in a scratch repo; `WorkflowNames` in `TestFactory_Registered`.

### Reference files
- [x] `References` field; guided has none. Evidence: `TestFactory_References`.
- [x] `InstallWorkflow` and `SetupWorkflow` write them. Evidence: `TestFactory_InstallWritesReferences`, `TestFactory_SetupWritesReferences`.
- [x] `ComputeInitPlan` lists them. Evidence: `TestFactory_InstallWritesReferences`; manual run (the plan lists `xpo-workflow/references/*.md` for each agent).

### Planning model
- [x] No milestone epic; Done when in the goal; stories up to 5 points; epics with children, not started; cuts and traps. Evidence: `TestFactory_PlanningModel`.

### Skill content
- [x] Requirements and Rules. Evidence: `TestFactory_RequirementsAndRules`.
- [x] Public by default from one function. Evidence: `TestPublicByDefault_InWorkflowNotReference` (all templates).
- [x] When to stop, Orient, Plan the milestone (PLANNED on approval, in-scope work as PLANNED), tophat report, After the tophat. Evidence: `TestFactory_PlanningModel`.
- [x] All steps and sections present. Evidence: `TestFactory_Skill`.

### Step 6: review rounds
- [x] Only a blocker starts another round; majors fixed or filed; minors fixed, filed or dropped. Evidence: `TestFactory_ReviewRounds`.
- [x] Later rounds check only the fixes, with the fix diff. Evidence: `TestFactory_ReviewRounds`, `TestFactory_CodeReviewIsCalibrated`.
- [x] "Not realistic" is a valid rejection, except in the strict areas; rejections go in the completion comment and the tophat Deferred list. Evidence: `TestFactory_ReviewRounds`.
- [x] BLOCKED after 3 rounds that still have a blocker. Evidence: `TestFactory_ReviewRounds`.

### Briefs
- [x] Open-questions brief: placeholders and answer format. Evidence: `TestFactory_Briefs`.
- [x] Code-review brief: checks (mechanical first), test and lint commands, verdict, AC table, severities. Evidence: `TestFactory_Briefs`.
- [x] Both edit no file. Evidence: `TestFactory_Briefs`.
- [x] Balanced framing, PASS is normal, realistic trigger, no unlisted inputs, strict areas, round 2+ variant. Evidence: `TestFactory_CodeReviewIsCalibrated`.

### Nothing project-specific
- [x] No project paths, commands, names, sub-agent types or products; generic phrases present; no reference half, metadata or marker. Evidence: `TestFactory_NothingProjectSpecific`.
- [x] Fallback when a project keeps no decision records. Evidence: `TestFactory_NothingProjectSpecific`.

### Inline version
- [x] `## Workflow` with Requirements, the workflow, and both briefs as subsections; no `references/` links. Evidence: `TestFactory_Inline`.

### Tests
- [x] The generic template tests run over every template. Evidence: `TestUSSpelling`, `TestPublicByDefault_InWorkflowNotReference`.
- [x] `make test` passes.
