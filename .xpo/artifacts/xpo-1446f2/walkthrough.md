# Walkthrough: All chats page with search, archive, and token/cost summaries

## What was built

The Agent view gets an "All chats" page at `#/agent/history`. It lists every chat in a full-width table. You can search, filter, archive, unarchive, rename and delete chats there, one at a time or in bulk. A summary strip totals the tokens and cost of the rows the filters show.

An archived chat leaves every active list and becomes read only. The board no longer deletes old chats at startup. It archives chats that are idle for 30 days.

## How the pieces fit together

### Backend (`internal/chat`, `internal/server`)

1. **New `Meta` fields** (`event.go`):
   - `ArchivedAt` is set while a chat is archived.
   - `Usage` holds `Turns`, `TokensIn`, `TokensOut`, `CostUSD` and `CostKnown`.
2. **Totals.** `Runner.endTurn` adds each `turn_done` event to `Usage`.
   - `Store.Create` starts a chat with zero totals.
   - A meta saved before this change has no `Usage`. `Store.Meta` sums its `turn_done` events once, through `backfillUsage`, and saves the result.
   - `endTurn` calls `Store.Meta` before it logs `turn_done`. Otherwise the backfill would count that turn, and so would the update.
3. **Archive** (`archive.go`):
   - `Runner.Archive` refuses with `ErrBusy` while a turn runs, and with `ErrArchiveInWork` in Work Mode.
   - It checks and saves under the runner lock. `Send` registers its turn first and then reads the meta again, so an archive and a send cannot both win.
   - After it saves, it stops the chat's agent, because background tasks can start a turn on their own.
   - `Runner.Unarchive` clears the flag.
4. **Read only.** `Send` and `BindWork` return `ErrArchived`.
   - The work-chat entry (`checkResumable`) refuses an archived `chat_id` before the issue starts.
5. **Endpoints.** `POST /api/local/chats/{id}/archive` and `/unarchive`.
   - `respondChatError` maps `ErrArchived` and `ErrArchiveInWork` to 409.
6. **Auto-archive.** `Store.AutoArchive` replaces `Store.Prune`.
   - It archives chats last updated more than `AutoArchiveAfter` (30 days) ago.
   - It skips running chats and chats in Work Mode, and keeps `updated_at`.
   - `board.go` calls it at startup. Nothing deletes chats on its own anymore.
7. **Titles.** `maxTitleRunes` goes from 60 to 200, so `TitleFrom` keeps more of the first message. Every list truncates titles with CSS.

### Frontend (`web/src`)

1. **Route.** `Route.chatHistory` (`views.ts`) holds the history page and its optional `?issue=` filter.
   - `AppNav.openChatHistory(issueId?)` opens the page.
   - `routeTitle` gives the browser tab "All Chats".
2. **Active and all chats.** `AgentChatProvider` exposes two lists:
   - `chats`, active only: pills, recent lists, the switcher and the tab status read it.
   - `allChats`: the history page, `prefsFor`, the open chat's meta and `chatsForIssue` read it.
   - Pills of archived chats drop out because `visiblePills` filters on the active list.
3. **The page** is `ChatHistory.tsx`. Its pure helpers are in `chat-history-utils.ts`: `filterHistory`, `summarize`, `formatCost`, `formatTokens` and `archiveBlocker`.
   - A `TopBar` holds the title, the search in the center, and New chat.
   - The filters row has the agent and period dropdowns and an issue chip. The Active/Archived tabs sit at the far right.
   - The summary strip shows chats, tokens in, tokens out and cost. A cost total ends in "+" when some chats report no cost.
   - The bulk bar always keeps its space, `invisible` and `aria-hidden` without a selection, so the table never moves.
   - Each row has a state slot, the title, the agent (hover shows a usage card), issue chips (`IssueRef`), last activity, and a ⋯ menu.
4. **`SelectSlot`.** The table has no checkbox column. Each row's checkbox is stacked on its state icon.
   - The checkbox takes the icon's place on row hover, on keyboard focus, and while any row is selected.
   - It is the `peer`, and the icon hides on `peer-focus-visible`. A checkbox focused by a mouse click therefore does not leave the slot empty after unselecting, and the rename input does not hide the icon.
   - An idle chat shows a muted ring, so every row has a marker.
5. **Archive everywhere else.**
   - `ChatTitleBar` gets Archive or Unarchive. A disabled Archive shows its reason as a menu line.
   - `ArchivedBanner` replaces the composer in the Agent view and the popup.
   - `IssueChatsList` adds an "N archived chats" item that opens the history page filtered by the issue.
   - The status-bar history menu ends with "All chats".
   - `useArchiveChat` wraps archive, unarchive and delete for one chat or many, with toasts.
6. **Refresh after a failed send.** `useChatSession` refreshes the chat list when a send fails. A chat archived in another tab then shows its banner.

## Key decisions

- **Auto-archive, not prune.** Old chats stay on disk, and only the user deletes chats. `.xpo/chats/` now grows without limit. Bulk Delete is the way to clear it.
- **Archive lives on the runner.** Only the runner knows whether a turn runs.
- **Totals in `Meta`.** The list endpoint reads only meta files, so the table and the summary never read event logs.
- **Filtering in the browser.** The list endpoint already returns every chat. Revisit this if the list gets slow.
- **Usage in a hover card, not columns** (review). Three number columns made the table noisy.
- **Checkbox swaps with the state icon** (review). A checkbox column left an empty gutter when nothing was selected.
- **Accent-colored checkboxes** (review). They match the task-list checkboxes. MergeView's "Delete branch" checkbox gets the accent too, so no checkbox shows the browser's default blue.

## Non-obvious details

