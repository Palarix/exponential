# Factory workflow template

## What

`xpo init` offers a second workflow template, **Factory**. It delivers one milestone at a time without waiting for the human. The human approves the milestone backlog at the start and tophats the milestone at the end. Between those two points, sub-agents answer open questions and review the change, and the agent merges approved work. The template ships two sub-agent briefs as reference files: `references/open-questions.md` and `references/code-review.md`.

## Why

The guided workflow stops for the human on every issue. Some projects want the agent to deliver a whole milestone and to stop only at its end. Today such a project writes that workflow by hand. With a template, `xpo init --workflow factory` sets it up in one step.

## Acceptance Criteria

### Registry and init

- [ ] `WorkflowTemplates` has a second entry: `Name` `factory`, `Title` "Factory". Guided stays first and stays the default.
- [ ] The `Description` in the init prompt says that the template needs an agent with sub-agents.
- [ ] `--workflow factory` works on first init and on an initialized project.

### Reference files

- [ ] `WorkflowTemplate` has a `References` field: file name to content. Guided has none.
- [ ] `InstallWorkflow` and `SetupWorkflow` write each reference file to `xpo-workflow/references/<name>`.
- [ ] `ComputeInitPlan` lists the reference files on first init.

### Planning model

- [ ] Milestones are for target planning: a label, a one-line goal and, if useful, a target date. The workflow uses no "milestone epic".
- [ ] Epics and stories decompose the work. A story is a `feature`, `bug` or `task` issue with story points.
- [ ] A story over 5 story points becomes an `epic`. The agent splits it into child issues of 5 points or fewer, in the same milestone. The agent does not start an epic. An epic is DONE when all its children are DONE.
- [ ] The milestone's **Done when** is part of its goal.
- [ ] A cut to pay back later becomes a BACKLOG issue in no milestone, linked to the issue it came from.
- [ ] A trap that later issues must know about becomes a comment on each issue it affects.

### Skill content

- [ ] A **Requirements** paragraph near the top: the workflow needs an agent that can start a sub-agent with a fresh context. An agent that cannot stops and tells the user.
- [ ] **Rules**: every code change has an issue in DOING; a milestone starts only after the human approves its backlog; a BACKLOG issue outside the started milestone is never started without the human; the guided bug rule (Rule 3).
- [ ] **Public by default**: the same section as in guided. One function produces it for both templates.
- [ ] **When to stop**: stop only to get the milestone backlog approved, at milestone completion, or when every story left in the milestone is BLOCKED. A table says how to handle all other situations without stopping.
- [ ] Step 1 Orient: the milestone awaits its tophat when it has started, no story in it is PLANNED or DOING, and it is not completed. The agent then stops and repeats the tophat report.
- [ ] Step 2 Plan the milestone: if the Current milestone has no issues, the agent splits its goal into stories in BACKLOG in the milestone, and stops for approval. When the human approves the backlog and starts the milestone, the agent sets every BACKLOG issue in it to PLANNED.
- [ ] In-scope work found during a started milestone is filed as PLANNED in the milestone. Out-of-scope work goes to BACKLOG in no milestone.
- [ ] Steps: 1 Orient, 2 Plan the milestone, 3 Spec (open questions to a sub-agent), 4 Start, 5 Implement and test, 6 Adversarial review, 7 Spec stays true, 8 Walkthrough, commit, merge, 9 Next issue or hand off, 10 Milestone completion and tophat, After the tophat. Then Backlog Review and Prioritization.
- [ ] Step 10 gives the tophat report to the human: Delivered, How to tophat, Agent Decisions, Proposed decision records, Deferred (including rejected review findings). After approval, the agent completes the xpo milestone.

### Step 6: review rounds

- [ ] Only a **blocker** starts another round. The agent fixes each major or files it as a follow-up issue. The agent fixes each minor if it is cheap, or else files or drops it.
- [ ] Round 2 and later check only the fixes and the code that the fixes touched. The agent gives the new reviewer the previous findings and the fix diff. The new reviewer does not review the whole change again.
- [ ] Valid reasons to reject a finding: it is wrong, the spec says otherwise, or its trigger is not realistic in normal use of the product. The agent writes each rejection and its reason in the completion comment. The tophat report lists them under **Deferred**.
- [ ] A "not realistic" rejection is not valid for a finding in the strict areas (see Briefs).
- [ ] After 3 rounds that still have a blocker, the agent sets the issue BLOCKED with the open findings.

### Briefs

