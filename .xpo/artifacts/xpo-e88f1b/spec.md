# Spec: Agents skip committing before merge; installed skills go stale

## What
1. Make step 9 (Complete) of the generated xpo skill say plainly that the agent commits in the worktree and then calls `merge`, with no contradicting sentence.
2. Say in the `merge` tool description that the worktree must be committed first.
3. Get template updates to reach projects that are already set up. `xpo doctor` detects an out-of-date or hand-edited skill, and `xpo init` / `xpo doctor --fix` refresh it, using the same replace/keep/diff prompt as managed instruction blocks.

## Why
- Step 9 currently says "Commit all changes… then use `merge`" and then "Do not use manual git commands to merge or commit". Agents follow the ban, call `merge` on a dirty worktree, get refused by `WorktreeRequireClean`, and have to retry. This happens in every project.
- An earlier fix to the template (a1746b7) never reached existing projects. `doctor` only reports "Update available" when the instruction block's *CLI version* is older than the binary. Template changes shipped without a version bump go unnoticed, and nothing compares the installed skill files with the current template. This repo alone has three different step-9 texts: the template, `.claude/`, and `.codex/`.
- Separately, `init` rewrites the skill files unconditionally (`WriteAgentSkill` → `os.WriteFile`), so a user's edits to `SKILL.md` are silently lost. Managed instruction blocks already avoid this with a content hash (xpo-76e2a1). Skills should get the same protection.

## How

### 1. Step 9 wording (`GenerateSkillMD`, `internal/exponential/agents.go`)
Replace the step 9 body with:

```markdown
### 9. Complete

**Gate:** do not complete until ALL of the following are true:

1. The user has explicitly approved the changes
2. Tests pass
3. The walkthrough is written
4. All changes are committed on the issue's worktree or branch

If any are missing, go back to the missing step.

Then, in the issue's worktree (or on its branch):

1. If the project's instructions require a changelog entry, add it now.
2. Run `git status`. Delete or move scratch files that are not part of the change.
3. Commit everything: `git add -A && git commit -m "<issue-id>: <issue title>"`.
4. Call the xpo MCP server's `merge` tool. It squash-merges the branch into the default
   branch, records a MERGE event, closes the issue, and removes the worktree or branch.
   It refuses to run while the worktree has uncommitted or untracked files.

Committing on the issue's worktree or branch is your job. Merging is `merge`'s job: never run
`git merge`, `git rebase`, `git push`, or commit on the default branch yourself.
```

Step 6's line "Do not write the walkthrough, commit, or merge until the user has explicitly approved" stays unchanged. It controls when to commit, and is consistent with the new wording.

### 2. `merge` tool description (`internal/mcpserver/tools.go`)
> Merge an issue's branch into the default branch (squash by default), record a MERGE event, and close the issue. All changes on the issue's worktree or branch must be committed first — merge refuses to run while the worktree has uncommitted or untracked files. When worktrees are enabled, the merge runs from the hub (primary checkout on main) and the worktree is cleaned up automatically.

Update `GenerateMCPToolsRef()` (`references/mcp-tools.md`) to match if it describes `merge`.

### 3. Hash marker for skill files
Skill files get a trailing marker line, reusing `ComputeBlockHash`:

```
<!-- xpo:skill <cli-version> sha256:<hash of everything above the marker> -->
```

- It goes at the **end** of each file, so `SKILL.md`'s YAML frontmatter stays first, as harnesses require.
- It applies to all three generated files: `SKILL.md`, `references/spec-guide.md` and `references/mcp-tools.md`.
- New helpers in `internal/exponential/skill_files.go`:
  - `WrapSkillFile(content, version) string`
  - `ParseSkillFile(fileContent) (body, version, hash string, hasMarker bool)`
  - `SkillFileState(path, generated) → Missing | UpToDate | Stale | Edited`:
    - **Missing:** the file does not exist.
    - **Edited:** a marker is present and its hash doesn't match the body, meaning a person changed the file.
    - **Stale:** the body differs from `generated` and the file is not Edited. This includes legacy files with no marker, following the existing rule that a missing hash means "not edited".
    - **UpToDate:** the body equals `generated`.

  Comparison is on trimmed body content, not version, so template changes without a version bump are detected.

### 4. Writing skills (`WriteAgentSkill`)
Signature becomes `WriteAgentSkill(agent, globalBaseDir string, force bool) (SkillWriteResult, error)`.
- For each file: Missing/Stale/UpToDate → write the wrapped content. Edited and `!force` → skip, and record the file in `result.Edited` with its old and new body so a diff can be shown.
- `SkillWriteResult{Dir string; Edited []EditedSkillFile}`.

