# Spec: Demo data attributes agent work to a human principal

## What
`xpo demo` produces data in which every agent action has a human principal. Agent-worked issues are assigned to a person, with the agent recorded as `assignee_via`. Agent-authored events carry `on_behalf_of`.

## Why
In xpo an agent always works for a person. Today the demo seed sets the agent as the assignee on 543 of the 878 seeded actor entries, with no principal. As a result the demo never exercises the principal and "via" display (xpo-a103c3, xpo-a9560c), and it shows a model the product doesn't support.

## How
All changes are in `internal/demo/demo.go`; `seed.go` stays unchanged.

1. **Principal per issue.** Each persona gets a `Principal` index; humans have none (-1). The defaults are Claude → Sarah, Devin → Jake, Whiskers → Luna. For each seed issue, `principal = actor` when the actor is human, and `personas[actor].Principal` when the actor is an agent.
2. **CREATE events.** `Assignee` = the principal's identity. When the actor is an agent, `CreatedBy` = the agent and `OnBehalfOf` = the principal. Replay then resolves `Assignee` = principal and `AssigneeVia` = agent through `identity.ResolveAssignment`, the same path real agent events take.
3. **Timeline events** (status, comment, link). When `ev.Actor` is an agent, `CreatedBy` = the agent and `OnBehalfOf` = the principal of the issue the event belongs to. Using the issue's principal keeps every agent event on an issue attributed to the person who owns it. Human events are unchanged.
4. **Contributors.** `config.yaml` lists only the human personas. The web people pickers already filter agent identities (`collectKnownPeople`), so this keeps the config consistent with that filter.

## Acceptance criteria
- [ ] No demo issue has an agent identity as its `Assignee`
- [ ] Agent-worked demo issues show a human assignee with the agent as "via"
- [ ] Every agent-authored demo event has a human `OnBehalfOf`
- [ ] `config.yaml` contributors contain no agents
- [ ] A test in `internal/demo` replays the generated events and asserts all of the above

## Decisions
- **Fixed agent → principal mapping instead of a per-issue choice in `seed.go`.** It is deterministic and needs no edits to the 1,400-line seed file, and it still spreads agent work across all three humans. Revisit if the demo needs one agent working for several people.
- **Agent identities keep their demo names** (`Claude <agent@claude.ai>`, …) rather than `claude-code/x.y <agent@mcp>`. `IsAgent` only checks the `agent@` local part, so they are treated as agents.
