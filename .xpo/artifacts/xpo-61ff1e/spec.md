# Spec: Artifact list shows the principal (artifact on_behalf_of)

## What
The issue-detail artifact list shows `shortName(updated_by)`, which for agent writes is the raw agent ("claude-code/2.1.295"). Under the app-wide rule from xpo-a9560c, it should show the principal with a dashed underline and reveal the agent on hover. Direct writes show the plain name.

## Why
The artifact list is the last place in the web UI that breaks the principal-first attribution rule. The principal is already recorded on each ARTIFACT event's `on_behalf_of`; it just isn't projected or returned.

## How
- `model.ArtifactSummary` gets `OnBehalfOf string \`json:"on_behalf_of,omitempty"\``. The name matches comments (`model.Comment.OnBehalfOf`) and events.
- In the projection (`internal/exponential/projection.go`), "created" and "updated" ARTIFACT events set `OnBehalfOf = evt.OnBehalfOf` along with `UpdatedBy`, overwriting it on every update. That way a later direct edit by a human clears an earlier agent's attribution.
- The field is passed through in `server.ArtifactResponse` (HTTP) and `jsonio.ArtifactEntry` (CLI `--json` / MCP), both with `on_behalf_of,omitempty`.
- Remote transport: no change, because artifacts are local-only (`ListArtifacts` returns `ErrLocalOnly` and `apiIssue` doesn't map artifacts).
- Frontend: `Artifact` in `api/types.ts` gets `on_behalf_of?: string`. `ArtifactList.tsx` derives `activityActor(updated_by, on_behalf_of)` and renders `<ActorName name={actor.name} via={actor.via} className="…existing muted xs classes…" />`.
- Legacy artifacts written by an agent without `on_behalf_of` keep showing the raw agent, the same fallback that comments use.

## Acceptance criteria
- [ ] Agent-written artifacts show the principal with a dashed underline, and the agent is revealed on hover and focus
- [ ] Human-written artifacts show the plain name
- [ ] A direct human update after an agent write clears the via
- [ ] `on_behalf_of` is present in the HTTP issue response and in jsonio artifact entries
- [ ] Go tests: projection (agent create, then human update) and serialization. Frontend test for the artifact author mapping
- [ ] `make test` passes
