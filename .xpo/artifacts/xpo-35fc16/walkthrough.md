# Walkthrough: Principal-first assignees

## What was built and why

xpo had **54 distinct actor strings** for about 4 real actors. Agents were told to build their own identity (`Claude Code <agent@<host>.local>`). They guessed hostnames and changed letter case, and every machine produced a new "Claude Code". In the board this showed up as ten identical-looking "Claude Code" rows that never merged, because the frontend hid the email with `split(" <")[0]`.

The fix changes the model instead of merging agent identities. **An issue is always assigned to a person (the principal).** An agent is how that person got the work done, not someone you assign work to. Two things made this possible without rewriting history:

- Every event an agent writes through MCP already carries `on_behalf_of`, the principal.
- All 126 historical events that assigned an issue to an agent have it, so 100% of old data resolves.

After projection, the real event log's 12 raw assignee strings resolve to 2 people: the user's private and work identities, which are intentionally kept separate.

## How the pieces fit together

### 1. `internal/identity` (new package)

This is the single place that interprets `"Name <email>"` strings. It lives in its own package so both `internal/ui` (CLI rendering) and `internal/exponential` (core) can import it without a cycle.

- `IsAgent(id)`: the email's local part is `agent` (`agent@mcp`, `agent@<host>.local`, drive's `agent@<host>`). `agent.smith@…` is *not* an agent.
- `AgentName(id)`: `claude-code/2.1.263 <agent@mcp>` → "Claude Code", `codex-mcp-client/…` → "Codex". Drive's `claude`/`codex` supervisor names are mapped too. Unknown agents pass through with any `/version` suffix removed.
- `Actor(createdBy, onBehalfOf)`: a principal-first label, "Nicolas (via Claude Code)", used by the CLI timeline and the inbox.
- `ResolveAssignment(raw, createdBy, onBehalfOf) (assignee, via)`: the core rule.

| Raw assignee written | Result |
|---|---|
| agent identity, event has `on_behalf_of` | assignee = principal, via = agent |
| agent identity, written by a human (e.g. the UI) | assignee = that human, via = agent |
| agent identity, no human anywhere (old drive runs) | assignee = raw (fallback) |
| the principal, written by an agent acting for them (new `start`) | assignee = principal, via = agent |
| any other human | plain assignment, no via |

The last row matters. An agent assigning a *different* person (e.g. "assign this to Alice") records no via, because via means "an agent is doing this work for this person", not "an agent clicked assign".

### 2. Projection: resolve on read, never rewrite

`ProjectIssues` calls `ResolveAssignment` on CREATE and UPDATE and fills `Issue.Assignee` (always a person) and the new `Issue.AssigneeVia`. The event log keeps the raw values forever. The resolution is purely derived, so changing the rule later never needs a migration.

`AssigneeVia` is exposed as `assignee_via` everywhere an issue is serialized: `jsonio` (CLI `--json`, MCP), `server/responses.go` (HTTP API), and `remote_transport.go` (so remote mode round-trips it).

`storage/collapse.go` is intentionally unchanged. Its no-op pruning compares *raw* values, which is still correct at the storage layer.

Downstream code didn't need special cases. My Issues, the inbox, `--mine`, metrics workload and the filter menu all key off `Assignee`, which is now a person, so agent rows fold into the principal's automatically.

### 3. `start` assigns the principal

`Client.startPayload` builds a single UPDATE containing both `status: DOING` and the assignee. The principal is `c.OnBehalfOf` (set by the MCP server and drive), falling back to `c.GetUser()` for a human running the CLI.

- Unassigned, already the principal, or held by an agent identity: assign the principal.
- Held by another person: left unchanged, with the notice "Assigned to Alice, left unchanged".
- `--force` (takeover): reassign to the starter.

One subtle fix: `StartWork` now calls `c.syncLocal()` first. It used `c.Transport` directly, so the agent identity (`UserOverride` and `OnBehalfOf`) was only applied if some earlier client call had synced it. MCP happened to call `GetIssue` first; drive and tests did not.

### 4. drive acts for the user

`setDriveIdentity` sets `OnBehalfOf = c.GetUser()` (the git-config user) and `UserOverride = "<supervisor> <agent@host>"`. Drive's events now carry a principal like MCP events do. This closes a gap left by xpo-8b54d8.

### 5. Agents no longer format their own identity

The "Agent Identity" section and the "set assignee via update before start" step are gone from:
- the generator templates in `agents.go`
- the checked-in CLAUDE.md, AGENTS.md and GEMINI.md
- the three skill copies

The skill now says `start` assigns the person you are working for and that agents should never set `assignee` to themselves. The checked-in managed blocks and skill files were regenerated with their `sha256` markers recomputed, so `doctor`/`init` see them as up to date rather than hand-edited. `TestAgentTemplates_DoNotAskAgentsToAssignThemselves` guards against regressions.

### 6. Web UI

**Avatars and the assignee always show the person.** `Avatar` is unchanged.

