# Walkthrough: Artifact list shows the principal

## Problem
xpo-a9560c set the app-wide rule: when an agent acted for a person, show the person's name with a dashed underline and the agent on hover. The artifact list was the one place left out, because it only had `updated_by`. For agent writes, `updated_by` is the raw agent ("claude-code/2.1.295").

## Key insight: the data already existed
Every ARTIFACT event already records `on_behalf_of` (the local transport stamps it on every MCP write). The projection simply dropped it when it built `ArtifactSummary`. That means no new data had to be written. Because issues are rebuilt from the event log on every read, existing artifacts pick up the principal as soon as the new code runs.

## Changes
**Backend**
- `model.ArtifactSummary` gains `OnBehalfOf` (`on_behalf_of,omitempty`). The name matches `model.Comment` and `model.Event`.
- `projection.go`: "created" and "updated" ARTIFACT events set `OnBehalfOf = evt.OnBehalfOf` along with `UpdatedBy`. The value is assigned unconditionally, so a later direct human edit sets it back to "". Attribution always describes the latest write, never a stale agent.
- The field is passed through in `server.ArtifactResponse` (HTTP / web UI) and `jsonio.ArtifactEntry` (CLI `--json`, MCP output). `omitempty` keeps human-written entries unchanged on the wire.
- Remote transport: untouched. Artifacts are local-only (`ListArtifacts` returns `ErrLocalOnly`, and `apiIssue` doesn't map artifacts).

**Frontend**
- `ArtifactSummary.on_behalf_of?` in `api/types.ts`.
- `ArtifactList.tsx` builds `activityActor(updated_by, on_behalf_of)` and renders `<ActorName name via />` with the existing muted `text-xs` classes. That means principal-first with an underline for agent writes, and a plain name for human writes. Legacy agent writes without a principal fall back to the raw agent, the same fallback comments use.

## Tests
- `TestProjectIssues_ArtifactOnBehalfOf`: an agent create carries (agent, principal), and a later human update clears the via.
- `TestIssueToResponse_ArtifactOnBehalfOf` and `TestToArtifactEntriesOnBehalfOf`: the field reaches HTTP and jsonio output, and is omitted for human writes.
- `format.test.ts`: an `activityActor` legacy-fallback case (agent, no principal), which the artifact mapping relies on.
- All were written first and failed to compile or failed before the change.

## Acceptance criteria
- [x] Agent-written artifacts show the principal with a dashed underline, and the agent on hover and focus. Verified by the user in the UI ("lgtm").
- [x] Human-written artifacts show the plain name
- [x] A direct human update after an agent write clears the via (`TestProjectIssues_ArtifactOnBehalfOf`)
- [x] `on_behalf_of` is present in the HTTP issue response and in jsonio artifact entries
- [x] Go tests for the projection and serialization, and a frontend test for the author mapping
- [x] `make test` passes (553 frontend tests plus Go) and `make frontend` builds
