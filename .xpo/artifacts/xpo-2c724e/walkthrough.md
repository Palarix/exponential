# Walkthrough: Work Mode allows edits to the bound issue's artifacts

## The problem

In Work Mode, a Claude chat runs with an allow list. A tool call that isn't on the list turns into an approval card (`--permission-prompt-tool stdio`). The list only allowed edits inside the worktree, `<root>/.xpo/worktrees/<branch>`.

Specs and walkthroughs aren't in the worktree. They're in the main checkout at `<root>/.xpo/artifacts/<issue>/`. The `spec` tool can only rewrite the whole file, and its result includes the file's path. For a small change, the agent edits that path with `Edit`. The xpo skill says to update the spec on every round of feedback, so the user got a card after nearly every reply.

## The fix

**`internal/chat/profile.go`**
- `Profile` gets a `Root` field, and `WorkProfile(issueID, worktree, root)` sets it.
- In Work Mode, `claudeAllowed` adds `Edit(//<root>/.xpo/artifacts/<issue>/**)` next to the worktree rule. An `Edit` rule also covers `Write`, so creating a walkthrough file works too.
- The rule is only added when both `Root` and `IssueID` are set. A profile without them never allows a bare `//.xpo/artifacts/<issue>/**`.
- A new `editRules(base, sub)` builds the rule for `base/sub`, plus the version with `base` symlink-resolved. The worktree rule now uses it with an empty `sub`.

**`internal/chat/mode.go`**
- `profileFor` passes `r.cfg.Dir` as the root. The board sets this to `storage.HubRoot()`, the same base the `spec` tool uses for `XpoDir()`, so the rule matches the path the agent gets back.

## Why only the base is resolved

The first version resolved the whole artifacts path. The folder doesn't exist until the issue's first spec is written, and `filepath.EvalSymlinks` fails on a missing path, so the resolved rule was silently left out. On macOS, temp dirs live under `/var` → `/private/var`, and the test caught it there. Resolving the root, which always exists, and joining the artifacts path onto it gives the right rule either way.

## Scope

- Only the bound issue's folder is allowed. Editing another issue's spec is a change the user should see, so it still asks.
- Chat Mode is unchanged: it has no `Edit` tool.
- Codex is not covered. Its `workspace-write` sandbox mode on `thread/start` can't take extra writable roots, and outside writes ask through `on-request`.

## Acceptance criteria

- [x] The Work Mode `--allowedTools` list contains `Edit(//<root>/.xpo/artifacts/<issue>/**)`. Covered by `TestWorkProfile_EditRuleForIssueArtifacts`.
- [x] A symlinked root also adds the resolved path. The same test asserts the `/private/var/...` rule on macOS temp dirs.
- [x] No rule covers other issues' artifacts or `.xpo` as a whole. The same test checks that no rule ends in `/.xpo/**)` or `/.xpo/artifacts/**)`. `TestWorkProfile_NoArtifactRuleWithoutRoot` covers an empty root.
- [x] Chat Mode is unchanged. Covered by `TestChatProfile_NoArtifactRule` and the existing `TestChatProfile_Claude`.
- [x] `make test` passes (exit 0).
