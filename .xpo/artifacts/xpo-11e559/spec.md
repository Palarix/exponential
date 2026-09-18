# Spec: Align rationale --json with shared output types

## What

Refactor `rationale --json` to output `jsonio.RationaleOutput` instead of the raw
`exponential.RationaleSearchResult`. Fields are identical — this is a type alignment,
not a schema change.

## Flow

1. In `rationale.go`, convert `result.Results` to `[]jsonio.RationaleHit` and wrap in
   `jsonio.RationaleOutput` before encoding.
