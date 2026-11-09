package exponential

import "strings"

// WorkflowSkillName is the skill that holds a project's workflow. xpo writes
// it once at init and never compares or updates it afterwards: the project
// owns it.
const WorkflowSkillName = "xpo-workflow"

// WorkflowTemplate is a workflow `xpo init` can write.
type WorkflowTemplate struct {
	Name        string // value for --workflow
	Title       string // label in the init prompt
	Description string // one line in the init prompt
	// Skill is the SKILL.md for agents with skill support.
	Skill func() string
	// Inline is a "## Workflow" section appended below the managed block of
	// agents without skill support.
	Inline func() string
}

// WorkflowTemplates lists the workflows `xpo init` offers, in prompt order.
// The first is the default.
var WorkflowTemplates = []WorkflowTemplate{
	{
		Name:        "guided",
		Title:       "Guided",
		Description: "The agent specs, implements, and stops for your review before it merges",
		Skill:       guidedWorkflowSkill,
		Inline:      guidedWorkflowInline,
	},
}

// LookupWorkflow finds a workflow template by name (case-insensitive).
// "none" is not a template: callers handle it.
func LookupWorkflow(name string) (WorkflowTemplate, bool) {
	for _, t := range WorkflowTemplates {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return WorkflowTemplate{}, false
}

// WorkflowNames returns the accepted --workflow values, including "none".
func WorkflowNames() []string {
	names := make([]string, 0, len(WorkflowTemplates)+1)
	for _, t := range WorkflowTemplates {
		names = append(names, t.Name)
	}
	return append(names, "none")
}

func guidedWorkflowSkill() string {
	return md(`---
name: xpo-workflow
description: >-
  This project's workflow for xpo issues. Use before planning,
  implementing, fixing, or completing any code change; when filing
  issues or writing specs or walkthroughs; and when the user asks to
  review the board.
---

# Workflow

This is how this project works with xpo issues. The project owns this file: ¦xpo init¦ wrote it
once and xpo never changes it. For how the xpo tools work (statuses, labels, links, specs,
walkthroughs, starting and merging), load the ¦xpo¦ skill.

`) + guidedWorkflowBody("the ¦xpo¦ skill")
}

func guidedWorkflowInline() string {
	return md(`## Workflow

This is how this project works with xpo issues. The project owns this section: ¦xpo init¦ wrote
it once and xpo never changes it. For how the xpo tools work, see the xpo section above.

`) + guidedWorkflowBody("the xpo section above")
}

// guidedWorkflowBody is the guided workflow. ref names where the tool
// mechanics are documented.
func guidedWorkflowBody(ref string) string {
	return guidedRules() + md(strings.ReplaceAll(`### Public by default

Issues, comments, specs and walkthroughs are stored in the repository, and the repository
may be public. Write them for a reader who has only this repository:

- No local file paths (¦/Users/…¦, ¦~/…¦, ¦../other-repo¦). Use paths relative to the repository root.
- No names of other private repositories or projects, and no internal hostnames or URLs.
- No credentials, tokens or personal data.

Describe the situation in general terms instead, for example "another project".

### 1. Check existing work

Understand what's on the board before doing anything. Call ¦list¦ to see current issues; use the
¦match¦ parameter to search for related work. Call ¦show¦ for full details on candidate issues.

Before any ¦add¦, search twice: ¦list¦ with ¦match¦ for issues that already cover the work, and
¦rationale¦ for specs and walkthroughs of prior work in the same area. When you propose the issue,
tell the user what the searches found, or that they found nothing.

### 2. Plan (Write great issues)

Decide what the issue title and description should be before filing it.

- **Features/design work:** propose the idea to the user. Discuss and iterate on the proposal. Only create an issue after the user approves.
- **Bugs reported by the user:** investigate and understand the bug before filing. If possible, reproduce it or identify the root cause, then file the issue.
- **Bugs found during implementation:** follow Rule 3.

See **Issue Creation** in @REF@ for labels, titles, descriptions, and parent linking.

### 3. Spec (Think Before You Build)

Before implementing, ensure the issue has an up-to-date spec. If none exists, write one (see
**Specs** in @REF@). If the issue modifies or extends behavior introduced by prior work, search
¦rationale¦ for related specs and walkthroughs first. Skip this for greenfield features or bug
fixes in code with no deliberate design history.

The spec is a thinking tool — it collapses the design space so the user can steer decisions
instead of discovering unwanted choices after the code is written. Scale the depth to the task:
a bug fix gets a brief What/Why/How/AC spec; a feature gets the full template.

**Interactive mode** (user present — CLI, IDE plugin, board chat): after writing the spec, surface
open questions and uncertainties to the user and wait for answers before implementing. The spec is
a conversation artifact — the user should confirm or redirect before code is written.

**Non-interactive mode** (headless — xpo drive, piped prompts): implement from the spec without
waiting, BUT the spec must explicitly mark where you made judgment calls. For every open question,
document the question, the answer you chose, and your reasoning in an **Agent Decisions** section:
the question, the choice made, the reasoning, and what to revisit if the choice was wrong.

**Do not proceed to step 4 until the spec exists and (in interactive mode) the user has confirmed it.**

### 4. Start

**Before starting, check these gates:**

- Only issues with PLANNED status are eligible (see Rule 2).
- Check the issue's dependencies. If any ¦depends_on¦ or ¦blocked_by¦ targets are not DONE, stop and ask the user how to proceed. Do not start multiple issues in a dependency chain simultaneously.
- If the issue is already in DOING and assigned to someone else, stop and ask the user before taking it over (¦force: true¦ only after the user approves).

Call ¦start¦ with the issue ID **before touching any file**, and work only in the worktree it
returns (see **Starting Work** in @REF@).

### 5. Implement & Test

Follow the spec — its flow steps, decisions, and edge cases are your requirements.

- If you encounter a decision not covered by the spec, make a reasonable choice and note it when you write the completion comment in step 6. If the decision is significant (would surprise the user or constrain future work), stop and ask.
- If you realize the spec has a significant gap or is wrong, stop. Explain the issue, propose a spec update, and wait for approval.
- If you discover bugs or needed work outside the scope of this issue, follow Rule 3: file them and link them back. Do not fix them silently or leave TODOs.

### 6. Handoff

Verify your work before handing off:

- Run the project's build and test commands. All tests must pass.
- Review your own changes against the spec — confirm all acceptance criteria are met.

Then add a brief completion comment using the ¦comment¦ tool, summarizing what changed and any decisions not in the original spec. Keep it lightweight — the walkthrough (step 8) is where the full explanation lives.

**Stop here and wait.** The user needs to review and test your changes before anything else happens. Do not write the walkthrough, commit, or merge until the user has explicitly approved.

### 7. Revisions

When the user requests changes, update the spec to reflect the corrections **before** modifying
code. The spec must always match the final implementation. If the user's feedback changes the
approach, acceptance criteria, or decisions, those changes belong in the spec — not just in the
code diff. Without this step, the spec drifts from reality and future agents reading it will build
the wrong thing.

After applying the changes, go back to step 6.

### 8. Walkthrough

**Prerequisite:** the user has explicitly approved the changes. If the user has not responded since
your completion comment, you are NOT on this step — go back to step 6 (Handoff) and wait.

Write the walkthrough (see **Walkthroughs** in @REF@) **after** any user-requested corrections
are applied, so it reflects the final state.

### 9. Complete

**Gate:** do not complete until ALL of the following are true:

1. The user has explicitly approved the changes
2. Tests pass
3. The walkthrough is written

If any are missing, go back to the missing step.

Then, in the issue's worktree:

1. If the project's instructions require a changelog entry, add it now.
2. Run ¦git status¦. Delete or move scratch files that are not part of the change.
3. Commit everything: ¦git add -A && git commit -m "<issue-id>: <issue title>"¦.
4. Call the ¦merge¦ tool (see **Merging** in @REF@, which also covers conflicts).

### Backlog Review

When the user asks to review the board (not work an issue), this is a **read-only** operation:

1. Check DOING issues — what's actively underway and by whom
2. Check BLOCKED issues — what's stuck and what it's waiting on
3. Check PLANNED issues — what's ready to pick up, in priority order
4. Scan BACKLOG — anything notable worth planning

Flag stale DOING issues: an issue with no recent event in its ¦history¦. Do NOT change any issue status during a review.

### Prioritization

When choosing what to work on next (and dependency links do not resolve the order):

1. **Blockers** — issues blocking other work
2. **Bugs** — correctness problems in existing functionality
3. **Planned features** — by dependency order, then by story points (smaller first)
`, "@REF@", ref))
}

// guidedRules is the guided workflow's Rules section: the hard rules that
// lived in the agent stub before xpo-f19bfa.
func guidedRules() string {
	return `### Rules

1. Every code change must be backed by an xpo issue transitioned to DOING before any file is modified.
2. Never start work on a BACKLOG issue without explicit user approval to transition it.
3. Bugs found during implementation need no approval. Fix a bug that is in the scope of the current issue as part of it. File anything out of scope as a new issue and link it back to the current one.

`
}
