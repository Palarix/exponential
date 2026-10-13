# Walkthrough: show linked chats on issues

## What was built

xpo-00f0e5 let a chat work on an issue in Work Mode, but nothing on the board showed which issues had a chat. This issue adds those links, and a few pieces of polish that came out of review:

1. **Issue-to-chats links on the board:**
   - a chat pill on backlog rows and board cards;
   - a "k chats" row in issue details;
   - a "Chats" submenu and "Open work chat" in the issue context menu.
2. **Entering Work Mode on a past chat.** A chat that left Work Mode can go back into it from the chats list, conversation intact. Continue Work keeps its xpo-00f0e5 behavior. Leaving Work Mode means "done with that context window", so going back is a deliberate choice.
3. **Cold cache divider.** Each turn records when its prompt cache expires, and the thread shows a dashed "Context cache expired" divider after that time.
4. **One pill style for row indicators** (folded in from xpo-f8df84). Sub-issue progress, branch, chats and attachments share one Linear-style pill. Pills and label chips sit on the surface color.

## How the pieces fit

### Cache expiry (backend)

- **Parsing:** `claudeUsage.CacheTTL()` (`internal/exponential/agent.go`) reads `usage.cache_creation` from Claude Code's `result` event.
  - 1h when `ephemeral_1h_input_tokens > 0`, 5m when `ephemeral_5m_input_tokens > 0`, otherwise 0.
  - We measured this on 2026-10-13: a subscription login writes the 1h cache, so a fixed 5-minute guess would be wrong by 55 minutes.