- The page has its own `TooltipPrimitive.Provider`. The app's provider is in `main.tsx`, so static-render tests would throw without one.
- Chat titles saved before this change keep their 60-character cut. A one-off script rewrote the local automatic titles; renamed titles stayed as they were. Better titles are follow-up xpo-714245. Headless Claude sessions write no `ai-title` record, so we have nothing to read yet.
- Board processes started before this change run the old backend. They return no `usage` and have no archive endpoints, so restart the board after the merge.

## Acceptance criteria

- [x] The Agent view header shows an "All chats" link that opens `#/agent/history`. Evidence: `AgentView.test.tsx` "links to All chats from the header", `views.test.ts` route tests.
- [x] The status-bar history menu ends with an "All chats" item. Evidence: `AgentStatusBar.tsx`; tested in the browser.
- [x] The page is full width, with a stable scrollbar gutter. Evidence: `AgentView.test.tsx` (no `max-w-6xl`), `scroll-stable` utility; tested in the browser.
- [x] The browser tab reads "All Chats". Evidence: `views.test.ts` "routeTitle".
- [x] The chat title fills its column and truncates there. Evidence: tested in the browser.
- [x] Titles keep up to 200 characters. Evidence: `store_test.go` `TestTitleFrom`.
- [x] Linked issues show as `IssueRef` chips. Evidence: `ChatHistory.tsx`; tested in the browser.
- [x] Active chats are listed newest first, with title, agent and model, issues, and last activity. Evidence: `AgentView.test.tsx` "shows the chat history page", `chat-history-utils.test.ts` `filterHistory`.
- [x] State icons, with a muted ring for idle chats in the same slot. Evidence: `AgentView.test.tsx` Idle ring assertion; tested in the browser.
- [x] Hovering the agent shows turns, tokens and cost. Evidence: `UsageCard` in `ChatHistory.tsx`; tested in the browser.
- [x] A chat with no cost shows "—". Evidence: `chat-history-utils.test.ts` `formatCost`.
- [x] The search is in the center of the top bar, matching titles case-insensitively. Evidence: `chat-history-utils.test.ts` "matches titles case-insensitively"; tested in the browser.
- [x] The Active/Archived tabs sit far right. Evidence: `ChatHistory.tsx` (`ml-auto`); tested in the browser.
- [x] The agent filter works. Evidence: `chat-history-utils.test.ts` "limits rows to one agent".
- [x] The issue filter and `?issue=` work. Evidence: `chat-history-utils.test.ts`, `views.test.ts` "parses the chat history page, with an issue filter".
- [x] The summary strip shows count, tokens and cost. Evidence: `chat-history-utils.test.ts` `summarize`.
- [x] The period picker limits rows and summary. Evidence: `chat-history-utils.test.ts` "limits rows to chats active in the period".
- [x] A click on a row opens the chat. Evidence: `ChatHistory.tsx` row `onClick`; tested in the browser.
- [x] The row ⋯ menu has Rename, Archive or Unarchive, and Delete. Evidence: `RowMenu` in `ChatHistory.tsx`; tested in the browser.
- [x] Checkboxes swap with the state icon. Evidence: `AgentView.test.tsx` `peer` and `peer-focus-visible` assertions; tested in the browser, including the unselect fix.
- [x] Select-all is in the Chat header slot. Evidence: `AgentView.test.tsx` select-all assertion.
- [x] Checkboxes use the accent color. Evidence: `AgentView.test.tsx` accent assertions.
- [x] MergeView's checkbox uses the accent color. Evidence: `MergeView.tsx`.
- [x] Checkboxes center in the slot. Evidence: `SelectSlot` (`absolute inset-0` in a flex-centered slot); tested in the browser.
- [x] Bulk Archive, Unarchive and Delete, with confirmation for Delete. Evidence: `ChatHistory.tsx` bulk bar and `ConfirmDialog`; tested in the browser.
- [x] The bar ends with "Clear selection", flush right. Evidence: `AgentView.test.tsx` `pl-3 pr-0` assertion.
- [x] The bulk bar keeps its space. Evidence: `AgentView.test.tsx` `invisible` / `aria-hidden` assertion.
- [x] A disabled Archive says why. Evidence: `chat-history-utils.test.ts` `archiveBlocker`.
- [x] Archive returns 409 for a running chat and for one in Work Mode. Evidence: `chat_archive_test.go` `TestChat_ArchiveRunningChatConflicts`, `TestChat_ArchiveWorkModeChatConflicts`.
- [x] Archived chats leave the active lists. Evidence: `chat-history-utils.test.ts` `activeChats`; the provider splits `chats` and `allChats`.
- [x] Archiving drops the chat's pill. Evidence: `visiblePills` filters on active chats.
- [x] The issue chat list hides archived chats and links "N archived chats". Evidence: `issue-chats.test.ts`, `IssueChats.test.tsx`.
- [x] An archived chat opens read only, with the banner. Evidence: `AgentView.test.tsx` "shows an archived chat read only".
- [x] Sending to an archived chat returns 409. Evidence: `chat_archive_test.go` `TestChat_ArchiveAndUnarchive`, `archive_test.go` `TestRunner_ArchiveAndUnarchive`.
- [x] Unarchive restores the chat. Evidence: the same tests send after unarchive.
- [x] Startup auto-archives chats idle for 30 days, skipping running and Work Mode chats. Evidence: `archive_test.go` `TestStore_AutoArchive`.
- [x] The board never deletes a chat by itself. Evidence: `Store.Prune` is gone; `TestStore_AutoArchive` checks that all 5 chats remain.
- [x] `make test` passes: Go tests, `go vet`, eslint, and 1081 frontend tests.
