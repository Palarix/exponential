## What
On a DOING issue, Continue Work (xpo-00f0e5) renders as a secondary button while a chat is in Work Mode on the issue, and as the primary button otherwise.

## Why
Once a chat already holds the issue in Work Mode, continuing is no longer the obvious next step: the work is underway in that chat. Primary styling should mark the next step only (see xpo-926f0b, primary action on top).

## How
- `PropertySidebar` already reads `useIssueChats(issue.id)`, whose `holder` is the chat in Work Mode on the issue (`issue-chats.ts`).
- The work button is secondary when Review Changes shows (as today) **or** when the action is Continue Work and `issueChats.holder` is set.
- `holder` derives from the agent chat list, which the board keeps live, so the style follows chats entering and leaving Work Mode with no extra wiring.
- Start Work is unaffected. Click behavior is unchanged.

## Acceptance criteria
- [ ] Continue Work is primary when no chat is in Work Mode on the issue.
- [ ] Continue Work is secondary when a chat is in Work Mode on the issue.
- [ ] The style updates live when a chat enters or leaves Work Mode.
- [ ] Clicking it works as before (xpo-00f0e5, xpo-219de9).
- [ ] `make test` passes.
