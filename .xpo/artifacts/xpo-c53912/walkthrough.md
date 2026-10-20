# Walkthrough: Worked-on issue link in the chat title

## What was built

A chat's title row now links to the issues the chat worked on in Work Mode. The link sits right after the title, in both the Agent view topbar and the chat popup header.

- **One issue:** a link icon. Hovering shows the issue preview card, and a click opens the issue.
- **Several issues:** the same icon opens a menu. Each row shows the status icon, ID and title, newest first. The issue the chat is in Work Mode on now carries a "Working" marker. A row click opens that issue.
- **No issues:** nothing shows.

The link stays after the chat leaves Work Mode.

## Why

xpo-c94d85 removed the composer's context chip in Work Mode. After that, the only pointer to the bound issue was the small mode segment at the bottom of the composer. The title row is always visible, so the link goes there.

A chat can enter and leave Work Mode on several issues in one timeline, and its history stays about those issues after it leaves. So the link covers every worked-on issue, not only the current one.

## How the pieces fit

### `useIssueLink` (`IssueRef/useIssueLink.tsx`)

`IssueRef` already showed a preview card when you hover over an issue chip. That logic now lives in a hook, so the chip and the new icon share it:

- It looks up the issue in the issue list.
- It creates the hover intent with the 250 ms delay.
- It returns `linkProps` (ref, `href`, mouse and focus handlers) to spread on an `<a>`, and the `preview` element to render beside it.

`IssueRef` is now a thin wrapper: it spreads `linkProps` on its chip and renders `preview`.

The hook has its own file because lint's `react-refresh/only-export-components` rule does not allow a component file to export a hook.

### `WorkedIssuesLink` (`Agent/WorkedIssuesLink.tsx`)

It takes the chat's `ChatMeta` and reads `worked_issues`, reversed so the newest comes first. Then:

- With 0 issues, it renders nothing.
- With 1, it renders `IssueLink`: an `<a aria-label="Go to <id>">` with lucide's `Link` icon, plus `useIssueLink`'s preview.
- With 2 or more, it renders `IssuesMenu`: a `<button aria-label="Worked-on issues">` that opens a `Popover` + `Menu`. Rows are `MenuItem`s:
  - `icon` is the status icon;
  - `content` is the mono ID plus the truncated title;
  - `suffix` is "Working" for `chat.work.issue_id` while `chat.mode === "work"`;
  - `onClick` calls `openIssue` from `useAppNav`.

### `ChatTitleBar`

It renders `<WorkedIssuesLink chat={chat} />` between the title and the "⋯" menu, and hides it while renaming, like the menu button. All the data comes from the `chat` prop, so `AgentView` and `ChatPopup` did not change.

### Data

No backend change was needed. The server already keeps `worked_issues` on the chat record, in entry order and without duplicates (`internal/chat/mode.go`). It keeps the list after the chat leaves Work Mode.

## Decisions and how they evolved

1. **Chip → icon.** The first version used the `IssueRef` chip (status icon + ID). In tophat, the ID then showed several times on one screen: in the title, the "Entered Work Mode" divider, and the first message. A temporary tophat page put five variants side by side, and the user picked the plain link icon after the title.
2. **Preview card, not a tooltip.** The first icon version had a "Go to <id> <title>" tooltip. The user asked for the issue preview card, as on chips. That led to extracting `useIssueLink`.
3. **Every worked-on issue, not just the current one.** In tophat, a chat that had left Work Mode lost its link. The user asked to keep the link and to list every issue when there are several.
4. **Menu details (confirmed by the user):** a plain icon with no count badge, and no preview card on menu rows, because each row already shows status, ID and title.

## Acceptance Criteria

- [x] Agent view and popup: a chat with one worked-on issue shows the link icon after its title; a click opens the issue and hovering shows the preview card. *Evidence: unit test "links to the Work Mode issue with an icon right after the title" (href `#/issues/<id>`, before "Chat actions"). The user checked the popup, the Agent view and the hover card on the demo board.*
- [x] The link stays after the chat leaves Work Mode. *Evidence: unit test "keeps the issue link after the chat leaves Work Mode". The user found this case on the demo board, and it now works there.*
- [x] A chat with several worked-on issues opens a menu from the icon: newest first, the current Work Mode issue marked "Working", and a row click opens that issue. *Evidence: unit test "opens a menu of worked-on issues when there are several". The user approved the menu on the demo board.*
- [x] A chat that never entered Work Mode shows no icon. *Evidence: unit test "shows no issue link for a chat that never worked on an issue".*
- [x] A long chat title truncates; the icon stays visible. *Evidence: the icon is `shrink-0` and the title wrapper is `min-w-0` with `truncate`. Checked with a long title on the tophat page.*
- [x] Unit tests: one issue (link), after leaving Work Mode (link), several issues (menu button), none (no icon). *Evidence: four tests in `AgentView.test.tsx`.*
- [x] `make test` passes. *Evidence: lint clean, 1406 frontend tests and the Go suite pass.*