- **Plumbing:** the TTL travels on `ClaudeTurnResult.CacheTTL` and `TurnResult.CacheTTL`.
- **Codex** reports no TTL, so it records `codexCacheTTL` (5 minutes, the lower bound of OpenAI's documented 5 to 10 minutes).
- **Choosing the TTL:** the runner (`internal/chat/runner.go`) uses `turnCacheTTL(wrote, lastSeconds)`.
  - A turn that only read the cache refreshes the TTL the chat last wrote, so it keeps the chat's last known TTL.
  - The default is 5m.
- **Recording it:** `turn_done` carries `cache_ttl`. The deferred meta update stores `Meta.CacheTTL` and `Meta.CacheExpiresAt` (turn end plus TTL), only for turns that finished; a failed turn leaves them alone.

### Entering Work Mode with an existing chat (backend)

`POST /api/local/issues/{id}/work-chat` takes an optional `{"chat_id"}` (`internal/server/chat.go`).

- **Checks first:** with a chat ID, `checkResumable` runs before the issue starts, so a refusal changes nothing. It refuses:
  - an unknown chat (404);
  - a running chat (409, `ErrBusy`);
  - a chat in Work Mode on another issue (409, the new `ErrWorkingElsewhere`).
- **Held issue:** if another chat holds the issue, the endpoint answers 409 (`ErrIssueHeld`). If the given chat already holds it, it gets that chat back.
- **Bind:** after the usual start or worktree lookup, it calls `BindWork` on the given chat and answers `{chat, created: false, resumed: true}`.
- **Backstop:** `bindLocked` now also refuses a chat bound to a different issue. That would drop the other issue's only writer.

### Issue-to-chats lookup (frontend)

- **No new API.** The chat list the agent provider already polls carries `work` and `worked_issues`.
- **`issue-chats.ts`** holds the pure logic:
  - `chatsForIssue` returns the issue's chats (newest first), the holder (the chat in Work Mode on the issue) and the most urgent state, in the order waiting, running, interrupted, work, idle.
  - `filterChats` is the title search.
  - `enterWorkBlocker` says why a chat can't enter Work Mode.
- **`useIssueChats`** returns nothing when the agent is unavailable, so remote mode shows no chats.
- **Polling:** the provider now polls the chat list while any chat runs (`chatListPollMs`), not only chats with a status-bar pill, so row icons follow running chats.

### Components

- **`IssueChatIcon`** is the chat pill on rows and cards.
  - It opens the holder directly, or else `IssueChatsPopover`.
  - A wrapping span stops click and pointerdown, so neither the row nor board card drag fires.
  - React events from the portaled popover bubble through that span too.
- **`IssueChatsList`** is a `Menu` with a `MenuFilter` and one `ChatRow` per chat.
  - Row actions open inline as an extra menu item under the row, not in a nested popover: menus close on any mousedown outside them, which would drop the action.
  - Actions open with right-click, the `⋯` button (visible on hover or focus) or Shift+F10.
  - In the context menu's submenu flyout the list is 280px wide, the flyout's maximum.
- **`useStartWorkChat(issueId, chatId?)`** passes the chat ID through and picks the draft: "Let's continue with …" for a resumed chat, "Let's implement …" for a new one.
- **Placement:** the issue-details row sits below Relations. The context menu gains "Open work chat" and a "Chats" submenu.

### Row indicators and labels

- **`indicatorPill`** (`ui/indicator-pill.ts`) is the shared class for all of them:
  - fully rounded, 24px tall, at least 32px wide;
  - faint border, surface background, muted 12px icon.
- **Uses:** `BranchBadge`, the new `AttachmentIndicator`, the sub-issue progress pill (with `SubProgress size={12}`) and the chat pill. They sit in one group after the title in `BacklogIssueRow` and `BoardCard`; My Issues picks up `BranchBadge` automatically.
- **Labels:** bordered `LabelBadge` chips get the same surface background. Borderless ones in the label picker don't.

### Chat polish

- **Dividers:** a generic `Divider` in `ChatThread` draws mode dividers ("Entered Work Mode on …", "Entered Chat Mode") and the dashed cold-cache divider. Its tooltip carries the explanation.
- **Timer:** `usePassed` re-renders at the expiry time, so the divider appears live, and the thread scrolls to it.
- **Mode segment:** shows only the hammer and the issue ID, with the accessible name "Work Mode on …".
- **Send and stop buttons:** 24px, with `shrink-0`. At 32px they had been squeezed to 29.75px wide.

## Decisions from review

| Decision | Why |
|---|---|
| Continue Work never reopens a chat that left Work Mode | The user treats Exit work mode as closing that context window. Reopening is explicit, from the chats list. |
| Row actions hide behind right-click or `⋯` | A button on every chat row was visual noise. |
| Monochrome icons; state by shape (`message-square`, `-dot`, `-more`, `-warning`) | Colored icons broke the row style. Circles read too rounded, so we use squares. `-dot` marks waiting for the user. |
| One Linear-style pill for all row indicators, counts on branch and chats | Square bordered chips next to bare icons looked incoherent, and bare icons floated. The pill grounds them. Counts read at a glance and are always shown, also 0 or 1. |
| Pills and labels on the surface color | They stay dark on a hovered or highlighted row. |
| Dashed cold-cache divider | Solid dividers mean mode changes; this one doesn't. |
| Fold xpo-f8df84 into this issue | The pill style only works if the branch badge changes with it. |

## Known gaps

- Opening a chat from a row or the list adds no status-bar pill. That is xpo-b45049.
- The cold-cache divider tracks time only. A model change also empties the cache; it doesn't detect that.
- Keyboard row actions (Shift+F10, the context-menu key) and the disabled "Enter work mode" state were not exercised in the browser. Unit and server tests cover the blocker logic.

## Acceptance Criteria

**Row icon**
- [x] A backlog row and a board card show a chat pill when at least one chat worked on the issue (`worked_issues`). The pill shows the chat count, also when it is 1. The tooltip names it ("2 chats on xpo-…"). *`IssueChats.test.tsx` (counts 1 and 2, aria-label); `BacklogIssueRow.test.tsx`; `BoardCard.test.tsx`. Live: upe-000050 showed 2.*
- [x] The chat pill is a button: it gets a background on hover. *`IssueChats.test.tsx` checks the pill classes and `hover:bg-`.*
- [x] The icon follows the most urgent state among the issue's chats, monochrome, shape tells the state. *`issue-chats.test.ts` (urgency order); `IssueChats.test.tsx` (`message-square-dot`, `-more`, `-warning`, no color classes).*
- [x] With a chat in Work Mode on the issue, clicking the icon opens that chat. Otherwise it opens the chats popover. The click never opens the issue. *Live: the holder opened directly, the popover opened for upe-000050 without a holder, and the issue page stayed closed.*

**Row indicators (folded in from xpo-f8df84)**
- [x] On backlog rows, board cards and My Issues, every row indicator uses one pill style: fully rounded, 24px tall, at least 32px wide, faint border, muted 12px icon. *`RowIndicators.test.tsx`. Live: pills measured 32×24 with 12px icons.*
- [x] The branch pill shows its commit count and the chat pill its chat count, always, also 0 or 1. The attachment pill shows only its icon. Details are in the tooltip. *`RowIndicators.test.tsx` (counts 3 and 0, tooltip text, attachments tooltip); `IssueChats.test.tsx`.*
- [x] The branch pill keeps its dot for uncommitted changes. *`RowIndicators.test.tsx`.*
- [x] Sub-issue progress uses the same pill and keeps its "1/6" count. *`BacklogIssueRow.test.tsx`. Live: "1/6" on upe-000006.*
- [x] Only clickable pills (the chat icon) get a background on hover. *Only `IssueChatIcon` adds `hover:bg-`; checked in review.*
- [x] Pills have the surface background, so they stay dark when the row is hovered or highlighted. *`RowIndicators.test.tsx`; confirmed by the user on the demo board.*
- [x] Bordered label chips get the same surface background, everywhere labels appear. *`RowIndicators.test.tsx` (LabelBadge bordered vs borderless).*

**Issue details**
- [x] Issue details shows a "k chats" button below Relations when the issue has chats. *`PropertySidebar.test.tsx` ("2 chats", "1 chat"). Live on upe-000050.*
- [x] The button opens the chats popover. *Live: the popover listed both chats, with the Work badge.*

**Chats list (popover and submenu)**
- [x] The list shows the issue's chats, newest first, with state icon, title, last activity, and a "Work" badge on the holder. *`IssueChats.test.tsx`. Live.*
- [x] A search field filters the list by title. *`issue-chats.test.ts` (`filterChats`); the search field is in the rendered markup.*
- [x] Clicking a row opens the chat and closes the popover or menu, without changing its mode. *A row click calls `openChat` only, and Menu closes on activate. Live.*
- [x] Rows carry no buttons; actions open with right-click or a `⋯` button shown on hover or focus. *`IssueChats.test.tsx` (actions hidden until opened). Live: `⋯` opened "Enter work mode" inline.*
- [x] With no chat in Work Mode on the issue, a row's actions include "Enter work mode"; it binds the chat, opens it with "Let's continue with xpo-….", and starts a BACKLOG or PLANNED issue first. *`TestChat_WorkChatResumesGivenChat`; `TestChat_WorkChatResumeStartsPlannedIssue`. Live: the chat went back into Work Mode with its history and the draft.*
- [x] "Enter work mode" is disabled on a running chat and on a chat in Work Mode on another issue, with a hint. *`issue-chats.test.ts` (`enterWorkBlocker`); `TestChat_WorkChatResumeRefused`; `TestRunner_BindWorkRefusesChatBoundElsewhere`.*

**Issue context menu**
- [x] On an issue with chats, the context menu has a "Chats" submenu with the chats list. *Live: "Chats 2", and the flyout listed both chats.*
- [x] With a chat in Work Mode on the issue, it also has "Open work chat". *Live.*

**Continue Work**
- [x] Continue Work keeps its xpo-00f0e5 behavior. *The existing work-chat tests pass unchanged (`TestChat_WorkChatStartsIssueAndBindsNewChat`, `TestChat_LeaveWork`, `TestChat_WorkChatContinuesDoingIssue`).*

**Cold cache warning**
- [x] Each finished turn records `cache_expires_at`; Claude's TTL comes from the turn's usage and Codex uses 5 minutes. *`TestClaudeStreamParser_CacheTTL`; `TestRunner_RecordsCacheExpiry`; `TestRunner_CacheExpiryDefaultsToFiveMinutes`; `TestRunner_FailedTurnKeepsCacheExpiry`; codex driver test. Live: a Haiku turn recorded `cache_ttl` 3600.*
- [x] A chat whose cache has expired shows a dashed "Context cache expired" divider with the tooltip, appearing live, and the thread scrolls to it. *`ChatThread.test.tsx` (separator, title, `border-dashed`). Live: it appeared at the expiry time.*
- [x] The marker goes away when a new turn starts. *`ChatThread.test.tsx` (hidden while running).*

**Thread and composer polish**
- [x] Mode dividers read "Entered Work Mode on xpo-…" and "Entered Chat Mode". *`chat-thread-utils.test.ts`; `ChatThread.test.tsx`.*
- [x] In Work Mode, the mode segment shows the hammer and the issue ID only, with accessible name "Work Mode on xpo-…". *`ChatComposer.test.tsx`. Live.*
- [x] The send and stop buttons are 24×24px circles that don't shrink. *`ChatComposer.test.tsx`. Live: measured 24×24.*

**General**
- [x] Nothing shows when the agent is unavailable, including in remote mode. *`IssueChats.test.tsx` (unavailable agent and no provider render nothing); `useIssueChats` returns no chats.*
- [x] Icons and the popover update when a chat enters or leaves Work Mode, starts or ends a turn, or is deleted. *They read the shared chat list. `mode_changed`, `turn_done` and the other meta events, plus the work-chat and leave-work calls, invalidate it, and it polls while any chat runs (`chatListPollMs` test). Live: icons changed after leave-work and Enter work mode.*
- [x] `make test` passes. *807 frontend tests and the Go suite, run before commit.*
