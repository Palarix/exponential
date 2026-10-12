# Walkthrough: tool calls in the chat work list

## Problem

The "Worked for N seconds" list in agent chat had three defects:

1. **Broken arguments.** A Grep pattern like `"estimate":\s*\{` showed as `\\{\`.
2. **Raw commands.** Rows read as raw tool names and arguments, such as `Grep <regex>` and `xpo list`.
3. **Misaligned bullets.** In the popup, the bullets did not line up. Also, `xpo add` rows showed the issue description instead of its title.

The code came from xpo-56ae3a. The fix shipped on the xpo-c047ce branch.

## Fixes

### Argument decoding (`internal/exponential/claudestream.go`)

Tool input arrives as partial JSON chunks. `extractJSONStringValue` reads a value from that partial buffer by string matching.

- **Before:** it stopped at the first `"`, even an escaped one, and returned the raw escaped text.
- **Now:**
  - It scans for the first quote that is not escaped.
  - It drops a dangling backslash at the end of a chunk.
  - It decodes the value with `json.Unmarshal`. If an escape is incomplete, such as half of a `\u` sequence, it returns the raw text until the next chunk arrives.
- When the tool block ends, the input is complete. `finalToolDetail` then decodes it as real JSON, so the stored `tool_call.arg` is always exact.

Drive's terminal spinner uses the same parser, so it benefits too.

### Which argument is shown

- Built-in tools use `toolDetailKeys`, which checks `description` early. That suits Bash and Task, where `description` is a one-line summary.
- MCP tools (`mcp__*`) now use `mcpToolDetailKeys`: `id`, `title`, `query`, `match`, …. For xpo's tools, `description` is the issue body.
- `id` comes before `title`, so `update` shows the issue it changed.

### Readable summaries (`web/src/components/Agent/chat-thread-utils.ts`)

`toolSummary(name, arg)` maps known tools to a verb plus a detail. Examples:

- "Read board.go" (base name only)
- "Searched the code for …"
- "Found files matching …"
- "Listed issues matching …", or "Listed issues" when there is no filter
- "Opened xpo-…"
- "Created issue …"
- "Updated xpo-…"

Unknown MCP tools read as "server tool". Other tools keep their name. `ChatThread` shows search patterns in monospace, and hovering a row shows the full argument.

### Alignment (`ChatThread.tsx`)

In the 420px popup, a row with a long detail is wider than its column. Flexbox then shrinks all shrinkable items in proportion to their size. The 12px bullet slot had no `shrink-0`, so it shrank by a different amount on each row.

The slot is now `w-3 shrink-0`. Only the detail (`min-w-0 truncate`) gives way.

## Acceptance criteria

- [x] A pattern with escaped quotes is stored and shown intact. *Evidence: a real turn stored `"title":\s*\{`, and screenshot `12-tools` shows it. Tests: `TestExtractJSONStringValue_Escapes` and `TestClaudeStreamParser_ToolDoneDecodesFullInput`.*
- [x] Tool calls read as readable summaries. *Evidence: screenshot `12-tools` reads "Searched the code for … · Listed issues matching login · Opened upe-0000c7". Tests: the `toolSummary` tests.*
- [x] `xpo add` shows the title, `update` shows the ID, and Bash keeps its description. *Evidence: `TestToolDetail_MCPAddShowsTitleNotDescription`, `TestToolDetail_PrefersIDOverTitle`, and `TestToolDetail_BashKeepsDescription`.*
- [x] Bullets line up in the popup. *Evidence: a Playwright measurement put all seven bullets at x=1002, and screenshot `14-popup-align` confirms it.*

## Things to know

Chats saved before this fix keep their old `arg` values, because the arguments were stored that way.
