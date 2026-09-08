# Walkthrough: MCP default strategy test

## What was built

`TestMCPMerge_DefaultStrategy` — passes `Strategy: ""` and verifies:
1. Merge succeeds with a non-empty SHA
2. HEAD is a single-parent commit (squash semantics, not a merge commit)

This complements `TestMCPMerge_Squash` which explicitly passes `"squash"` — the new test proves the empty-string default path produces the same result.
