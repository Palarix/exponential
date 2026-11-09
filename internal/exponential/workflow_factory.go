package exponential

import "strings"

// The factory workflow (xpo-82ee77): the agent delivers one milestone at a
// time. The human approves the milestone backlog at the start and tophats
// the milestone at the end. Sub-agents answer open questions and review the
// change in between. It ships two sub-agent briefs as reference files.

const (
	openQuestionsFile = "open-questions.md"
	codeReviewFile    = "code-review.md"
)

func factoryWorkflowSkill() string {
	return md(`---
name: xpo-workflow
description: >-
  This project's workflow for xpo issues: autonomous delivery, one
  milestone at a time. Use before planning, implementing, fixing, or
  completing any code change; when filing issues or writing specs or
  walkthroughs; when a milestone starts or ends; and when the user asks
  to review the board.
---

# Workflow

This is how this project works with xpo issues. The project owns this file: ¦xpo init¦ wrote it
once and xpo never changes it. For how the xpo tools work (statuses, labels, links, milestones,
specs, walkthroughs, starting and merging), load the ¦xpo¦ skill.

`) + factoryWorkflowBody("the ¦xpo¦ skill",
		"the brief in ¦references/"+openQuestionsFile+"¦",
		"the brief in ¦references/"+codeReviewFile+"¦")
}

func factoryWorkflowInline() string {
	return md(`## Workflow

This is how this project works with xpo issues. The project owns this section: ¦xpo init¦ wrote
it once and xpo never changes it. For how the xpo tools work, see the xpo section above.

`) + factoryWorkflowBody("the xpo section above",
		"the **Brief: open questions** below",
		"the **Brief: adversarial code review** below") +
		"\n" + md("### Brief: open questions\n\n"+openQuestionsBrief()) +
		"\n" + md("### Brief: adversarial code review\n\n"+codeReviewBrief())
}

func factoryReferences() map[string]string {
	return map[string]string{
		openQuestionsFile: md("# Brief: answering a spec's open questions\n\n" + openQuestionsBrief()),
		codeReviewFile:    md("# Brief: adversarial code review\n\n" + codeReviewBrief()),
	}
}

