# Spec: Content-based staleness for managed instruction blocks

## What

`CheckAgentHealth` compares an unedited managed block with the content `WriteAgentInstructions` would write now (stub or full docs, depending on whether the agent has a skill dir). If they differ, the block is stale and auto-fixable. `xpo init` (same version) refreshes it silently, and `xpo doctor --fix` includes it in the fix plan.

## Why

Staleness is currently gated on the version stamp only, so template changes shipped without a version bump never reach existing repos. Skills already compare content (xpo-e88f1b). Blocks should match.

## How

1. `CheckAgentHealth(agent, integrationVer, prefix string)`: new `prefix` parameter, because the stub embeds the issue prefix. Callers already have it: `doctorIntegrations(prefix)`, and init's `freshCfg.Prefix`.
2. A new helper `GeneratedInstructions(agent, prefix)` returns `GenerateAgentStub(prefix)` when the agent has a `SkillDir`, else `GenerateAgentDocs(prefix)`. `WriteAgentInstructions` uses the same helper, so the two can't drift.
3. A new field `AgentHealth.BlockStale`. For an unedited block:
   - version older → existing "Update available (X → Y)" AutoFix (unchanged), and `BlockStale = true`
   - else, content (trimmed) ≠ generated (trimmed) → `BlockStale = true`, AutoFix "Instructions out of date"
4. `xpo init` Case A (version matches): agents with `BlockStale` (and not edited) get `WriteAgentInstructions(agent, prefix, false)`, with the line "✓ Claude Code: instructions updated to the 1.2.1 template". This mirrors how stale skills are handled there.
5. `xpo doctor --fix`: no change needed. A non-empty AutoFix already routes the agent through `applyInit`, which rewrites the instructions.

## Decisions

- **Compare trimmed content**, like `CheckSkillFile` and `ComputeBlockHash`, so whitespace at the edges doesn't count.
- **Edited blocks are unchanged:** still Interactive (replace/keep/diff). Edited wins over stale.
- **Legacy hash-less blocks** already count as not edited, so they will refresh if their content differs. This is the same as what a version bump does to them today.

## Acceptance Criteria

- [ ] An unedited block whose content differs from the current template, at the same version, is reported by `CheckAgentHealth` as AutoFix "Instructions out of date".
- [ ] A block matching the template reports no problem (no regression in `TestHealth_FullyConfiguredAgent_NoProblem`).
- [ ] An edited block is still Interactive, not AutoFix.
- [ ] The stale-version case still reports "Update available" and does not double-report.
- [ ] `xpo init` at the same version rewrites a stale block and leaves the user's content outside the block untouched.
- [ ] `make test` passes.
