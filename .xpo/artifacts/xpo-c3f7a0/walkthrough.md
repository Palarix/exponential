# Walkthrough: Add `--json` output to `blocked` command

## What changed

One file: `cmd/exponential/blocked.go`.

## How it works

The `blocked` command was previously mutation-only (`xpo blocked <id>` marks an issue as
BLOCKED). Now it's dual-mode:

- **No args** (`xpo blocked`): lists all issues with status BLOCKED via
  `client.ListIssues(FilterOptions{Statuses: ["BLOCKED"]})`. Renders as a terminal table
  by default, or as `jsonio.ListOutput` JSON when `--json` is set.
- **One arg** (`xpo blocked <id>`): marks the issue as BLOCKED (unchanged behavior).

Changed `Args` from `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)` to support both modes.
Updated `Short` description to reflect the dual purpose.

Reuses `jsonio.ListOutput` — blocked issues are just a filtered issue list, so no new
envelope type was needed.
