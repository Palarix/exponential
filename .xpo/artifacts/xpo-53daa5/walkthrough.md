# Walkthrough: The demo data has agents assigned to issues with no person behind them

## What changed and why
In xpo an agent always works on behalf of a person: the issue's `assignee` is the person and `assignee_via` is the agent. The `xpo demo` generator treated its agents (Claude, Devin, Whiskers) as ordinary personas. Any seed issue whose actor was an agent was assigned to that agent, and agent events had no `on_behalf_of`. The demo therefore showed a model the product doesn't support, and never exercised the principal/"via" display from xpo-a103c3 and xpo-a9560c.

## How it fits together (`internal/demo/demo.go`)
- **`persona.Principal`:** each agent persona names the human it works for (Claude → Sarah, Devin → Jake, Whiskers → Luna). Humans have `-1`. `isAgent()` is simply `Principal >= 0`.
- **Issue owner:** `principalFor(actor)` returns the actor for a human, or the agent's principal for an agent. `Generate` records one owner per seed ref in `principalMap`, next to `idMap`.
- **CREATE payload:** `Assignee` is always the owner's identity.
- **`attribute(evt, actor, principal)`:** the single place that sets authorship. It always sets `CreatedBy` to the actor and, for an agent, adds `OnBehalfOf` = the issue owner. CREATE, status, comment and link events all go through it. Timeline events use the *issue's* owner as principal, not the agent's default, so all agent work on one issue is attributed to the person who owns it.
- **Contributors:** `writeConfig` skips agents, so `config.yaml` lists only Sarah, Jake and Luna.

No new projection logic was needed. With `Assignee` = person, `CreatedBy` = agent and `OnBehalfOf` = person, the existing `identity.ResolveAssignment` already yields `assignee` = person and `assignee_via` = agent. That is the same path real MCP-driven work takes.

## Decisions
- **Fixed agent → principal mapping** instead of choosing a person per issue in `seed.go` (confirmed with the user). It is deterministic, leaves the 1,400-line seed file untouched, and still spreads agent work across all three humans.
- **Agents keep their demo identities** (`Claude <agent@claude.ai>` …) rather than the real `claude-code/x.y <agent@mcp>` form (confirmed with the user). `identity.IsAgent` keys on the `agent@` local part, so they are still recognised as agents.

## Acceptance criteria
- [x] No demo issue has an agent identity as its `Assignee`. Evidence: `TestGenerateAttributesAgentWorkToHumans` projects the generated events with `exponential.ProjectIssues` and asserts no assignee is an agent.
- [x] Agent-worked demo issues show a human assignee with the agent as "via". Evidence: the same test asserts that `assignee_via` occurs and is always an agent identity.
- [x] Every agent-authored demo event has a human `OnBehalfOf`. Evidence: the same test checks every event whose `CreatedBy` is an agent.
- [x] `config.yaml` contributors contain no agents. Evidence: the same test asserts no `agent@` appears in the generated config.
- [x] A test in `internal/demo` covers the above. Evidence: the test failed against the old generator, passes now, and `make test` passes.
