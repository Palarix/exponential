# Walkthrough: Split the xpo reference skill from the workflow, and choose the workflow at init

## What was built and why

The `xpo` skill used to mix two things: how the xpo tools behave, and a fixed process the agent had to follow. A project that wanted a different process had to edit the managed skill and the managed CLAUDE.md block. Both then counted as hand-edited and stopped getting updates, including the generic parts.

The skill is now two skills:

- **`xpo`** — the **reference**. How the tools behave: statuses, labels, links, milestones, specs, walkthroughs, `start` and `merge`. xpo manages it like before (marker, hash, refresh on re-init and `doctor --fix`).
- **`xpo-workflow`** — the **workflow**. When to do what: rules, the nine steps, Backlog Review, Prioritization. The project owns it. xpo writes it only when asked: on first `xpo init`, or later with `xpo init --workflow`. Re-init and `doctor` never read, create, copy or report it.

The CLAUDE.md stub keeps only what holds for every workflow (MCP not CLI, report MCP errors, the issue prefix) and says to load `xpo-workflow` if the project has one. The board chat prompt says the same.

## How the pieces fit together

### Templates (`internal/exponential/agents.go`, `workflow_templates.go`)

- `GenerateSkillMD()` = reference frontmatter + `generateReferenceBody("##")`. The inline docs for skill-less agents (`GenerateAgentDocs`) reuse the same body with `"###"`, because they sit under `## Exponential (xpo)`. `md()` turns `¦` into backticks, so the templates are readable raw strings.
- `GenerateMCPToolsRef()` lists every tool the MCP server exposes, including `milestone`, the `git_*` tools and `code_history`. `GenerateSpecGuide()` has no workflow concepts ("when no user is present" instead of interactive/non-interactive modes).
- `WorkflowTemplates` is a list of `WorkflowTemplate{Name, Title, Description, Skill, Inline}`; today it holds `guided`. `Skill()` is the `xpo-workflow/SKILL.md`; `Inline()` is a `## Workflow` section for agents without skills. Both come from `guidedWorkflowBody(ref)`, where `ref` names where the mechanics live ("the `xpo` skill" or "the xpo section above"). xpo-82ee77 adds the software-factory template as a second entry.
- `GenerateAgentStub()` drops Hard Rules 1–4; the rules now open the guided workflow (`guidedRules()`).

### Writing the workflow (`internal/exponential/workflow_install.go`)

- `InstallWorkflow(agent, tpl)` — first init: writes the template unless the agent already has one. `tpl == nil` means "None".
- `SetupWorkflow(agents, tpl)` — `xpo init --workflow`: deletes each `xpo-workflow/` folder and writes the template; appends the inline workflow for skill-less agents (`InlineWorkflowTargets`: one per file, not when a skill-capable agent shares the file), and leaves a file that already has `## Workflow` alone.
- `WorkflowTargets()` — agents that are set up (MCP, managed block or `xpo` skill). `ExistingWorkflowDirs()` — what `--workflow` will replace, for the confirmation.
- `HadLegacyWorkflow(agent)` — the agent's `xpo/SKILL.md` still has the old `# xpo Development Workflow` heading. Checked before the skill update, to print the hint.

### CLI (`cmd/exponential/init.go`, `doctor.go`, `internal/exponential/plan.go`)

- `--workflow` takes an optional value (`NoOptDefVal = "ask"`). `--workflow guided` with a space leaves `guided` as an argument, so `runInit` picks it up from `args`. The value is validated before any prompt or write.
- **First init:** `resolveWorkflowChoice` (flag, else a select with Guided/None, else Guided without a terminal) → `ComputeInitPlan(agents, tpl)` lists the workflow files → `applyInit(prefix, agents, tpl)` writes them.
- **Re-init and `doctor --fix`** call `applyInit(…, nil)` and `ComputeInitPlan(…, nil)`: no workflow. `handleEditedSkills` checks `HadLegacyWorkflow` before the update and prints "The workflow is now a separate skill. Run `xpo init --workflow` to set it up." once per run.
- **`xpo init --workflow` on an initialized project:** `runReinit`, then `runWorkflowSetup`: pick the template, list the existing `xpo-workflow/` folders, ask "Replace them?" (default no; `--yes` skips; without a terminal and without `--yes` it exits with an error and changes nothing), then `SetupWorkflow`.

### Chat prompt (`internal/chat/prompt.md`)

Both modes load the `xpo` skill, and the `xpo-workflow` skill if the project has one. Work Mode follows the workflow and the project's instructions, including where they say to stop and wait.

## Key decisions

