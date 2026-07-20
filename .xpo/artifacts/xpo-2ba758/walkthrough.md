# Walkthrough: Interpolate prefix in agent docs

## What changed

**File:** `internal/exponential/agents.go` — `GenerateAgentStub` function

## The fix

The `_ = prefix` discard was removed and replaced with a line interpolating the prefix into the generated agent instructions, placed between the introductory paragraph and the "Hard Rules" section:

```
Issue IDs in this project use the prefix `xpo-` (e.g. `xpo-a1b2c3`).
```

The prefix value comes from the project's config (`cfg.Prefix`, defaulting to `issue-`). Since `GenerateAgentDocs` delegates to `GenerateAgentStub` for its header, both functions now correctly include the prefix in their output.

This ensures that agents reading the generated instructions know what issue ID format to expect and can correctly construct or reference IDs.
