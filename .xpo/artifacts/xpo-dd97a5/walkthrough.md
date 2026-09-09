# Walkthrough: Upgrade to v3 skill

## What was built

Updated the generated skill content to match the v3.1 reference skill, plus CLI polish improvements that surfaced during review.

## Skill content (v1.0 → v3.1)

**`GenerateSkillMD()`** — new frontmatter (`name: xpo`, `version: "3.1"`) and intro line.

**`generateWorkflowBody()`** — restructured workflow steps:
- Step 1: "Discover" → "Check existing work"
- Step 2: "Plan" → "Plan (Write great issues)" with bug investigation guidance
- Step 3: Spec — added `rationale` tool for checking prior design decisions, added gate ("Do not proceed to step 4 until the spec exists")
- Step 4: Start — reorganized with explicit gates section, assignee set before `start`
- Step 5: Implement — added "file new issues for out-of-scope bugs, don't fix silently"
- Step 6: "Completion Comment" → "Handoff" with self-review checklist
- Step 7: "Review & Spec Update" → "Revisions" with "go back to step 6"
- Step 9: Complete — explicit "use `merge` tool, not manual git commands"
- Reference section: status table with CANCELED/DUPLICATE, Issue Labels (primary/secondary system), Backlog Review section

**`generateMCPToolsBody()`** — added `rationale` tool row.

## Skill directory rename (xpo-workflow → xpo)

Aligned everything to match the reference skill's `name: xpo`:
- `WriteAgentSkill` and `DetectSkillInstall` use `"xpo"` directory
- Agent stub says "load the `xpo` skill"
- All tests updated

## CLI polish

**Colored output** — added lipgloss-styled prefixes: green `✓`, yellow `!`, red `✗`. Replaced the `ℹ` info icon with `!` for warnings. Removed the no-op `Stylize()` wrapper.

**`xpo init` improvements:**
- Quick health check: shows MCP and skill status as `!` warnings (without naming specific harnesses)
- Interactive prefix prompt: `Issue ID prefix [myproject-]:` — user can accept or customize
- `InitProject()` refactored to accept prefix parameter; `DefaultPrefix()` exported
- Single call-to-action: "Run xpo doctor to address any warnings."

**`xpo doctor` improvements:**
- `doctorCounts` tracker with summary line ("7 passed, 4 warnings")
- Clean section headers with right-aligned fix command
- One fix hint per section instead of per item
- Tighter line format

**`xpo` root command** — "The git-native engineering system for human-AI teams" (was "A JSONL-based issue tracker")

## Follow-on

Filed xpo-76e2a1 in BACKLOG for further first-run UX polish (wizard flow, output formatting, interactive prompts).
