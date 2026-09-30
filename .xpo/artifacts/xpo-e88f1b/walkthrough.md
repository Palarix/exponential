# Walkthrough: Agents skip committing before merge; installed skills go stale

## The problem
Agents in every xpo project reached step 9, called `merge` on a dirty worktree, and were refused by `WorktreeRequireClean`. There were two causes:

1. **The instructions contradicted themselves.** Step 9 said "Commit all changes… then use `merge`" and, in the same paragraph, "Do not use manual git commands to merge or commit". Agents obeyed the ban.
2. **Template fixes never reached existing projects.** An earlier attempt (a1746b7) changed the template, but `doctor` only flagged "Update available" when the managed block's *CLI version* was older than the binary. Nothing compared installed skill files with the template, so a change shipped without a version bump went unnoticed. This repo had three different step 9 texts. Separately, `init` rewrote skill files unconditionally, silently discarding hand edits.

## What changed

### 1. The instruction itself (`internal/exponential/agents.go`)
- The step 9 gate list gained **"4. All changes are committed on the issue's worktree or branch"**.
- The body is now numbered sub-steps:
  1. add a changelog entry *if the project's instructions require one* (deliberately generic, since xpo doesn't manage changelogs);
  2. run `git status` and remove scratch files;
  3. `git add -A && git commit -m "<issue-id>: <issue title>"`;
  4. call `merge`, which is described as squash-merging and refusing dirty worktrees.
- The ban is reworded so there is a clear split of responsibility. The agent commits on the issue's branch; `merge` merges; the agent never runs `git merge`, `git rebase` or `git push`, or commits on the default branch.
- The `merge` row in `GenerateMCPToolsRef()` and the MCP tool description (`internal/mcpserver/tools.go`) both say a committed worktree is required. An agent that reads the tool list now learns this before failing.

### 2. Out-of-date detection (`internal/exponential/skill_files.go`, new)
Each generated skill file (`SKILL.md`, `references/spec-guide.md`, `references/mcp-tools.md`) now ends with:

```
<!-- xpo:skill 1.2.1 sha256:7dd87a72d002 -->
```

- **Why at the end:** harnesses require `SKILL.md`'s YAML frontmatter to come first, and an HTML comment is invisible when rendered.
- **Hash:** reuses `ComputeBlockHash` (the managed-block hash), taken over the body above the marker.

`CheckSkillFile(path, generated)` classifies a file. The order of the checks matters:
1. file missing → **Missing**
2. trimmed body equals the template → **UpToDate**. This covers a legacy file with no marker that happens to match, and an edit that happens to match the new template.
3. marker present and its hash doesn't match the body → **Edited** (a person changed it)
4. otherwise → **Stale**. This includes legacy files without a marker, following the managed-block rule that a missing hash means "not edited".

The key design choice is **comparing content, not version**. That is what fixes "template changed, version didn't".

### 3. Writing (`WriteAgentSkill`)
The signature is now `WriteAgentSkill(agent, globalBaseDir, force) (SkillWriteResult, error)`. It writes every file except Edited ones (unless `force`). Skipped files are returned as `EditedSkillFile{Path, OldContent, NewContent}` with marker-free bodies, ready for `showSimpleDiff`. `ReplaceSkillFile` resolves a single file once the user picks "replace".

### 4. Health, plan, init and doctor
- **`CheckAgentHealth`:**
  - Stale or missing skill files → one `AutoFix: "Skill out of date"` entry.
  - Each edited file → `Interactive: "<path>: skill has local edits"`.
  - New fields `BlockEdited`, `SkillStale` and `EditedSkills` let callers tell *what* needs a prompt; the text lists were display-only.
- **`ComputeInitPlan`:** lists only skill files that aren't up to date, so the `doctor --fix` change plan is accurate.
- **`cmd/exponential/init.go`:**
  - `applyInit` goes through `handleEditedSkills`. Edited files are prompted for with replace/keep/diff (`promptEditedSkill`), mirroring `promptEditedBlock`. In non-interactive mode or with `--yes` they are replaced, and a note is printed so the overwrite is visible.
  - The "Case A" early return (integration version == CLI version) used to say "up to date" and stop. It now also refreshes out-of-date or edited skills. Without this, `xpo init` would never apply a template change made without a version bump, which was the original problem.
  - `blockNeedsPrompt(h)` is true when there are interactive items beyond the edited skills (an edited or legacy block). It stops a skill-only edit from triggering the block prompt.
- **`cmd/exponential/doctor.go`:** the interactive loop prompts for the block only when `blockNeedsPrompt`, and for skills only if `applyInit` hasn't already handled that agent (`skillsHandled`). This avoids a double prompt when an agent has one out-of-date file and one edited file.

### 5. This repo
The `.claude`, `.codex` and `.opencode` skill copies were regenerated with the new binary. The only differences are the new step 9 text, the tools-reference row and the markers.

## Tests
- `internal/exponential/skill_files_test.go`:
  - step 9 commits before `merge`, has no commit ban, has the commit gate and the generic changelog line;
  - the mcp-tools merge row mentions the commit requirement;
  - marker round-trip, with frontmatter still first;
  - every `CheckSkillFile` state;
  - `WriteAgentSkill`: writes markers, overwrites stale and legacy files, skips edited ones without force but still writes the other files, and overwrites with force;
  - health: stale at the same version → AutoFix, edited → Interactive, fresh setup → no edits;
  - the plan skips up-to-date skills.
- `internal/mcpserver/integration_test.go`: the `merge` description mentions "must be committed" and "squash".
- Existing `WriteAgentSkill` call sites in tests were updated to the new signature.

## Acceptance Criteria
- [x] Step 9 lists the commit as a gate and as an explicit numbered step; there is no sentence banning commits. Evidence: `TestSkillMD_Step9_*`.
- [x] The `merge` tool description and the mcp-tools reference say a clean, committed worktree is required and that the default is squash. Evidence: `TestMergeToolDescriptionRequiresCommit`, `TestMCPToolsRef_MergeRequiresCommit`.
- [x] Generated skill files end with an `xpo:skill` marker containing the version and hash, and frontmatter stays first. Evidence: `TestWrapSkillFile_RoundTrip`, `TestWriteAgentSkill_WritesMarker`.
- [x] `xpo doctor` reports "Skill out of date" when skill content differs from the template at the same CLI version. Evidence: `TestHealth_StaleSkill_SameVersion_IsAutoFix`, plus a manual end-to-end run in a scratch repo.
- [x] Local edits are reported as interactive, and `init`/`doctor --fix` offer replace/keep/diff instead of overwriting. Evidence: `TestHealth_EditedSkill_IsInteractive`, `TestWriteAgentSkill_EditedSkippedWithoutForce`, plus a manual run (non-interactive mode replaces the file with a note; the interactive prompt reuses the block prompt's flow).
- [x] Legacy skill files without a marker are treated as out of date and refreshed automatically. Evidence: `TestCheckSkillFile/legacy_without_marker`, plus a manual `doctor --fix` run.
- [x] This repo's `.claude`, `.codex` and `.opencode` skill copies are regenerated. Evidence: `xpo doctor` reports all three "Up to date".
- [x] `make test` passes (exit 0).

## Worth knowing
- **Global skill installs** (`~/.config/xpo/skills` via symlink) are still not checked for staleness. `init` never uses that path today.
- **Legacy files are overwritten once.** Hand edits made to a legacy (marker-less) file are overwritten on the first refresh, because without a hash there's no way to detect them. This matches previous behaviour, where `init` always overwrote.
- **In non-interactive mode, edited skills are replaced**, matching `handleEditedBlock`.
