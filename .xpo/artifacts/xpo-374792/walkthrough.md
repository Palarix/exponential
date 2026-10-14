## What changed
On a DOING issue, the Continue Work button in issue details is now secondary while a chat is in Work Mode on the issue. With no such chat it stays primary. Clicking it does the same as before: it opens the Work Mode chat, or creates one (xpo-00f0e5).

Primary styling marks the next step (xpo-926f0b). Once a chat holds the issue, the work is already underway there, so continuing it is no longer the next step.

## How it works
`PropertySidebar` already called `useIssueChats(issue.id)` for the "k chats" row (xpo-219de9). That hook returns `holder`, the chat whose `mode` is `"work"` and whose `work.issue_id` is this issue (`issue-chats.ts`).

The sidebar now computes one flag:

```ts
const workSecondary = showReview || (workAction === "Continue Work" && !!issueChats.holder);
```

The work button picks its classes from `workSecondary` instead of `showReview`. Nothing else changed.

## Decisions
- **Only Continue Work.** Start Work keeps its styling, because a chat can't usually hold a PLANNED or BACKLOG issue.
- **Any holder counts.** A chat in Work Mode on the issue counts whether it's idle, running, waiting or interrupted. Chats that worked on the issue earlier but have left Work Mode don't count.
- **Review Changes still takes priority.** If it shows, Continue Work stays secondary either way.
- **No new wiring for live updates.** `holder` comes from the agent chat list in context, which the board keeps polling, so the button restyles when that list changes.

## Acceptance criteria
- [x] Continue Work is primary when no chat is in Work Mode on the issue. Evidence: test "keeps Continue Work primary once the chat has left Work Mode", plus the existing "keeps the work button primary without Review Changes".
- [x] Continue Work is secondary when a chat is in Work Mode on the issue. Evidence: test "makes Continue Work secondary while a chat is in Work Mode on the issue".
- [x] The style updates live when a chat enters or leaves Work Mode. Evidence: the style is computed on each render from the live chat list in context. The two tests above cover each state, but nothing tests the switch itself and it wasn't tried in the board.
- [x] Clicking it works as before (xpo-00f0e5, xpo-219de9). Evidence: `handleStartWork` is unchanged, and the existing Continue Work tests pass.
- [x] `make test` passes. Evidence: exit 0, 70 frontend test files and 969 tests passed, plus lint and Go tests.