- **No migration.** We first built a migration that moved an edited old skill into `xpo-workflow` and fixed it up: rename, new description, cut the old reference half, rewrite links, insert Rules, move extra files, handle leftover pre-rename folders. An extended test across 25 local projects still found unclean cases for nearly every rule. The final design drops all of it: re-init updates `xpo/` like any skill and prints a hint, and the user runs `xpo init --workflow`. Previous versions stay in git.
- **xpo does not manage the workflow after writing it.** No marker, nothing in config, no copy to agents set up later, no `doctor` check. The only way xpo writes it again is `xpo init --workflow`, which asks first.
- **Instruction files change only between the markers.** Skill-less agents (Copilot, generic `AGENTS.md`) lose an old inline workflow on update; `--workflow` gives them a new one below the block.
- **The reference says how, the workflow says when.** "Update the spec first", "search `rationale` first", Backlog Review, Prioritization and **Public by default** (no local paths or private project names in issues, specs and walkthroughs) are workflow. Public by default is policy, and a private project may want another.
- **Linking describes the real gate.** Moving an issue to DOING is refused while an issue that blocks it is not DONE, whichever side stores the link (`internal/exponential/update.go`); `start` also refuses BACKLOG and BLOCKED (`start.go`). The old line "Only `blocked_by` prevents `start`" was ambiguous.
- **Step 9's gate no longer lists "all changes committed".** The steps under it commit explicitly before `merge`, which was the fix in xpo-e88f1b, and `merge` refuses uncommitted work.
- **The skill is named `xpo-workflow`**, so the prompt and the stub can name it, and it doesn't clash with a project's own `workflow` skill.

## Non-obvious things

- A few projects still have a leftover `xpo-workflow/` folder from before xpo-dd97a5 renamed the skill to `xpo`. It holds the old combined skill. xpo now treats it as the project's workflow and leaves it alone. When the project has no old `xpo/` skill, it gets no hint either. `xpo init --workflow` replaces it.
- Files a project added to `xpo/references/` stay in place; the reference update only writes the generated files.
- This repo's three workflows are the guided template plus the project's writing style (`references/writing-style.md`, linked from steps 3 and 8). That style used to be an edit to the Claude spec guide.

## Acceptance Criteria

### Reference
- [x] Reference-only skill, no "when" content. Evidence: `TestReferenceSkill_HasNoWorkflow`, `TestReference_NoWhenContent`.
- [x] Description starts with "How the xpo (Exponential) MCP tools work", no `metadata.version`, intro as specified. Evidence: `TestReference_DescriptionAndIntro`.
- [x] `##` in the skill, `###` inline; sections as listed. Evidence: `TestReference_HeadingLevels`.
- [x] Linking and Starting Work describe the gate; no "Work Mode". Evidence: `TestReference_LinkingAndStartDescribeTheGate`, `TestReference_NoWhenContent`.
- [x] Milestones section; tool list linked from Finding Work. Evidence: `TestReference_Milestones`, `TestReference_LinksToolList`.
- [x] `mcp-tools.md` lists every tool and every status. Evidence: `TestMCPToolsRef_ListsAllTools`.
- [x] Neutral spec guide, US spelling. Evidence: `TestSpecGuide_NeutralWording`, `TestUSSpelling`.

### Workflow
- [x] Guided template with Rules, Public by default, steps 1–9, Backlog Review, Prioritization. Evidence: `TestGuidedWorkflow_Skill`, `TestGuidedWorkflow_Inline`, `TestGuidedStep1SearchesBoardAndRationaleBeforeAdd`.
- [x] Public by default in the workflow only, generic example. Evidence: `TestPublicByDefault_InWorkflowNotReference`, `TestGuided_RulesAndSteps`.
- [x] BACKLOG rule once, bug split, three-item step 9 gate, stale DOING defined. Evidence: `TestGuided_RulesAndSteps`, `TestSkillMD_Step9_*`.
- [x] Stub keeps only universal rules; inline docs point to a Workflow section. Evidence: `TestAgentStub_OnlyUniversalRules`, `TestAgentDocs_ReferenceOnly`.
- [x] Template registry is a list. Evidence: `WorkflowTemplates`, `TestLookupWorkflow`.

### First init
- [x] Guided/None prompt, `--workflow guided|none`, Guided without a terminal, unknown name rejected before any write. Evidence: manual run in a temp repo (`--workflow bogus` exits 1 with no files written; `--workflow none` writes no workflow).
- [x] Workflow written without a marker, never over an existing one; inline for skill-less agents; listed in the plan. Evidence: `TestInstallWorkflow_*`, `TestAppendInlineWorkflow_Once`, `TestComputeInitPlan_ListsWorkflowOnlyForFirstInit`.

### Re-init and doctor
- [x] Never touch `xpo-workflow`, never write outside the markers. Evidence: `TestReinit_LeavesWorkflowAlone`, `TestWriteAgentInstructions_OnlyBetweenMarkers`; extended test across 25 local projects: no `xpo-workflow` file changed, no instruction file changed outside its markers (except converting the old unmarked format, which already worked this way).
- [x] Old combined skill updated like any skill; hint printed once. Evidence: `TestHadLegacyWorkflow`; manual run; extended test (18 of 25 projects got the hint, the other 7 had no old combined `xpo/` skill).

### `xpo init --workflow`
- [x] Re-init, then template for set-up agents; existing folders listed and replaced after confirmation; `--yes`; error without a terminal; inline kept with a note. Evidence: `TestSetupWorkflow_ReplacesFoldersAndAppendsInline`, `TestWorkflowTargets_SetUpAgentsOnly`; manual run (no terminal, no `--yes`: error and the existing workflow unchanged; `--yes`: folder replaced, extra file gone; `--workflow none`: error).

### Chat prompt
- [x] Both modes load `xpo` and `xpo-workflow`. Evidence: `TestSystemPromptFor`.

### This repo
- [x] Reference regenerated, guided workflow plus writing style in all three agents, `doctor` reports all agents up to date. Evidence: `xpo doctor` in the worktree.
- [x] `make test` passes.
