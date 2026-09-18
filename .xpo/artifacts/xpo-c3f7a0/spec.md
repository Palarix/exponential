# Spec: Add `--json` output to `blocked` command

## What

Make the `blocked` command dual-mode: without an ID it lists all blocked issues,
with an ID it marks that issue as BLOCKED (existing behavior). Add `--json` flag
for the list mode.

## Flow

1. Change `Args` from `ExactArgs(1)` to `MaximumNArgs(1)`.
2. If no args: list issues with status BLOCKED via `client.ListIssues(FilterOptions{Statuses: ["BLOCKED"]})`.
   - If `--json`: output as `jsonio.ListOutput` via `jsonio.ToIssueSummary()`.
   - Otherwise: render with `ui.RenderIssueList()`.
3. If one arg: existing mark-as-blocked behavior (unchanged).

## Decisions

1. **Reuse `jsonio.ListOutput`** — blocked issues are just a filtered list, same envelope.
