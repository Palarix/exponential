# Walkthrough: comments carry `on_behalf_of`

## The bug
When an agent writes a comment for someone, the COMMENT event stores both identities: `created_by` is the agent (`claude-code/2.1.263 <agent@mcp>`), and `on_behalf_of` is the person it acted for. xpo-35fc16 introduced the display rule for this: show the principal's avatar and name the agent up front, as "Claude Code on behalf of Nicolas Bettenburg". The issue activity timeline follows that rule because it reads the raw event history.

Every other view reads the **projected** `issue.comments`. The projection copied `created_by` but dropped `on_behalf_of`, so the merge view's Conversation tab, `xpo comments` and the TUI details pane had nothing to apply the rule to. They showed the raw agent identity, and in the web views a robot avatar.

## The fix: carry the field through every hop, then share one display rule

### Backend
`on_behalf_of` is added with the same `json:"on_behalf_of,omitempty"` tag that `model.Event` uses, at each place a comment is copied:

| Hop | File |
|---|---|
| Model | `internal/model/types.go`, `Comment.OnBehalfOf` |
| Projection (events → issue) | `internal/exponential/projection.go`, from `evt.OnBehalfOf` |
| HTTP response | `internal/server/responses.go`, `CommentResponse` and `issueToResponse` |
| Remote/proxy mode | `internal/exponential/remote_transport.go`, `apiComment` and `apiIssueToModel` |
| CLI `--json` / MCP `show` | `internal/jsonio/output.go`, `CommentSummary` and `ToCommentSummaries` |

The remote hop is easy to miss. `xpo` in remote mode rebuilds `model.Issue` from the HTTP JSON, so without it the field would be dropped again on the client.

Because `omitempty` is set, human comments' JSON is unchanged.

Comments written **before** this change need no migration. Their events already stored `on_behalf_of`, and the projection is rebuilt from events, so old agent comments render correctly too.

### Web
`commentAuthor(createdBy, onBehalfOf?)` in `utils/format.ts` builds on the existing `activityActor` and returns:
- `identity`: the principal when present, otherwise `created_by`. It keeps the `<email>` so `Avatar` can find a gravatar.
- `name`: `agentCommentAuthor(via, principal)` for agent comments, otherwise the short name.

MergeView uses it for each comment. ActivityTimeline now uses it too, instead of its own inline `entry.via ? agentCommentAuthor(…) : entry.author`, so both views follow one rule. The timeline's comment entry type also loses its now-unused `via` field. Legacy agent comments with no principal fall back to the raw author, the same as before.

### Terminal
`xpo comments` (through a small `commentAuthor(model.Comment)` helper, so it can be tested) and the TUI details pane now use `identity.Actor`. That's the terminal's existing principal-first format: "Nicolas (via Claude Code)", the same one `FormatActor` uses in the timeline. As a side effect, the TUI pane shows the short name instead of the full `Name <email>`, the same as the CLI already did. The user agreed to include the CLI and TUI in this issue.

## Acceptance criteria
- [x] Agent-written comments in the merge view show the principal's avatar and "<Agent> on behalf of <Principal>". Evidence: MergeView renders `commentAuthor(...)`, covered by the `commentAuthor` tests in `format.test.ts`. The user checked it in the browser and approved ("lgtm").
- [x] Human and legacy agent comments render as before. Evidence: the `commentAuthor` tests "human comment by its author's short name" and "falls back to the raw author for legacy agent comments".
- [x] `on_behalf_of` is carried through the HTTP response, remote transport and jsonio. Evidence: `TestProjection_CommentCarriesOnBehalfOf`, `TestIssueToResponse_CommentOnBehalfOf`, `TestApiIssueToModel_CommentOnBehalfOf` and `TestToCommentSummariesOnBehalfOf`, which also checks `omitempty` for human comments.
- [x] `xpo comments` and the TUI details pane are principal-first. Evidence: `TestCommentAuthor` (cmd) and `TestRenderIssueDetails_AgentCommentIsPrincipalFirst`.
- [x] `commentAuthor` tests were written first and failed with "commentAuthor is not a function" before the implementation.
- [x] `make test` passes: lint, 536 vitest tests and the Go suite.
