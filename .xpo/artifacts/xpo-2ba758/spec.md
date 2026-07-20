# Spec: Interpolate prefix in GenerateAgentDocs/GenerateAgentStub

## Problem

`GenerateAgentStub(prefix)` and `GenerateAgentDocs(prefix)` accept a `prefix` parameter but immediately discard it with `_ = prefix`. The generated agent instructions never tell agents what issue ID format the project uses.

## Fix

Remove the `_ = prefix` discard and interpolate the prefix into the generated text. Specifically, add a line to the stub's "Hard Rules" or top section noting the project's issue ID format, e.g.:

> Issue IDs in this project use the prefix `xpo-` (e.g. `xpo-a1b2c3`).

This gives agents enough context to recognize and construct issue IDs correctly.

## Files changed

- `internal/exponential/agents.go` — `GenerateAgentStub` function

## Acceptance criteria

- [ ] `GenerateAgentStub` uses the `prefix` parameter in its output
- [ ] The generated text tells agents the project's issue ID prefix and shows an example
- [ ] `GenerateAgentDocs` inherits the fix via its call to `GenerateAgentStub`
- [ ] Existing tests pass
