# Spec: Principal-first assignees — agents act *for* a person

## What

Assignees always resolve to the **principal**: the human the work is for. The assignee is displayed as that person (name and avatar) everywhere, even when their agent does the work. Agents stop writing their own identity strings: `start` assigns the issue to the principal automatically. Which agent did the work is kept as data (`assignee_via`) and surfaces in two places only: a "Worked by" filter, and agent-written comments, which are labelled "Claude Code on behalf of <Principal>" (with the principal's avatar) so it is explicit that an agent wrote them.

## Why

The event log has **54 distinct actor strings** for about 4 real actors. Ten of them are "Claude Code" assignees that differ only in hostname (`agent@macbook.local`, `agent@archlinux.local`, `agent@MBPM1X.local`, `agent@mbpm1x.local`, …), and about 45 are `claude-code/<version> <agent@mcp>` authors. There are three causes:

1. CLAUDE.md, AGENTS.md, GEMINI.md and the skills tell agents to format `<agent@<host>.local>` by hand. Agents guess the hostname, change its case, and run on many machines.
2. The frontend hides the email with `split(" <")[0]`, so these identities look identical but never merge.
3. Agents are treated as people you can assign work to. They are really the way a person gets the work done.

The data already supports the fix: **all 126 historical events that assigned an issue to an agent also carry `on_behalf_of`**. So the principal can be resolved for 100% of history with no migration and no guessing.

This finishes the "principal-first" rule from xpo-8b54d8, which was never fully applied. `activityActor` in `web/src/utils/format.ts` still shows `created_by` as the name, and `drive` still doesn't set `on_behalf_of`.

## Acceptance Criteria

- [ ] An issue assigned to an agent (old or new) shows the principal's name everywhere: board, backlog, My Issues, issue sidebar, filter menu, dashboard workload, and the CLI `list`/`show`.
- [ ] Avatars always show the person, never a robot, including for agent-worked issues and agent-authored activity.
- [ ] The assignee is shown as the person only: no "via <Agent>" next to the assignee.
- [ ] The assignee filter menu lists only people. The 10 "Claude Code" rows collapse into the principal's row.
- [ ] A new **Worked by** filter dimension with options *Agent* and *Human*.
- [ ] My Issues, the inbox and `xpo list --mine` include issues an agent worked on for me, with no special-casing (the principal's email matches).
- [ ] Activity timeline (web): the principal is the actor, shown with their own avatar, e.g. "Nicolas Bettenburg moved to DOING". Hovering the name of an agent-made event shows "Claude Code for Nicolas Bettenburg". Comments written by an agent keep the principal's avatar, but the author name reads "Claude Code on behalf of Nicolas Bettenburg". Comments are the one place the agent is named up front, because who wrote the words matters. The same label is used for comment events in the Overview activity feed and the Timeline view ("Claude Code on behalf of Nicolas Bettenburg commented on …").
- [ ] CLI timeline: `FormatActor` prints "Nicolas Bettenburg (via Claude Code)". Today it prints the reverse.
- [ ] Dashboard workload shows the person only, with no agent breakdown.
- [ ] MCP `start` assigns the issue to the principal automatically. The agent never sets `assignee` itself.
- [ ] `xpo drive` sets `on_behalf_of` to the git-config user on every event it creates.
- [ ] The "Agent Identity" rules are removed from CLAUDE.md, AGENTS.md, GEMINI.md, `.claude/.codex/.opencode` skills, and the templates in `internal/exponential/agents.go`. Skill step 4 no longer says "set assignee before start".
- [ ] The event log is not rewritten. All resolution happens at projection/read time.
- [ ] `make test` passes, with new tests for resolution rules, `start` auto-assign, drive's `on_behalf_of`, `FormatActor`, and the frontend helpers.

## Model

The projected `Issue` gets one new field:

```go
Assignee    string // always a principal ("Name <email>") or empty
AssigneeVia string // raw agent identity that did the work for the assignee, or ""
```

**Agent detection** (`isAgentIdentity`): the email's local part is `agent` (`agent@*`). This covers `agent@mcp`, `agent@<host>.local`, and drive's `agent@<host>`.

**Resolving an UPDATE event that sets `assignee = raw`** (in `projection.go`; `storage/collapse.go` prunes no-op updates on the *raw* value and is unchanged):

| Case | `Assignee` | `AssigneeVia` |
|---|---|---|
| `raw` is an agent identity, event has `on_behalf_of` (all old agent self-assignments) | `on_behalf_of` | `raw` |
| `raw` is an agent identity, no `on_behalf_of`, `created_by` is human (a human assigned an agent in the UI) | `created_by` | `raw` |
| `raw` is an agent identity, no human anywhere | `raw` (fallback) | `raw` |
| `raw` is human and equals the event's `on_behalf_of`, and `created_by` is an agent (new-style auto-assign by `start`) | `raw` | `created_by` |
| `raw` is human otherwise (including an agent assigning to a *different* person) | `raw` | `""` |
| `raw` is empty (unassign) | `""` | `""` |

The `raw` value as typed in the old log is shown nowhere in the UI. It stays in the event log for history only.

**Agent display name** (`agentDisplayName`, Go and TS): `claude-code/2.1.263 <agent@mcp>` → "Claude Code"; `codex-mcp-client/0.154.0 <agent@mcp>` → "Codex"; `Claude Code <agent@macbook.local>` → "Claude Code"; `claude <agent@NICSPC>` (drive's supervisor) → "Claude Code"; anything else → its name portion with any `/version` suffix removed. A small table of known client names, with everything else passed through unchanged.

## Flow

1. **Go resolver**: new package `internal/identity` (importable from `ui` and `exponential` without a cycle): `IsAgent`, `AgentName`, `Actor` (principal-first label), `Same`, `ResolveAssignment`. Table-driven tests first. `AgentName` also maps drive's `claude`/`codex` supervisor names.
2. **Projection**: `projection.go` applies `ResolveAssignment` on CREATE and UPDATE. Add `AssigneeVia` to `model.Issue`, `jsonio` output, `server/responses.go`, `remote_transport.go`, and `web/src/api/types.ts` (`assignee_via?: string`).
3. **`start` auto-assigns** (`Client.startPayload`, shared by MCP, CLI and drive; status and assignee go in one UPDATE): in `StartWork`, before the DOING transition, set `assignee = OnBehalfOf` (the principal). This happens if the issue is unassigned, already assigned to the principal, or assigned to an agent identity. If the issue is assigned to a different human, leave it and return a message ("assigned to X, left unchanged"). The CLI `xpo start` gets the same rule with the git-config user (a human starting work takes it too).
4. **Drive**: in `drive.go`, capture `c.GetUser()` before the override, then set `c.OnBehalfOf` to it alongside `c.UserOverride`.
5. **Metrics**: unchanged. Workload already groups by `Assignee`, which is now the principal, so agent work folds into the person's row. (A robot count in the WIP column was tried during tophat and rejected.)
6. **CLI**: the inbox sentence (`FormatInboxItem`) is principal-first too, and resolves legacy agent assignees to the person. Flip `ui.FormatActor` to `"<principal> (via <agentDisplayName(createdBy)>)"`. `show` prints the assignee as the person only.
7. **Frontend helpers** (`utils/format.ts`): `isAgentIdentity`, `agentDisplayName`, `agentTooltip`, `resolveAssignee` (TS mirror of the Go rule, used by the activity, timeline and dashboard feeds to say "assigned to Nicolas"). `displayActor` and `activityActor` become principal-first and return the agent as `via`.
8. **`Avatar`**: unchanged. It always renders the person passed in, which after projection is the principal. (A robot avatar and badge were tried during tophat and rejected: they replaced almost every face on the board.) The PropertySidebar shows the assignee's name only.
9. **Filters**: the assignee options already key off `issue.assignee`, so they now collapse on their own. Add a "Worked by" dimension (Agent/Human) to `FilterMenu`, `FilterChips`, and the `BacklogFilters` type and matcher in `filters.ts`. `parseStoredFilters` merges filters saved in localStorage with `EMPTY_FILTERS`, so filters saved before `workedBy` existed keep working. Replace the `split(" <")[0]` copies with `shortName`.
10. **Comments**: `ActivityTimeline` comment header shows the principal's avatar and the author name "Claude Code on behalf of Nicolas Bettenburg" (`agentCommentAuthor` in `utils/format.ts`) when the comment was made by an agent.
11. **Docs and templates**: remove the "Agent Identity" section and the "set assignee via update before start" step from CLAUDE.md, AGENTS.md, GEMINI.md, the three skill copies, and the `agents.go` templates. Update `docs/design.md` "Identity resolution" with the principal-first rule.
12. `make test`.

## Decisions

- **Principal-first, not agent-first.** The person is accountable; the agent is how the work got done. This removes the need to merge agent identities at all.
- **Resolve on read, never rewrite the log.** The event log is append-only, and `on_behalf_of` is present on 100% of historical agent assignments.
- **Agents are never shown as avatars.** Every agent kind is treated the same; its name appears in hovers, in comment authorship ("Claude Code on behalf of …"), and as the robot glyph on the "Worked by" filter.
- **No mailmap or identity merging.** `nicbet@gmail.com` and `nicbet@kuy.io` are intentionally separate (private vs. work) and stay as two people.
- **`created_by` keeps its raw format.** The version and hostname inside it are harmless metadata now that it's never used as an identity for display. No change to the MCP identity format.
- **The assignee is always shown as the person** (decided at tophat). A robot avatar or "Principal via Claude Code" label read as awkward; the agent is an implementation detail of how the person got the work done.
- **An agent assigning a *different* human sets no via.** `assignee_via` means "an agent is doing this work for this person", not "an agent clicked the assign button".

## Edge Cases

- **HIGH:** `start` on an issue assigned to another human: don't reassign, return the notice "Assigned to X, left unchanged". `start --force` (takeover) does reassign to the starter. Decided with the user.
- **MEDIUM:** `XPO_AGENT_IDENTITY` set to a non-`agent@` email: it isn't detected as an agent by email. Mitigation: the "human `raw` == `on_behalf_of` and `created_by` ≠ `on_behalf_of`" rule also catches this, because `on_behalf_of` is set on every MCP call.
- **MEDIUM:** Old filter state saved with raw agent assignee values (URL/localStorage) will match nothing after this change. Accept it: the filter shows no results and the user clears it.
- **LOW:** Old `drive` events with no `on_behalf_of` and an agent author fall back to showing the agent string as the assignee.
- **LOW:** Picking an agent identity in the assignee picker by hand: `collectKnownPeople` should exclude `agent@` identities, so the picker lists only people.

## Assumptions

- `agent@` as the email local part reliably marks an agent. True for every string in the current log.
- The server-mode `httpUserOverride` principal path in `mcpserver/server.go` already sets `OnBehalfOf` correctly for remote users.
