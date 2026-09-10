# Polish first-run CLI UX

## What

Redesign `xpo init` into a unified, wizard-style experience that walks users through project setup, MCP configuration, and skill installation in one shot — with a terraform-style change plan and explicit confirmation. Remove the `init mcp` and `init skill` subcommands. Reorganize `xpo doctor` into clearer sections with a `--fix` flag. Introduce managed blocks with version stamping so integrations can be safely refreshed.

## Why

The first-run experience is the first impression users have of xpo. Today it works correctly but feels like raw plumbing: three separate commands, flat checkmark output, no visual hierarchy, and no change plan before writing files. The design document (`cli-design.md`) defines a polished, trust-building UX that shows users exactly what will change before changing it.

## Acceptance Criteria

- [x] `xpo` with no args in a non-project directory shows branded welcome with command hints
- [x] Any non-init command in a non-project directory shows the same branded welcome
- [x] `xpo init` in a git repo runs the full wizard: prefix prompt → agent multi-select → change plan → confirm → apply
- [x] `xpo init` in a non-git directory offers to create a repository before proceeding
- [x] All prompts happen before any writes — Ctrl-C or decline at any point leaves no partial state
- [x] `xpo init` in an already-initialized, up-to-date project prints a one-liner
- [x] `xpo init` in an already-initialized project with stale integrations shows version diff and change plan
- [x] `xpo init` refuses to downgrade when the project's integrations are from a newer CLI version
- [x] `xpo init` on up-to-date project with edited managed block shows targeted replace/keep/diff prompt (not full re-init)
- [x] Agent instruction files use `<!-- xpo:begin -->` / `<!-- xpo:end -->` managed blocks with version + content hash
- [x] Refreshing integrations replaces only the managed block, never touches user content outside it
- [x] When a managed block has been edited by hand (hash mismatch), the user is prompted: replace / keep / show diff
- [x] `init mcp` and `init skill` subcommands are removed
- [x] `xpo doctor` uses reorganized sections: xpo, Project, Integrations, This machine
- [x] `xpo doctor` validates issues.db content and reports invalid entries with line number
- [x] `xpo doctor --fix` auto-fixes local/gitignored state silently, shows a plan + confirm for committed files
- [x] `xpo doctor --fix` offers replace/keep/diff prompt for edited managed blocks inline
- [x] `xpo doctor --fix` only fixes agents that are already configured (not unconfigured ones)
- [x] Non-interactive flags: `--yes`, `--prefix`, `--agents`, `--force`
- [x] `--strict` flag on doctor for CI (warnings exit non-zero)
- [x] Output follows conventions: sentence-case fragments, past tense for results, `→` for versions, `·` as separator
- [x] Skills are always installed locally (project-level); no global/local prompt
- [x] Prefix is stored without the separator (e.g. `pay` not `pay-`); the `-` is added at point of use
- [x] Tagline defined once (`ui.Tagline`) and used consistently across all CLI output
- [x] Decline messages are friendly: "No changes were made. When you're ready: ..."
- [x] Exit codes: decline exits 1, Ctrl-C exits 130

## Decisions (final, post-review)

1. **Eliminate subcommands**: `init mcp` and `init skill` removed. `xpo init` is the single entry point.
2. **Always local skills**: No global/local prompt. Refreshed by re-running `xpo init`.
3. **Prefix without separator**: Store `pay` not `pay-`. Normalized on config load.
4. **Interactive library**: `huh` for multi-select and select (arrow key navigation). Simple inline `(Y/n)` prompts for confirms. Raw-mode live-updating input for prefix with preview line.
5. **Managed block migration**: Legacy heading-based sections auto-migrate to markers on first refresh.
6. **Health check classification**: `AutoFix` (silent), `Interactive` (needs user decision), `attention` (truly unfixable like corrupted issues.db). Doctor --fix handles both AutoFix and Interactive; only attention goes to "Cannot fix automatically."
7. **Doctor --fix scope**: Only fixes agents already configured in the project. Unconfigured agents (CLI on PATH but never set up) shown as informational hint pointing to `xpo init`.
8. **All questions before all writes**: The init wizard collects every answer (including git init consent) before touching any file.
