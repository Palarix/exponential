# Merge view comments show the raw agent identity instead of the principal

## What
Agent-written comments in the merge view's Conversation tab show the raw agent identity (`claude-code/2.1.263`) as the author, with a robot avatar. The issue activity timeline shows the same comment as "Claude Code on behalf of Nicolas Bettenburg" with the principal's avatar. The CLI `xpo comments` text output and the TUI details pane print the raw identity too.

## Why
The projection drops the event's `on_behalf_of` when it builds `model.Comment`, so every consumer of `issue.comments` sees only `created_by`. The activity timeline gets it right only because it reads the raw event history (xpo-35fc16 rule 10).

## How
**Backend: carry `on_behalf_of` through every comment hop** (`json:"on_behalf_of,omitempty"`, the same tag `Event` uses):
- `model.Comment` gets `OnBehalfOf string`.
- `projection.go`: `OnBehalfOf: evt.OnBehalfOf` on COMMENT events.
- `server/responses.go`: `CommentResponse.OnBehalfOf`, copied in the issue response mapping.
- `remote_transport.go`: `apiComment.OnBehalfOf`, copied in `apiIssueToModel`, so remote/proxy mode keeps it.
- `jsonio/output.go`: `CommentSummary.OnBehalfOf`, filled in `ToCommentSummaries` (CLI `--json` and MCP `show`).

**Terminal surfaces** (`identity.Actor` / `ui.FormatActor`, "Nicolas Bettenburg (via Claude Code)"):
- `cmd/exponential/comments.go` text output: the author is `FormatActor(created_by, on_behalf_of)` instead of a hand-trimmed `created_by`.
- `internal/ui/details.go` comments section: same.

**Frontend**
- `api/types.ts`: `Comment.on_behalf_of?: string`.
- `utils/format.ts`: new `commentAuthor(createdBy, onBehalfOf?) → { identity, name }`. `identity` is the principal when an agent acted for one, otherwise `created_by`, and it keeps the email for the gravatar. `name` is `agentCommentAuthor(...)` for agent comments, otherwise `shortName(created_by)`. It's built on `activityActor`.
- `MergeView` comment list: `<Avatar name={identity}>` and `name`.
- `ActivityTimeline` comment entries use the same helper, so both views share one rule instead of two inline copies.

## Acceptance criteria
- [ ] Agent-written comments in the merge view show the principal's avatar and the author "<Agent> on behalf of <Principal>".
- [ ] Human comments, and agent comments with no `on_behalf_of` (legacy), render as before.
- [ ] `on_behalf_of` comes back in the HTTP issue response, in remote transport and in jsonio `CommentSummary` (Go tests for projection, the response mapping, the remote round trip and `ToCommentSummaries`).
- [ ] `xpo comments` and the TUI details pane show "Principal (via Agent)" for agent comments (Go tests).
- [ ] `commentAuthor` has unit tests, written first.
- [ ] `make test` passes.

## Decisions (confirmed with user)
1. CLI `xpo comments` text output and the TUI details pane are in scope.