- [ ] `open-questions.md`: how to start the sub-agent, and a prompt with placeholders (issue ID, path, milestone goal, spec, questions). The answer format: Answer, Evidence, Alternatives rejected, Confidence, Human's call.
- [ ] `code-review.md`: the same structure. The sub-agent runs the project's test and lint commands. Verdict, AC table and findings by severity.
- [ ] Both briefs tell the sub-agent to edit no file.
- [ ] The review framing is balanced. The brief says that a false alarm has a cost: the implementer fixes code that did not need it. It says that "PASS, no findings" is a normal result.
- [ ] Mechanical checks come first and need no realistic trigger: an AC with no test, a test that still passes when the behavior is broken, a violated decision record or project rule.
- [ ] Every **blocker** and **major** finding names a realistic trigger: who does what, in normal use of the product, that causes it. A finding that needs several unlikely conditions together is **minor**, or the reviewer drops it.
- [ ] The brief does not ask for inputs that the spec does not list. Correctness covers the spec's edge cases and the inputs of normal use.
- [ ] Strict areas: authentication, authorization and permissions, secrets and credentials, and data loss. There, an unlikely path that an attacker can cause, or that loses data, can be a blocker. The finding names the path.
- [ ] Round 2+ variant: the prompt has a placeholder for the previous findings and the fix diff. It tells the reviewer to check only those, and to report a new problem only in the code that the fixes touched.

### Nothing project-specific

- [ ] No `make`, `docs/adr/`, `plan.md`, `CLAUDE.md`, ADR numbers, project names, UI themes or code paths. The text says "the project's test and lint commands", "the project's instructions" and "the project's decision records".
- [ ] If the project keeps no decision records, the agent writes the decision into the spec Decisions and flags it in the tophat report.
- [ ] No sub-agent type names (for example `general-purpose`) and no product names. The text says "a fresh sub-agent, not a fork of your context".
- [ ] No tool mechanics: statuses, labels, linking and merge conflicts refer to the reference (`the xpo skill` or `the xpo section above`).
- [ ] No reference half (`---` + `## Reference`), no `metadata` block, no `xpo:skill` marker.

### Inline version

- [ ] Agents without skill support get `## Workflow` with the Requirements, the workflow, and both briefs as `### Brief: …` subsections. The steps point to "the brief below" instead of `references/…`.

### Tests

- [ ] The existing template tests (US spelling, Public by default, no `xpo:skill` marker) run over every template, not only guided.
- [ ] New tests cover each item above. `make test` passes.

## Flow

1. Add `References func() map[string]string` to `WorkflowTemplate`.
2. Move Public by default to `publicByDefault()`. Call it from both templates.
3. Add `factoryWorkflowSkill`, `factoryWorkflowInline` and `factoryWorkflowBody(ref, oq, cr)`. `oq` and `cr` name where the briefs are.
4. Add `openQuestionsBrief()` and `codeReviewBrief()`.
5. Register the template.
6. Write the reference files in `InstallWorkflow` and `SetupWorkflow`. List them in `ComputeInitPlan`.
7. Add a CHANGELOG entry.

## Decisions

- **The template name is `factory`.** The skill stays `xpo-workflow`, like every template.
- **Milestones plan targets; epics and stories decompose work.** A milestone has no epic of its own. The milestone goal holds the **Done when**.
- **Stories are 5 story points or fewer.** A bigger story becomes an epic with children. The agent works only on the children.
- **The human approves the milestone backlog.** Then the agent sets its BACKLOG issues to PLANNED. That approval covers in-scope work found during the milestone.
- **The milestone state is read from the board.** It has not started while all its stories are BACKLOG. It awaits its tophat when it has started, no story is PLANNED or DOING, and it is not completed. Epics do not count. No status or comment marks the state.
- **Approving a tophat does not approve the next milestone.** The next milestone gets its own backlog approval in step 2.
- **Specs always record Agent Decisions.** The agent does not wait for spec approval inside a milestone.
- **The review is calibrated, not maximal.** A reviewer that is told only that a miss ships always reports something. Then a fresh reviewer each round finds new corner cases, and the loop runs to its limit. So the brief states the cost of a false alarm, requires a realistic trigger for blockers and majors, and lets the implementer reject unrealistic findings. The human checks those rejections at the tophat.
- **Mechanical checks keep a hard bar.** A missing test, a test that cannot fail, or a broken rule does not depend on the reviewer's taste.
- **Security and data loss keep a strict bar.** There, an unlikely path is what an attacker looks for, and lost data does not come back.
- **Later rounds verify fixes only.** This keeps the round count bounded by the fixes, not by the reviewer's creativity.
- **The review sub-agent gets no assessment from the implementer**, so its review is not anchored. This rule is in the workflow and in the brief.
- **Requirements name the capability, not products.** A product list goes out of date.
- **No writing-style file.** That is a preference of one project, not part of the template.
- **Merge mechanics stay in the reference.** Step 8 says: commit, then call `merge` (see **Merging**). It keeps "never push, never commit on the default branch".

## Edge Cases

- **MEDIUM — an agent without sub-agents.** The Requirements paragraph tells it to stop and tell the user.
- **MEDIUM — a story without story points.** The agent estimates it in step 2 and splits it if it is over 5.
- **MEDIUM — a fix in round 2 that breaks code outside the fix.** The project's test and lint commands still run on the whole change in every round.
- **LOW — a project that changes its `xpo-workflow` later.** xpo does not touch it. `--workflow` replaces it after confirmation (existing behavior).
- **LOW — an extra file in `xpo-workflow/references/`.** `SetupWorkflow` deletes the folder first, so the extra file goes (existing behavior).
