# Split the xpo reference skill from the workflow, and choose the workflow at init

## What

The `xpo` skill becomes a managed **reference** (how xpo works). The **workflow** (how this project works) moves to a separate `xpo-workflow` skill that the project owns. xpo writes it only when asked: on first `xpo init`, or with `xpo init --workflow` later. Re-init and doctor never read, create, copy or report it. The CLAUDE.md stub keeps only the rules that hold for every workflow. The board chat prompt loads the workflow skill if there is one.

## Why

Today a project that wants a different process (for example, an autonomous one) has to fork the whole `xpo` skill and edit the managed CLAUDE.md block. Both then count as hand-edited, so the project stops getting updates, including for the generic reference parts. After the split, the reference stays managed and up to date, and the workflow can be changed freely.

## Acceptance Criteria

### Reference (`xpo` skill, managed)

- [ ] `GenerateSkillMD()` produces a reference-only skill: how the tools behave, never when to use them. No numbered workflow steps, no "stop and wait", no "update the spec first", no "search `rationale` first".
- [ ] Description starts with "How the xpo (Exponential) MCP tools work". No `metadata.version`.
- [ ] Intro: "How the xpo tools behave: statuses, labels, links, milestones, specs, walkthroughs, start and merge. When to use them is the project's call: if the project has an `xpo-workflow` skill, load it and follow it."
- [ ] Sections use `##` in the skill and `###` in the inline docs: Status Graph, Finding Work (links `references/mcp-tools.md`), Issue Creation, Issue Labels, Linking, Milestones, Specs, Walkthroughs, Starting Work, Merging.
- [ ] Linking describes the actual gate (moving to DOING is refused while a blocking issue is not DONE, whichever side stores the link; a workflow may add gates). Starting Work says `start` refuses BACKLOG and BLOCKED. No "Work Mode".
- [ ] `references/mcp-tools.md` lists every tool, including `milestone`, `git_*` and `code_history`; the `update` row lists all statuses.
- [ ] `references/spec-guide.md` has no workflow concepts. US spelling throughout.

### Workflow (`xpo-workflow` skill, project-owned)

- [ ] A guided template: **Rules** (the stub's hard rules), **Public by default**, steps 1–9, Backlog Review, Prioritization. It points to the reference for mechanics.
- [ ] **Public by default**: no local file paths, names of other private repositories or projects, internal hostnames or URLs, or credentials; example "another project". Not in the reference or spec guide.
- [ ] The BACKLOG approval is stated once (Rules). Bugs: in-scope fixed as part of the issue, out-of-scope filed and linked back. Step 9 gate has three items; the explicit commit step stays. Stale DOING = no recent event in `history`.
- [ ] The stub keeps the MCP intro, the prefix, "never shell out to the CLI", "report MCP errors", and the `xpo-workflow` line. The inline docs say "If this file has a Workflow section, follow it".
- [ ] The template registry is a list, so xpo-82ee77 adds **Software factory** with one entry.

### First init

- [ ] The user picks **Guided** (default) or **None**. `--workflow guided|none` (or `--workflow=guided`) sets it without a prompt; non-interactive / `--yes` without the flag uses Guided. An unknown name is an error before any write.
- [ ] Guided writes `<SkillDir>/xpo-workflow/SKILL.md` (no `xpo:skill` marker) for each selected skill-capable agent, unless one exists, and appends `## Workflow` below the managed block of skill-less agents that share no file with a skill-capable one. The change plan lists these files.

### Re-init and doctor

- [ ] Manage `xpo/` and the managed block only. They never read, create, copy or report `xpo-workflow`, and never write outside the markers.
- [ ] An old `xpo/SKILL.md` that still holds the workflow is updated like any stale or edited skill (quiet if unedited; prompt, or replace with a note without a terminal, if edited).
- [ ] If an agent's `xpo/SKILL.md` had the old `# xpo Development Workflow` heading before the update, re-init prints once: "The workflow is now a separate skill. Run `xpo init --workflow` to set it up."

### `xpo init --workflow [name]` on an initialized project

- [ ] Runs the normal re-init, then sets up the workflow.
- [ ] Without a value it asks which template (non-interactive: the first); with a value it uses that template. `none` and unknown names are errors.
- [ ] Targets: every agent that is set up (MCP, managed block or `xpo` skill present).
- [ ] If any target has an `xpo-workflow/` folder, it lists them and asks "These will be replaced. Continue?" (default no). `--yes` skips the question; without a terminal and without `--yes` it stops with an error and changes nothing.
- [ ] Each target's `xpo-workflow/` folder is deleted and the template is written. Skill-less targets get `## Workflow` appended below the block if the file has none; if it has one, it is left alone with a note.

### Chat prompt

- [ ] `internal/chat/prompt.md`: both modes load `xpo`, and `xpo-workflow` if the project has one.

### This repo

- [ ] All three agents have the regenerated reference and the guided workflow plus the project's writing style (`references/writing-style.md`, linked from steps 3 and 8). `doctor` reports all agents up to date.
- [ ] `make test` passes. Tests cover each item above.

## Flow

1. **`agents.go`**: reference body (`##`/`###`), Milestones, reworded sections; full `mcp-tools.md`; neutral spec guide. `WriteAgentInstructions` only touches the managed block (plus the existing conversion of the old unmarked format).
2. **`workflow_templates.go`**: guided template and registry.
3. **`workflow_install.go`**: `InstallWorkflow(agent, tpl)` (first init, only if missing); `SetupWorkflow(agents, tpl)` (delete and write, inline append); `ExistingWorkflowDirs(agents)`; `WorkflowTargets()` (set-up agents); `HadLegacyWorkflow(agent)`; `InlineWorkflowTargets`, `AppendInlineWorkflow`.
4. **`init.go`**: `--workflow` with an optional value; first-init prompt; `runWorkflowSetup` for initialized projects with the confirmation; the re-init hint.
5. **`plan.go`**: workflow files only for first init.
6. **`doctor.go`**: no workflow handling.
7. **`prompt.md`** + test. 8. This repo. 9. CHANGELOG.

## Decisions

- **The skill is named `xpo-workflow`.**
- **xpo does not manage the workflow after writing it.** No marker, nothing in config, no migration, no copy to agents added later. The user asks for it with `xpo init --workflow`, which overwrites after confirmation.
- **No migration.** An extended test across 25 local projects showed every per-case migration rule (moving edited files, rewriting names, descriptions and links, cutting the old reference half, inserting Rules, leftover pre-rename folders, Hard Rules in old instructions) still left cases unclean. Upgraded projects get a one-line hint instead, and their previous skill stays in git.
- **Instruction files: only between the markers.** Skill-less agents lose an old inline workflow on update; `--workflow` gives them a new one.
- **The reference says how, the workflow says when.** Public by default is workflow (policy).
- **Step 9 gate drops "all changes committed"**; the explicit commit step stays (xpo-e88f1b).
- **Only Guided and None at first init** (Software factory: xpo-82ee77; `--workflow <path>`: xpo-c35460).

## Edge Cases

- **MEDIUM — `--workflow` without a terminal and without `--yes`, with existing workflows**: error, nothing changed.
- **MEDIUM — a skill-less agent's file already has `## Workflow`**: left alone, note.
- **LOW — `xpo init --workflow` before `xpo init`**: it is the first init with that choice.
- **LOW — an agent set up later**: no workflow until `xpo init --workflow`.

## Assumptions

- Global skill install (xpo-ceb434) is not wired into the CLI. `xpo drive` does not name the skill.
