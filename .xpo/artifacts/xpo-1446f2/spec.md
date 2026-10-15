# Agent chat: All chats page with search, archive, and token/cost summaries

## What

Add an "All chats" page at `#/agent/history`. A link in the Agent view header opens it. The page lists every chat in a table, with search, filters, per-chat token and cost totals, and a summary strip. You can archive, unarchive, rename and delete chats there. An archived chat leaves the active chat lists and becomes read only. The board no longer prunes chats. It archives chats that are idle for 30 days.

## Why

The Agent view home shows 5 recent chats, and the header switcher and status-bar history show 20 (`RECENT_CHAT_LIMIT` in `recent-chats.tsx`). Older chats are hard to find again. The board also deletes chats older than 30 days at startup (`Store.Prune`), so old chats are lost. The chat store already records `cost_usd`, `tokens_in` and `tokens_out` on each `turn_done` event, but no view shows them. This issue extends xpo-c047ce, which added the Agent view and the recent chats lists.

## Acceptance Criteria

- [ ] The Agent view header shows an "All chats" link. The link opens `#/agent/history`.
- [ ] The status-bar history menu ends with an "All chats" item that opens the page.
- [ ] The page fills the full width of the Agent view. Its scroll container keeps a stable scrollbar gutter, so the layout does not shift.
- [ ] The browser tab title reads "All Chats" on the page.
- [ ] The chat title grows to fill its column and truncates there.
- [ ] New chat titles and renames keep up to 200 characters, not 60. Existing titles stay as stored.
- [ ] Linked issues show as issue chips, like issue IDs in Markdown, with the hover preview.
- [ ] The page lists all active chats, newest first. Each row shows the title, agent and model, linked issues, and last activity.
- [ ] A row shows a spinner or a colored dot for a running, waiting, interrupted, failed or unseen chat. An idle chat shows a small muted ring in the same slot, not a message icon, so rows stay quiet next to the checkboxes and titles line up.
- [ ] Hovering the agent cell shows a card with the chat's turns, tokens in/out and cost.
- [ ] A chat with no cost data shows "—" as its cost, not "$0.00".
- [ ] The search box sits in the center of the page's top bar, like other views. It filters rows by title as you type. Matching is case-insensitive.
- [ ] Tabs switch between "Active" and "Archived" chats. They sit at the far right of the filters row.
- [ ] An agent filter limits rows to one provider.
- [ ] An issue filter limits rows to chats linked to one issue. `#/agent/history?issue=<id>` opens the page with this filter set.
- [ ] The summary strip shows the chat count, total tokens in/out and total cost for the filtered rows.
- [ ] A period picker (7 days, 30 days, All) limits the rows and the summary to chats active in that period.
- [ ] A click on a row opens the chat in the Agent view.
- [ ] Each row has a "⋯" menu with Rename, Archive (or Unarchive), and Delete.
- [ ] Checkboxes select many rows. There is no checkbox column: a row's checkbox takes the place of its state icon on row hover, on focus, and while any row is selected.
- [ ] The select-all checkbox sits in the same slot of the "Chat" header, and shows on header hover, on focus, and while any row is selected.
- [ ] Checked checkboxes use the accent color (`--color-accent-primary`), like task-list checkboxes.
- [ ] The "Delete branch" checkbox in `MergeView` uses the accent color too, so no checkbox in the app shows the browser default blue.
- [ ] Checkboxes center in the state icon slot, on their row and on the header text.
- [ ] Bulk Archive (or Unarchive) and bulk Delete act on the selection. Delete asks for confirmation.
- [ ] The bulk actions bar ends with a "Clear selection" button, flush with the bar's right edge.
- [ ] The bulk actions bar has a reserved slot between the summary strip and the table. It is hidden, not removed, when no row is selected, so the table does not move.
- [ ] Archive is disabled for a running chat and for a chat in Work Mode. A line under the menu item says why.
- [ ] `POST /api/local/chats/{id}/archive` returns 409 for a running chat or a chat in Work Mode.
- [ ] An archived chat does not show in the Agent view recent chats, the header switcher, the status-bar history, or the status-bar pills.
- [ ] Archiving a chat that has a status-bar pill removes the pill.
- [ ] An issue's chat list hides archived chats. It shows "N archived chats", which links to `#/agent/history?issue=<id>`.
- [ ] An archived chat opens read only, in the Agent view and in the popup. A banner "This chat is archived" with an Unarchive button replaces the composer.
- [ ] `POST /api/local/chats/{id}/messages` returns 409 for an archived chat.
- [ ] Unarchive restores the chat to the active lists. The chat accepts messages again.
- [ ] At startup, the board archives chats last updated more than 30 days ago. It skips running chats and chats in Work Mode.
- [ ] The board never deletes a chat by itself. Only the user deletes chats.
- [ ] `make test` passes.

## Flow

### Backend

