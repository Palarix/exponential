# Walkthrough: Content-based staleness for managed instruction blocks

## What was built and why

`xpo init` and `xpo doctor` only considered the managed instruction block in CLAUDE.md, AGENTS.md etc. (`<!-- xpo:begin … -->`) out of date when its version stamp was older than the CLI's. xpo-35fc16 changed the stub template (removing "Agent Identity") without bumping the version from 1.2.1. As a result, every existing repo would have kept the old text forever: `init` said "up to date" and `doctor` said "Up to date".

Skills already avoided this: xpo-e88f1b made them compare file content with the template. This change gives instruction blocks the same treatment.

## How it works

1. **`GeneratedInstructions(agent, prefix)`** (`agents.go`) is now the single definition of what goes into the block: the short stub when the agent has a skill dir, the full docs otherwise. `WriteAgentInstructions` calls it too, so "what we check against" and "what we write" can't diverge.
2. **`CheckAgentHealth(agent, integrationVer, prefix)`** gained a `prefix` parameter, because the stub embeds the issue prefix. For an unedited block:
   - older version → `BlockStale = true` plus the existing "Update available (X → Y)". There is no second message.
   - otherwise, trimmed block content ≠ trimmed generated content → `BlockStale = true` plus AutoFix "Instructions out of date".
   Edited blocks stay Interactive (replace/keep/diff), and edited wins over stale.
3. **`xpo init` (Case A, same version)** collects agents whose block is stale but not edited, and rewrites them with `WriteAgentInstructions(agent, prefix, false)`. It prints "✓ Claude Code: instructions updated to the 1.2.1 template", mirroring how stale skills are reported there.
4. **`xpo doctor --fix`** needed no new code. A non-empty AutoFix already routes the agent through `applyInit`, which rewrites the instructions.

## Non-obvious decision: an unknown prefix skips the check

Without a loaded config, `doctor` uses the placeholder prefix `"issue"` for display. Comparing blocks against a stub generated with that placeholder would flag every block as stale, and `--fix` would then write the wrong prefix into the user's CLAUDE.md. So:
- `CheckAgentHealth` skips the content comparison when `prefix == ""`.
- doctor's `checkAgentHealth` wrapper passes `cfg.Prefix` (empty when there's no config), never the display fallback.

## Rewriting is safe for user content

`ReplaceManagedBlock` only replaces the bytes between the markers, so anything the user wrote outside the block is preserved. A test covers this, and the end-to-end run below confirmed it.

## Acceptance Criteria

- [x] An unedited block whose content differs from the template, at the same version, is AutoFix "Instructions out of date". *Evidence:* `TestHealth_StaleBlockContent_SameVersion_IsFixable`.
- [x] A block matching the template reports no problem. *Evidence:* `TestHealth_FullyConfiguredAgent_NoProblem` and `TestHealth_CurrentVersion_NoProblem` still pass.
- [x] An edited block is still Interactive, not AutoFix. *Evidence:* `TestHealth_EditedBlock_IsManualNotFixable` still passes.
- [x] An older version reports "Update available" without a double report. *Evidence:* `TestHealth_StaleVersion_DoesNotDoubleReport`, `TestHealth_StaleVersion_ReportsUpdate`.
- [x] `xpo init` at the same version rewrites a stale block and keeps content outside it. *Evidence:* `TestHealth_StaleBlock_RefreshKeepsOutsideContent`. End-to-end with the built binary in a throwaway project: `doctor` showed "Instructions out of date"; `init` printed "instructions updated to the 1.2.1 template", removed the injected "Agent Identity" section and kept "Keep me." outside the block; `doctor` then showed "Up to date".
- [x] `make test` passes.

## Related

- xpo-d053b5 was found during the end-to-end run: xpo stalls under a pty because bubbletea's package init queries the terminal's background colour.