// factoryWorkflowBody is the factory workflow. ref names where the tool
// mechanics are documented; oq and cr name where the two briefs are.
func factoryWorkflowBody(ref, oq, cr string) string {
	r := strings.NewReplacer("@REF@", ref, "@OQ@", oq, "@CR@", cr)
	return md(`Work is delivered **one milestone at a time, autonomously**. The human approves a milestone's
backlog when it starts and tophats the milestone when it is done. Between those two points you
do not wait for the human: open questions go to a sub-agent, finished work goes to an
adversarial review sub-agent, and approved work is merged. The human course-corrects at the
milestone boundary.

### Requirements

This workflow needs an agent that can start a sub-agent with a fresh context: one that does not
see your conversation or your reasoning. If you cannot do that, stop and tell the user that this
workflow needs sub-agents.

`) + factoryRules() + publicByDefault() + md(r.Replace(`### Planning model

Everything about the build is tracked in xpo; there is no separate roadmap file.

| What | Where in xpo |
| --- | --- |
| A delivery target | An xpo milestone (¦milestone¦ tool) with a label and a one-line goal that includes its **Done when**. A target date is optional. xpo's milestone order is the build order. |
| A unit of work | A story: a ¦feature¦, ¦bug¦ or ¦task¦ issue in the milestone, with story points |
| A large unit of work | An ¦epic¦ in the milestone, with child issues (¦parent¦) in the same milestone |
| An open product question | A ¦task¦ issue with the label ¦question¦, in the milestone that needs the answer |
| A cut to pay back later, or an idea not on the path | A BACKLOG issue in no milestone |

Milestones are for target planning. Epics and stories decompose the work into manageable
chunks:

- Every story has story points. A story is 5 story points or fewer.
- A story over 5 story points becomes an ¦epic¦: change its label, and split it into child issues
  of 5 story points or fewer, in the same milestone.
- Do not start an epic. Work only on its children. When its last child is DONE, set the epic DONE.
- A cut to pay back later becomes a BACKLOG issue in no milestone, linked to the issue it came
  from.
- A trap that later issues must know about becomes a comment on each issue it affects.

See **Issue Creation**, **Issue Labels** and **Milestones** in @REF@.

### When to stop

Stop and wait for the human only:

1. **To approve the milestone backlog**, before a milestone starts (step 2).
2. **At milestone completion**, for the tophat (step 10).
3. **When nothing unblocked is left** in the milestone, because every remaining story is BLOCKED.

Everything else is handled without stopping:

| Situation | What to do |
| --- | --- |
| Open question in a spec | Answer it with a sub-agent (step 3) |
| Decision the spec does not cover | Choose, add it to the spec's **Agent Decisions**, continue |
| Spec is wrong or has a gap | Update the spec first, record it in **Agent Decisions**, then change code |
| A new architectural decision | Write it into the project's decision records as proposed, follow it, flag it for the tophat. If the project keeps no decision records, write it into the spec's **Decisions** and flag it for the tophat |
| The work seems to need reopening an accepted decision record | Do not. Pick the option that conforms to it, record the tension in **Agent Decisions** |
| Needs the human: credentials, accounts, paid services, anything outward-facing or destructive | Set the issue BLOCKED with a comment saying exactly what is needed, move to the next issue |
| Bug or follow-up work found | Follow Rule 3 |
| Review keeps failing after 3 rounds | Set the issue BLOCKED with the open findings, move on |

### 1. Orient

- Call ¦milestone¦ ¦list¦ for the Current milestone, then ¦list¦ its issues.
- If an issue is DOING and assigned to the person you work for, resume it in its worktree,
  starting from its latest handoff comment.
- If every story in the milestone is BACKLOG, or it has no issues, the milestone has not started:
  plan it (step 2).
- If the milestone has started and no story in it is PLANNED or DOING, it is not completed, and:
  - some stories are BLOCKED: stop and say what each one needs;
  - otherwise it awaits its tophat: stop and give the tophat report again (step 10).

### 2. Plan the milestone

When the Current milestone has no issues, split its goal into stories in BACKLOG in the
milestone. When it already has issues, review them against the goal and add what is missing.

- Before each ¦add¦, search ¦list¦ with ¦match¦ and ¦rationale¦ for existing work; reuse it.
- Estimate every story in story points. Split any story over 5 story points (see **Planning model**).
- Link dependencies (¦blocked_by¦) so the order is explicit.
- A ¦question¦ issue is answered before the work that needs it: the open-questions sub-agent
  (step 3) answers it. Record the answer in the project's decision records as proposed (or in
  the spec of the work that needs it, if the project keeps none), link it in a comment, and
  close the question.

**Issue descriptions are brief.** Two to five sentences: the outcome, why it matters for the
milestone, the scope boundary (what is explicitly not in it), and pointers to decision records
or documents. No acceptance criteria, no design, no file lists: those belong in the spec, which
is written when the issue is started and knows more. A spec never repeats the description; it
starts where the description ends.

Then give the human the backlog (one line per issue, with its ID and story points) and stop
for approval. When the human approves the backlog and starts the milestone, set every BACKLOG
issue in it to PLANNED.

After the milestone has started, file in-scope work you find as PLANNED in the milestone: the
human's approval covers it. Out-of-scope work goes to BACKLOG in no milestone.

### 3. Spec

Write a spec with the ¦spec¦ tool before implementing (see **Specs** in @REF@). Check ¦rationale¦
for prior specs and walkthroughs in the same area, and read the decision records the work
touches.

**Open questions go to a sub-agent.** When the drafted spec has any Open Questions, start a fresh
sub-agent, not a fork of your context, with @OQ@. Do not answer them yourself first and do not
hand it your preferred answers; it should reach its own. Then fold its answers into the spec:

- A confident answer grounded in the code or a decision record becomes a **Decision**.
- A low-confidence answer, or one it marks as the human's call, becomes an **Agent Decision**
  (the question, the choice, why, what to revisit) and is flagged for the tophat.

Write the final spec with no Open Questions left, then continue. Do not wait for the human.

### 4. Start

- Only PLANNED stories in the Current milestone are eligible. Take them in dependency order,
  then by **Prioritization**.
- If a ¦blocked_by¦ target is not DONE, pick another issue. Never start two issues in one
  dependency chain at once.
- If the issue is DOING and assigned to someone else, skip it and note it for the tophat.

Call ¦start¦ with the issue ID **before touching any file**, and work only in the worktree it
returns (see **Starting Work** in @REF@).

### 5. Implement & test

Follow the spec: its flow, decisions and edge cases are your requirements. Every acceptance
criterion needs a test that would fail if the behavior broke. Prefer extending an existing
test or shared suite over adding a near-duplicate.

Decisions the spec does not cover, spec gaps, and out-of-scope findings are handled by the
table in **When to stop**. Never leave TODOs: file an issue instead.

### 6. Adversarial review

When the project's test and lint commands pass in the worktree, start a fresh sub-agent, not a
fork of your context, with @CR@. Give it the issue ID, the worktree path, the base commit, the
milestone goal and the final spec text. Do not give it your own assessment of the change.

- **Blocker:** fix it. Only a **blocker** starts another round.
- **Major:** fix it or file it as a follow-up issue. It does not start another round.
- **Minor:** fix it if cheap; otherwise file it or drop it.
- You may reject a finding when it is wrong, when the spec says otherwise, or when its trigger
  is not realistic in normal use of the product. "Not realistic" is not valid in the strict
  areas the brief names (security and data loss). Write each rejection and its reason in the
  completion comment, so the human can check your calls at the tophat.
- After fixing a blocker, run the project's test and lint commands again. Then start a new
  reviewer with the previous findings and the fix diff. It checks only the fixes and the code
  they touched. Do not ask it to review the whole change again.
- Stop after 3 rounds that still have a blocker: set the issue BLOCKED with the open findings.

If a finding shows the spec itself was wrong, update the spec first (step 7), then the code.

### 7. Spec stays true

The spec must match the final implementation. Any change of approach, acceptance criteria or
decisions, from review or from the human's tophat feedback, goes into the spec **before** the
code changes. Future agents read the spec, not the diff.

### 8. Walkthrough, commit, merge

Once the review passes:

1. Add a brief completion comment with ¦comment¦: what changed, the review rounds and their
   outcome, and any rejected findings with reasons.
2. Write the walkthrough with ¦walkthrough¦ (see **Walkthroughs** in @REF@).
3. If the project's instructions require a changelog entry, add it now.
4. In the worktree: run ¦git status¦, remove scratch files, then
   ¦git add -A && git commit -m "<issue-id>: <issue title>"¦.
5. Call ¦merge¦ (see **Merging** in @REF@, which also covers conflicts).
6. If the issue is the last open child of an epic, set the epic DONE.

Never run ¦git push¦, and never commit, merge or rebase on the default branch yourself.

### 9. Next issue, or hand off

Return to step 4 with the next eligible issue. When none is left, go to step 10.

A session can end at any point, and the next one resumes from the board alone. If you stop
with an issue still DOING, first add a handoff comment to it: what is done, what is left, the
exact next step, and any traps. Commit work in progress on the issue's branch, in its worktree,
so nothing lives only in an uncommitted file.

### 10. Milestone completion: stop for the tophat

The milestone is complete when every story and epic in it is DONE (or deliberately moved out,
with a reason), and its **Done when** is checked by a test that the project's test commands run.

1. On the default branch, run the project's test and lint commands. Both must pass. If they do
   not, file a bug in the milestone and fix it through the loop.
2. Do **not** ¦complete¦ the xpo milestone yet. That happens after the human approves.
3. Give the human the tophat report and stop:
   - **Delivered:** one line per issue, with its ID.
   - **How to tophat:** the exact commands, screens and flows to try, and what they should see.
   - **Agent Decisions:** every one made across the milestone's specs, grouped by issue, each
     with what to revisit if it was wrong. This is where the human is most likely to steer.
   - **Proposed decision records:** any written during the milestone, for acceptance.
   - **Deferred:** BLOCKED issues and what they need, follow-ups filed, rejected review
     findings, with reasons, and cuts that must be paid back later.

### After the tophat

- **Feedback:** each correction becomes a spec update and a fix on its issue. If the original
  issue is DONE, file a new PLANNED issue in the same milestone. Deliver them with the same loop
  (steps 3–9), then give a short tophat report covering only what changed, and stop again.
- **Approval:** accept or rework the proposed decision records as the human said, then
  ¦complete¦ the xpo milestone. The next milestone starts at step 2, with its own backlog
  approval.

### Backlog Review

When the user asks to review the board (not work an issue), this is a **read-only** operation:

1. Check the Current milestone: its goal, and how many stories are DONE, DOING, PLANNED and BLOCKED
2. Check BLOCKED issues — what's stuck and what it's waiting on
3. Check the Next milestone — is its backlog ready for approval
4. Scan BACKLOG issues in no milestone — anything notable worth planning

Flag stale DOING issues: an issue with no recent event in its ¦history¦. Do NOT change any issue status during a review.

### Prioritization

When choosing what to work on next (and dependency links do not resolve the order):

1. **Blockers** — issues blocking other work
2. **Bugs** — correctness problems in existing functionality
3. **Planned stories** — by dependency order, then by story points (smaller first)
`))
}

