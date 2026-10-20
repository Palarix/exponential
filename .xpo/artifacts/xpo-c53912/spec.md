# Spec: Worked-on issue link in the chat title

## What

A chat that has worked on issues shows a link icon button right after its title. This applies to the Agent view topbar and the chat popup header. The issues are every issue the chat has entered Work Mode on, now or earlier.

- **One issue:** hovering over the icon shows the issue preview card, and a click opens the issue.
- **Several issues:** a click opens a menu with one row per issue: status icon, ID and title. The newest is first. The issue the chat is in Work Mode on now carries a "Working" marker. Picking a row opens that issue.
- **No issues:** a chat that never entered Work Mode shows no icon.

## Why

xpo-c94d85 removed the composer context chip in Work Mode. Now the bound issue shows only in the composer's mode segment, which is small and sits at the bottom. The title row is always visible.

The link stays after the chat leaves Work Mode, because the chat's history is still about that issue. A chat can enter and leave Work Mode on several issues in one timeline, so the link covers all of them.

## How

- **Data:** `ChatMeta.worked_issues` already lists every issue the chat entered Work Mode on. The server keeps it in entry order, without duplicates and after the chat leaves Work Mode (`internal/chat/mode.go`). No backend change is needed.
- **Shared hover preview:** extract the hover-preview behavior of `IssueRef` into `useIssueLink(id)` (`IssueRef/useIssueLink.tsx`). It returns the props for the `<a>` (ref, `href`, hover and focus handlers) and the `IssuePreview` card. `IssueRef` uses it too.
- **`ChatTitleBar`** derives everything from its `chat` prop, so the callers pass nothing new:
  - `issues` is `chat.worked_issues`, reversed (newest first);
  - `working` is `chat.work.issue_id` while `chat.mode === "work"`.
- It renders `WorkedIssues` between the title and the "⋯" menu, hidden while renaming:
  - one issue: `<a href="#/issues/<id>" aria-label="Go to <id>">` with lucide's `Link` icon and `useIssueLink`'s preview;
  - several: a `<button aria-label="Worked-on issues">` with the same icon. It opens a `Popover` + `Menu`. Each `MenuItem` has `icon` = `StatusIcon`, `content` = ID (mono, muted) + title (truncated), and `suffix` = "Working" for the current issue. Clicking a row calls `openIssue(id)` from `useAppNav`.
  - The icon button is `shrink-0`, so a long title truncates first.
- In the popup, a click opens the issue in the main view and the popup stays open.

## Decisions

- **Icon button, not a chip.** The first version used the `IssueRef` chip (status icon + ID). In tophat, the ID then showed many times on screen: in the title, the "Entered Work Mode" divider, the first message. A plain link icon is quieter. (Chosen by the user from five tophat variants.)
- **Preview card, not a tooltip,** for one issue. (Requested by the user.)
- **Placement:** right after the chat title. (Confirmed by the user.)
- **All worked-on issues, not only the current one.** (Requested by the user after tophat.)
- **Several issues: plain icon, no count badge.** (Confirmed by the user.)
- **No preview card on menu rows;** each row already shows status, ID and title. (Confirmed by the user.)

## Acceptance Criteria

- [ ] Agent view and popup: a chat with one worked-on issue shows the link icon after its title; a click opens the issue and hovering shows the preview card.
- [ ] The link stays after the chat leaves Work Mode.
- [ ] A chat with several worked-on issues opens a menu from the icon: newest first, the current Work Mode issue marked "Working", and a row click opens that issue.
- [ ] A chat that never entered Work Mode shows no icon.
- [ ] A long chat title truncates; the icon stays visible.
- [ ] Unit tests: one issue (link), after leaving Work Mode (link), several issues (menu button), none (no icon).
- [ ] `make test` passes.