### 5. `init` / `doctor`
- **`applyInit`:** call `WriteAgentSkill(agent, "", initForce)`. If `Edited` is non-empty, `handleEditedSkill` runs. It behaves like `handleEditedBlock`: in non-interactive mode or with `--yes` it forces the write, otherwise it asks per file to "Replace with the X version / Keep your version / Show diff", reusing `showSimpleDiff`.
- **`CheckAgentHealth`:** for a locally installed skill, compute the state of each file.
  - Stale or Missing → `AutoFix: "Skill out of date"` (reported once per agent).
  - Edited → `Interactive: "<path>: skill has local edits"`.
- **`doctor --fix`:** out-of-date skills are handled by the existing `applyInit(autoFixAgents)` path. Edited skills go through the interactive loop, which gains the skill prompt alongside the existing block prompt.
- **`ComputeInitPlan`:** list a skill file as `update` only when its state is not UpToDate, so the change plan is accurate.

### 6. This repo
Regenerate `.claude/`, `.codex/` and `.opencode/` skills by running `xpo doctor --fix` with the new binary, and commit the result.

## Tests (write first)
- `GenerateSkillMD()`:
  - step 9 contains `git add -A && git commit` before the `merge` call;
  - the gate list contains the commit gate;
  - the string "Do not use manual git commands to merge or commit" is gone.
- `WrapSkillFile`/`ParseSkillFile` round-trip. Frontmatter is still the first line of the wrapped `SKILL.md`.
- `SkillFileState`: Missing, UpToDate, Stale (with marker, older body), Stale (legacy, no marker), Edited (body changed after wrap).
- `WriteAgentSkill`:
  - overwrites Stale/legacy files;
  - skips Edited files without `force` and reports them;
  - overwrites them with `force`.
- `CheckAgentHealth`: out-of-date skill → AutoFix; edited skill → Interactive; current skill → no problems.
- `merge` tool description mentions the commit requirement.

## Acceptance Criteria
- [ ] Step 9 lists the commit as a gate and as an explicit numbered step; there is no sentence banning commits.
- [ ] The `merge` tool description (and the mcp-tools reference, if it describes merge) says a clean, committed worktree is required and that the default is squash.
- [ ] Generated skill files end with an `xpo:skill` marker containing the version and hash; frontmatter stays first.
- [ ] `xpo doctor` reports "Skill out of date" when installed skill content differs from the template, even at the same CLI version.
- [ ] `xpo doctor` reports local edits to skill files as interactive, and `init`/`doctor --fix` offer replace/keep/diff instead of silently overwriting.
- [ ] Legacy skill files without a marker are treated as out of date and refreshed automatically.
- [ ] This repo's `.claude`, `.codex` and `.opencode` skill copies are regenerated.
- [ ] `make test` passes.

## Decisions
- **The marker goes at the end, as an HTML comment.** Harnesses parse the frontmatter, and an HTML comment is invisible when the markdown is rendered. Putting it after the frontmatter would also work, but the end is simpler to strip.
- **Compare content, not version.** This is what fixes "template changed, version didn't".
- **Legacy files with no marker count as out of date, not edited.** This matches `BlockWasEdited` treating a missing hash as unedited, and today's behaviour (init already overwrites them). The downside is that edits made before this change are overwritten once. That's acceptable because this has always been the behaviour.
- **Non-interactive mode or `--yes` forces the overwrite**, matching `handleEditedBlock`.
- **Global installs** (`~/.config/xpo/skills` via symlink) are out of scope. `init` never calls the global path today (`globalBaseDir` is always `""`), so only local installs are checked.

- **`init` handles out-of-date skills even when the version is current.** Case A in `init` used to return "up to date" early whenever the integration version matched. It now also refreshes skills that are out of date or edited. Otherwise, a template change without a version bump would never be applied by `init`.
- **Non-interactive replacement of edited skills prints a note** (`! <path>: local edits replaced with the X version`), so the overwrite is visible. The block path stays silent, as before.
- **`doctor --fix` never prompts twice.** Agents already handled by the auto-fix `applyInit` pass skip the skill prompt in the interactive loop. The block prompt is shown only when the block itself is edited or in legacy format.

## Resolved questions
- End-of-file HTML comment marker: approved.
- Changelog step stays generic ("if the project's instructions require a changelog entry"); xpo does not write changelog rules itself.