// factoryRules is the factory workflow's Rules section. The human's approval
// of a milestone backlog replaces the guided per-issue approval.
func factoryRules() string {
	return `### Rules

1. Every code change must be backed by an xpo issue transitioned to DOING before any file is modified.
2. A milestone starts only after the human has approved its backlog. That approval covers planning and starting the issues in it. Never start a BACKLOG issue outside the started milestone without the human's approval.
3. Bugs found during implementation need no approval. Fix a bug that is in the scope of the current issue as part of it. File anything out of scope as a new issue and link it back to the current one.

`
}

// openQuestionsBrief is the brief for the sub-agent that answers a spec's
// open questions, without a heading.
func openQuestionsBrief() string {
	return `Start a fresh sub-agent, not a fork of your context, with the prompt below. Fill in the
bracketed parts. Paste the spec text and the milestone goal in full; the sub-agent should not
need the xpo tools. For a ¦question¦ issue, paste its description in place of a spec. Do not
add your own answers.

¦¦¦text
You are answering open design questions for issue [ISSUE-ID] in the repository at
[WORKTREE OR REPO PATH]. Nobody is available to ask: your answers decide how this gets built,
and a human reviews the choices only when the whole milestone is done.

Read first: the project's instructions for agents, its README and planning documents, and
every decision record that touches the questions. Then read the code the questions are about.
Do not use git history as a design source.

The current milestone and its goal:

[MILESTONE GOAL]

The spec, as drafted:

[SPEC TEXT]

The open questions:

[NUMBERED QUESTIONS]

For each question, work out the answer from the evidence, not from what is most common in
general. An accepted decision record is settled: if a question would reopen one, answer with
the option that conforms to it and say so. Prefer the option that keeps the current
milestone's scope.

Do not edit any file. Report, for each question:

- **Answer:** the choice, stated plainly.
- **Evidence:** the files, decision records or documents it rests on (path:line where possible).
- **Alternatives rejected:** one line each, with why.
- **Confidence:** high, medium or low.
- **Human's call?** yes or no. Say yes only when the answer depends on product intent, cost,
  or external facts that the repository cannot settle, and say what the human would need to
  decide.
¦¦¦
`
}

