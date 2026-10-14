# Walkthrough: live turn layout with tool icons

## The problem

A running agent turn used to show as one collapsed line, "Working for 40s". Expanding it showed a flat list of narration and tool calls, each call written out as a verb ("Searched the code for", or the raw tool name "Bash"). The user couldn't follow what the agent was doing without opening the list, and the open list buried the narration that explained the work.

The issue started as "make the list a timeline with icons". After several tophat rounds it became a new turn layout. The spec records each round under "Decisions".

## What a turn looks like now

```
[user bubble]
I'll load the workflow and read the issue first.      ← narration: a normal message
6 actions · 2 failed                              ›    ← finished group
Now the code. The timeout lives in the handler.       ← narration
[🔍] checkoutTimeout\s*=                     12s  ›    ← active group, text shimmers
```

- **Narration** renders as regular messages, in the same markdown style as the answer. There is no separate "answer" block any more: the text after the last group is just the last message.
- **Each run of tool calls** between two messages is one line:
  - While the turn runs and the group is the turn's latest (the *active* group), the line shows the current call's icon and detail, with the turn's timer on the far right. The text shimmers to show progress.
  - Once the next narration arrives, the line becomes a summary: "6 actions", plus "· 2 failed" when calls failed.
  - Clicking a line expands the group's calls as a timeline on a rail.
- **Thinking** shows a Lightbulb, the word "Thinking" (shimmering) and the timer. It appears while the turn runs, no group is active, and no narration is streaming.
- **An approval wait** replaces the active line's content with an amber dot and "Waiting for your approval". The card below shows the action.
- **The finished turn's total time** ("1m 24s") sits on the far right of the row under the last message, next to the copy button. The "Worked for …" header is gone.

## How the pieces fit

### `chat-thread-utils.ts`: data, no React

- `turnSegments(turn)` replaces `splitTurn`. It walks `turn.steps` once and emits, in order, text segments (consecutive text steps joined with a blank line) and tool groups (consecutive tool steps). Everything the turn renders comes from this list.
- `groupSummary(steps)` returns "k action(s)" and "n failed". It counts failures with `isFailed`, so a call the *user* denied on a card is not counted as a failure.
- `timelineSteps(steps, active)` drops the active group's latest call from its expanded list, because the line already shows it (or the approval card does). Without this, the running call appeared twice, once with a shimmer. An active group whose only call is the current one has nothing to expand, so its line has no chevron.
- `formatDuration(seconds)` formats durations as "12s", "1m" or "1m 24s".
- Each `TOOL_VERBS` entry gained an `icon: ToolIconId`, a string ID so the utils stay free of React. `stepIcon(step)` returns the icon, the verb (the label for its tooltip and screen readers), the detail, and the tool name for unknown tools.
- Removed: `splitTurn`, `currentStep`, `collapsedFailureNote`. A turn with failures no longer auto-expands. In real use agents fail and recover often, so a failure isn't a reason to push the live view around.

### `ToolIcon.tsx`

`ToolIcon.tsx` maps each `ToolIconId` to a lucide icon. The icon sits inside `<span role="img" aria-label={verb} title={verb}>`, so the verb survives as the tooltip and accessible name. Icons are grouped by kind of action rather than one per tool: Eye for looking at something known, Pen for changes, Search for finding things. A few xpo tools keep their own icons (add, comment, link, unlink, artifact, git_status, git_diff, spec, walkthrough, start, merge). Unknown tools get a Wrench and show their name as text. The spec has the full table.

`MessageCircleText` needed lucide-react ≥ 1.57, so this issue bumps lucide-react from 1.17.0 to 1.57.0. A full dependency upgrade is filed as xpo-b2ca41. That issue also covers the stale `package-lock.json` next to `bun.lock`.

### `ChatThread.tsx`

