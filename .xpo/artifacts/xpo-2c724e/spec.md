# Work Mode: allow edits to the bound issue's artifacts

## What

In Work Mode, Claude may edit files in the bound issue's artifacts folder, `<root>/.xpo/artifacts/<issue>/`, without an approval card.

## Why

The `spec` tool can only rewrite the whole file, and its result includes the file path. For a small change, the agent runs `Edit` on that path. The path is outside the worktree, so the only allow rule, `Edit(//<worktree>/**)`, doesn't match, and each spec update after user feedback asks the user. The xpo skill tells the agent to update the spec before code on every revision, so the card comes up all the time.

## How

1. `WorkProfile(issueID, worktree, root)` takes the project root. `Profile` gets a `Root` field.
2. `claudeAllowed` in Work Mode adds `Edit(//<root>/.xpo/artifacts/<issueID>/**)`, plus the symlink-resolved path when it differs, the same way as the worktree rule.
3. `Runner.profileFor` passes `r.cfg.Dir` as the root.
4. The rule is skipped when `Root` or `IssueID` is empty, so a profile without them never allows `//.xpo/artifacts/**`.

Out of scope: Codex. Its `workspace-write` sandbox mode has no extra writable roots on `thread/start`, and it asks on request. A later issue can cover it if needed.

## Acceptance criteria

- [ ] The Work Mode `--allowedTools` list contains `Edit(//<root>/.xpo/artifacts/<issue>/**)`.
- [ ] A symlinked root also adds the resolved path.
- [ ] No rule covers other issues' artifacts or `.xpo` as a whole.
- [ ] Chat Mode is unchanged.
- [ ] `make test` passes.