1. Add `ArchivedAt *time.Time` (`archived_at`) to `Meta` in `internal/chat/event.go`.
2. Add `Usage *Usage` to `Meta`, with `Turns`, `TokensIn`, `TokensOut`, `CostUSD` and `CostKnown`. `Store.Create` sets zero totals.
3. Add the `turn_done` event to the totals in `Runner.endTurn`. Set `CostKnown` when the event has `cost_usd > 0`.
4. Add `Runner.Archive(id)` and `Runner.Unarchive(id)` in `internal/chat/archive.go`.
5. Make `Runner.Archive` return `ErrBusy` while a turn runs and `ErrArchiveInWork` in Work Mode. It checks under the runner lock.
6. Make `Runner.Archive` stop the chat's agent, so no background task starts a turn.
7. Make `Runner.Send` and `Runner.BindWork` return `ErrArchived` for an archived chat.
8. Add `POST /api/local/chats/{id}/archive` and `POST /api/local/chats/{id}/unarchive` in `internal/server/chat.go`. Map `ErrArchived` and `ErrArchiveInWork` to 409.
9. Make the work-chat entry refuse an archived `chat_id` before the issue starts.
10. Backfill the totals: when `Store.Meta` reads a meta without totals, sum the `turn_done` events once and save the meta.
11. Replace `Store.Prune` with `Store.AutoArchive(maxAge, now)`. It archives idle chats last updated before `now-maxAge`, keeps `updated_at`, and returns the count.
12. Replace `PruneMaxAge` and `PruneMaxCount` with `AutoArchiveAfter = 30 * 24 * time.Hour`.
13. Call `AutoArchive` in `cmd/exponential/board.go` where `Prune` ran. Log the count.

### Frontend

14. Add `archiveChat` and `unarchiveChat` to `web/src/api/chat.ts`. Add `archived_at` and `usage` to `ChatMeta`.
15. In `AgentChatProvider`, expose `chats` (active only) and `allChats`. Pills, recent lists and the switcher use `chats`.
16. Look up an open chat's meta in `allChats`, so an archived chat still opens.
17. In `chatsForIssue`, leave archived chats out of the list and state, and count them in `archived`.
18. In `IssueChatsList`, add the "N archived chats" item.
19. Add `chatHistory` to the route and `openChatHistory(issueId?)` to `AppNav`.
20. Add `web/src/components/Agent/ChatHistory.tsx` for the page, with helpers in `chat-history-utils.ts`.
21. Add the "All chats" link to the Agent view header, and the "All chats" item to the status-bar history menu.
22. Add Archive or Unarchive to the "⋯" chat menu in `ChatTitleBar`.
23. Add `ArchivedBanner`. The Agent view and the popup show it in place of the composer.
24. Add tests: archive rules, auto-archive, server 409s, totals, the backfill, routes, history filters and summary, `IssueChats` archived count, and the banner.

## Decisions

- **Auto-archive, not prune.** The board archives chats that are idle for 30 days and never deletes a chat. The 200-chat cap goes away. You delete chats by hand on the history page. This replaces `Store.Prune` from xpo-c047ce.
- **Archive lives on the runner, not the store.** Only the runner knows whether a turn runs. `Send` registers its turn first and then reads the meta again, so archive and send cannot both win.
- **Archive stops the chat's agent.** Background tasks can start a turn on their own. An archived chat must not run.
- **Totals live in `Meta`, not computed per request.** The list endpoint reads only meta files. Summing the event logs of every chat on each list request is slow. A separate usage index file adds a second source of truth.
- **Search matches titles only.** Searching message text needs the event logs and an index. That can come later.
- **Search, filters and period run in the browser.** The list endpoint already returns every chat. Revisit if the chat count makes the list slow.
- **The period filters on `updated_at`.** A chat counts in a period when it had activity in it. Its full totals count, not only the turns inside the period.
- **Archive is a flag, not a move.** The chat files stay in `.xpo/chats/`. Unarchive only clears the flag.
- **"Work Mode" means `Mode == ModeWork`.** A chat that left Work Mode can be archived, even if `WorkedIssues` is not empty.
- **The reason for a disabled Archive is a menu line, not a tooltip.** `MenuItem` has no tooltip. `IssueChatsList` shows "Enter work mode" blockers the same way.
- **The page is a full-width table, not cards.** Rows scan down the page like the issue list.
- **Usage per chat is a hover card, not columns.** Three number columns made the table noisy. The summary strip keeps the totals.
- **Titles keep 200 characters.** Every list truncates titles with CSS, so the 60-character cap only hid text on wide rows.
- **The checkbox swaps with the state icon, not a column of its own.** A separate column left an empty gutter when nothing was selected. While selecting, the state icons give way to checkboxes, like Linear and Gmail.

## Edge Cases

- **HIGH — Chats now accumulate without limit.** The board stops deleting chats, so `.xpo/chats/` grows. The history page shows the total, and bulk Delete clears old chats.
- **MEDIUM — A chat starts running while the archive request is in flight.** `Runner.Archive` checks the turn under the runner lock and returns 409. The page shows the error as a toast.
- **MEDIUM — An archived chat is open in another tab.** The next send returns 409. The view then reloads the chat list and shows the banner.
- **MEDIUM — Codex chats report no cost.** `CostKnown` stays false, and the page shows "—". The summary strip sums only known costs and marks the total with "+" when some rows have no cost.
- **MEDIUM — Auto-archive hits an interrupted chat.** `MarkInterrupted` runs first, so the chat is not running. Auto-archive archives it when it is older than 30 days.
- **MEDIUM — A turn ends on a chat without totals.** `endTurn` reads the meta before it logs `turn_done`, so the backfill does not count that turn twice.
- **LOW — Bulk archive includes a busy chat.** The page archives the others and toasts how many it skipped.
- **LOW — A chat ID in the URL is archived.** The Agent view opens it read only, with the banner.

## Assumptions

- Cost is the `cost_usd` that the agent reports. xpo does not calculate cost from token prices.
- "Tokens in" includes cache reads, as the `turn_done` event reports it now.
- The page is only in the Agent view. The popup does not get a history page; the status-bar history menu links to `#/agent/history`.
- Auto-archive runs only at board startup, like `Prune` did.