The tophat went through three revisions; they're recorded here so nobody re-proposes them:
1. A robot avatar (replacing the face at xs/sm, a badge at md/lg) was rejected: it swapped almost every face on the board for a robot.
2. "Nicolas via Claude Code" next to the assignee was rejected as awkward.
3. A robot count in the dashboard workload column was rejected, and its backend `via_agent` metric removed with it.

Where the agent *is* visible:
- **Agent-written comments**: the principal's avatar, with the author name "Claude Code on behalf of Nicolas Bettenburg" (`agentCommentAuthor`). Comments are the one place the agent is named up front, because who wrote the words matters. The same wording is used for comment events in the Overview activity feed and the Timeline view.
- **Other activity events** lead with the person, and hovering the name shows "Claude Code for Nicolas Bettenburg".
- **"Worked by" filter** (Agent / Human), driven by `assignee_via`.

Supporting helpers in `utils/format.ts` mirror the Go rules:
- `isAgentIdentity`, `agentDisplayName`, `agentTooltip`, `agentCommentAuthor`.
- `resolveAssignee`, so old "assigned to Claude Code" events read "assigned to Nicolas".
- `displayActor` and `activityActor` are principal-first. Their old naming was inverted: they labelled the agent as the principal.

`collectKnownPeople` excludes `agent@` identities, so the assignee pickers list only people. `parseStoredFilters` merges saved localStorage filters with `EMPTY_FILTERS`, so filters saved before `workedBy` existed don't crash `matchesFilters`.

## Key decisions

- **Principal-first, not agent-first.** This reverses xpo-53ed0e (merged the same day), which had made the activity avatar show the agent with an "on behalf of" tooltip. The user chose the flip deliberately.
- **No mailmap or email merging.** `nicbet@gmail.com` and `nicbet@kuy.io` are separate on purpose (private vs. work).
- **`created_by` keeps its raw format**, version and host included. It's no longer used as a display identity, so it's harmless metadata.
- **`start` doesn't steal issues.** Another person's assignment is kept unless `--force`.

## Non-obvious things for future readers

- `Issue.Assignee` is **derived**. To see what was literally written, look at the event payload, not the issue.
- If an agent is configured with `XPO_AGENT_IDENTITY` using a non-`agent@` email, the "human raw == on_behalf_of and created_by ≠ on_behalf_of" branch still detects that an agent did the work.
- `agentCommentAuthor` accepts both raw identities and already-shortened names (the activity feeds pass display names).

## Follow-up

- xpo-a103c3: the merge view renders comment authors from `issue.comments[].created_by` (raw). `model.Comment` needs `OnBehalfOf` so it can use the same "on behalf of" label.

## Acceptance Criteria

- [x] An issue assigned to an agent (old or new) shows the principal's name everywhere. *Evidence:* the real-log projection showed 2 people instead of 12 strings; the board, sidebar and workload were checked at tophat; tested by `TestProjectIssues_PrincipalFirstAssignee`.
- [x] Avatars always show the person, never a robot. *Evidence:* `Avatar.tsx` unchanged from main; confirmed at tophat.
- [x] The assignee is shown as the person only. *Evidence:* sidebar confirmed at tophat; CLI tested by `TestRenderIssueDetails_AssigneeVia`.
- [x] The assignee filter lists only people. *Evidence:* the filter keys off the resolved `assignee`; pickers tested by `collectKnownPeople` "excludes agent identities".
- [x] New "Worked by" filter (Agent/Human). *Evidence:* the `workedBy filter` tests in `filters.test.ts`.
- [x] My Issues, the inbox and `--mine` include agent work for me. *Evidence:* they match on the principal's email; tested by `TestFormatInboxItem` "legacy agent self-assignment names the principal".
- [x] Activity timeline is principal-first; agent comments read "Claude Code on behalf of …". *Evidence:* the `activityActor` and `agentCommentAuthor` tests; confirmed on the issue page and the Overview feed at tophat.
- [x] CLI `FormatActor` prints "Nicolas (via Claude Code)". *Evidence:* `TestFormatActor_OnBehalfOf`, `TestRenderTimeline_OnBehalfOf`.
- [x] Dashboard workload shows the person only. *Evidence:* `TestMetrics_WorkloadFoldsAgentWorkIntoPrincipal`; confirmed at tophat.
- [x] `start` assigns the principal automatically. *Evidence:* `TestStartWork_AgentAssignsPrincipal`, `_AssignsHumanStarter`, `_ReplacesLegacyAgentAssignee`, `_KeepsOtherHumanAssignee`, `_ForceTakeoverReassigns`.
- [x] `xpo drive` sets `on_behalf_of`. *Evidence:* `TestSetDriveIdentity_ActsForUser`.
- [x] The identity rules are removed from all agent docs and templates. *Evidence:* `TestAgentTemplates_DoNotAskAgentsToAssignThemselves`; regenerated files match the template (`CheckSkillFile` reports up to date).
- [x] The event log is not rewritten. *Evidence:* resolution happens only in `ProjectIssues`; no storage writes changed.
- [x] `make test` passes. *Evidence:* lint clean, 401 frontend tests, all Go packages ok.