// codeReviewBrief is the brief for the adversarial review sub-agent, without
// a heading.
func codeReviewBrief() string {
	return `Start a fresh sub-agent, not a fork of your context, with the prompt below. Fill in the
bracketed parts, pasting the spec and the milestone goal in full. Do not add your own view of
the change, its risks, or why it is correct. On round 2 and later, fill in the round 2+ part
and give the reviewer only the previous findings and the fix diff to check.

¦¦¦text
You are reviewing issue [ISSUE-ID] before it merges. The change is in the worktree at
[WORKTREE PATH], on top of base commit [BASE SHA]. See it with
git -C [WORKTREE PATH] diff [BASE SHA] plus any untracked files from git status. Nobody else
reviews it until the whole milestone is done.

Your job is an accurate review, not a long one. A missed real problem ships. A false alarm has
a cost too: the implementer changes code that did not need it, and the change gets riskier and
later. **PASS** with no findings is a normal result for a sound change. Report a finding only
when you can show it.

Read first: the project's instructions for agents, and every decision record that the change
touches. The current milestone and its goal:

[MILESTONE GOAL]

The spec the change must satisfy:

[SPEC TEXT]

[ROUND 2+ ONLY: The previous round's findings, and how the implementer says they were
addressed:
FINDINGS
The fix diff:
[FIX DIFF]
Check only these fixes and the code they touched. For each finding, verify the fix rather than
trusting the claim. Report a new problem only if it is in code that the fixes touched. Do not
review the rest of the change again.]

Check, in this order. The first three are mechanical: they need no realistic trigger.

1. **Acceptance criteria.** For each AC, find the code that meets it and the test that proves
   it. An AC with no test, or with a test that would still pass if the behavior were broken,
   is not met.
2. **Decision records and project rules.** Every rule in the project's instructions. Any
   departure from an accepted decision record is a blocker.
3. **Tests.** Are they meaningful (they assert behavior, not implementation details or
   mocks)? Search the existing tests: flag any new test that duplicates one already there, or
   that belongs in a shared suite instead of beside one implementation.
4. **Correctness.** Bugs, unhandled errors, races, the edge cases the spec lists, and the
   inputs of normal use of the product.
5. **Scope.** Changes the spec does not call for, and spec decisions the code ignores.

Severity:

- **Blocker:** an unmet AC, a violated decision record or project rule, or wrong behavior with
  a realistic trigger.
- **Major:** a missing or meaningless test, or a likely bug with a realistic trigger.
- **Minor:** clarity, duplication, small cleanups, and problems that need several unlikely
  conditions together. Drop such a problem if it is not worth a line.

A realistic trigger says who does what, in normal use of the product, that causes the problem.
A blocker or major outside the mechanical checks must name one; without it, the finding is
minor at most.

**Strict areas:** authentication, authorization and permissions, secrets and credentials, and
data loss. There, an unlikely path is what an attacker looks for, and lost data does not come
back. A path that an attacker can cause, or that loses or corrupts data, can be a blocker even
if normal use never takes it. Name the path.

Run the project's test and lint commands in the worktree and report the result. Do not edit
any file.

Report:

- **Verdict:** PASS (no blockers or majors) or FAIL.
- **AC table:** each AC, met or not, with the code and the test that show it.
- **Findings,** most severe first. Each with its severity, path:line, the realistic trigger
  (or the mechanical check it fails, or the strict-area path), and the fix you would make.

Report only what you have checked. Say "none" for a category that is clean.
¦¦¦
`
}