- `Turn` maps segments to `AnswerText` (messages) or `ToolGroup` (lines), then adds `ThinkingLine`, the approval cards, the status notes, and the footer with the copy button and duration.
- `ToolGroup` owns its own open/closed state, keyed by segment index. Segments only ever get appended, so the index stays stable while a turn streams.
- `ToolTimeline` (exported for tests) draws the rail. Each `<li>` draws its own piece of the rail: from its branch down, or from its top. The first row starts at its branch and the last row stops at its branch (`└─`). No row draws a gap, so the rail stays continuous at any row height. `RAIL_Y` (11px) is the middle of a row's first text line.
- `ToolRow`: a failed call (or one denied by the agent's rules) has only a red icon. Its detail and its "failed: …" reason use normal muted text, and the full message still opens in the hover card. A running call shows its icon with shimmering text. The old `Spinner` is gone.

### `index.css`: `.text-shimmer`

A gradient (muted → primary → muted) clipped to the text, three times the element's width, with an animated `background-position`. The result is a brighter band that sweeps left to right every 2.4s, like the Claude Code CLI. It deliberately ignores `prefers-reduced-motion`, at the user's request: the shimmer is the only sign the agent is still working.

### Backend: the detail now names the operation

While reviewing the icons, we found that rows for `xpo spec`, `xpo walkthrough` and `xpo artifact` always read "Read the … of" and showed no issue. The detail keys for MCP tools (`mcpToolDetailKeys` in `internal/exponential/claudestream.go`) contained neither `operation` nor `issue_id`.

- `issue_id` joins `mcpToolDetailKeys`, after `id`.
- A new helper, `mcpDetail(get)`, takes a key getter. It returns the first detail key, prefixed with the input's `operation` when there is one (`write xpo-1`). `MCPToolDetail` (used by Codex and its approvals), `ToolDetail` (Claude's final detail and Claude's approvals), and the partial-JSON scan while input streams all go through it. So Claude, Codex and approval cards produce the same string.
- On the frontend, `callVerbs(key, arg)` checks the first word against the tool's `ops` table ("Wrote the spec of", "Added an artifact to", …) and cuts it from the detail. Chats saved before this change have no operation word, so they fall back to the default verbs.

Carrying the operation as a prefix in the existing `arg` string avoids a new event field, which would have meant changes to `StreamEvent`, the runner, the chat event JSON and the reducer. The cost is a small convention that `callVerbs` parses. It only applies to tools with an `ops` table, so another tool whose detail happens to start with "write" is unaffected (covered by a test).

## Tophat

During review, a dev-only page (`web/tophat.html`) rendered sample turns from `chatThreadReducer`, without a backend: a finished turn, a live turn that took a step every 1.5s, and a pending approval. It was removed before merge, as the user asked.

## Acceptance criteria

- [x] Narration renders as regular chat messages, streaming live. Evidence: `turnSegments` tests; ChatThread test "shows narration as messages with a line for each group of calls between them" checks the order message → group → message → group → message.
- [x] Tool calls after a narration collapse into one line: while active, the current call (icon + shimmering detail) and the timer on the far right, with no spinner; once the next narration arrives, "k actions" (+ "· n failed"). Evidence: "shows the current call of the active group with the timer" (icon label, detail, timer, `text-shimmer`, no `animate-spin`); `groupSummary` tests; reviewed in tophat screenshots.
- [x] Each group line expands into a timeline of its calls: a continuous rail with branch entries. Evidence: `ToolTimeline` tests (`data-step="tool"` per call); `timelineSteps` tests; tophat screenshot of an expanded group.
- [x] "Thinking" (Lightbulb) with the timer shows while the turn runs and no group is active, unless narration streams. Evidence: "shows Thinking while no group is active and nothing streams" covers thinking, a fresh turn, streaming narration and an active group.
- [x] Waiting for an approval shows on the active line; the card shows the action. Evidence: "shows the approval wait on the active line"; tophat screenshot of the waiting panel.
- [x] Failures do not auto-expand; the group line shows the count. Evidence: "keeps a group with failures collapsed and counts them" (`aria-expanded="false"`, "1 failed", no reason text rendered).
- [x] A finished turn shows its total time on the far right under the last message. Evidence: "shows the total time of a finished turn under its last message" (`1m 24s` after the last message); `formatDuration` tests.
- [x] Tool calls show an icon in place of the verb for all known tools; each icon has a tooltip and accessible label with the verb; unknown tools show a fallback icon plus the tool name. Evidence: `stepIcon` test over every known tool; `ToolTimeline` test checks `role="img" aria-label="Read" title="Read"`, that no verb is rendered as text, and the WebFetch fallback.
- [x] Running, failed, denied, and approval states stay visible in the timeline. Evidence: tests "shows a running call with its icon and shimmering text", "marks a failed call with a red icon only", and the reason text "failed: no branch"; approval badges unchanged in `ToolRow`.
- [x] spec, walkthrough and artifact rows name the operation ("Wrote the spec of xpo-1"), with the issue as detail. Evidence: Go `TestToolDetail_ArtifactToolsNameOperationAndIssue` (final and partial detail, `MCPToolDetail`); vitest "names the operation of the spec, walkthrough and artifact tools" and "keeps the old wording for chats saved without an operation".
- [x] Tests updated (`chat-thread-utils.test.ts`, ChatThread tests). Evidence: both files updated, plus `claudestream_test.go`; `make test`, `bunx tsc -b` and `bun run build` pass.
