# Walkthrough: the browser tab favicon shows agent activity

## What was built
When an agent chat needs you, the xpo tab's favicon gets a colored dot in its top-right corner. You can see this without switching to the tab.

| Attention | When | Favicon |
|---|---|---|
| approval | any chat waits on an approval, whatever the view or visibility | amber dot (`#f2c94c`, the `--color-warning` hue) |
| output | the tab is hidden and a chat's turn ended after it was hidden | blue dot (`#4da6e8`, the `--color-info` hue, the same as the chip "New reply" dot) |
| none | otherwise | the plain `/favicon.svg` |

The title stays `Project ❯ View/Issue` in every state.

## How the pieces fit together

### Attention logic: `tabAttention` in `agent-state-utils.ts`
`tabAttention(chats, hiddenSince)` is pure and looks at every chat, not only the chat on screen. A waiting approval wins. Otherwise, while the tab is hidden, a chat counts as new output when it is not running and its `updated_at` is later than `hiddenSince`. Times are compared with `Date.parse`, as in `chatIndicator`, because Go's fractional seconds vary in length and text order can be wrong. An interrupted turn bumps `updated_at` too, so it counts as output.

### Visibility: `hooks/useHiddenSince.ts`
The hook gives an ISO timestamp from the moment `visibilitychange` reports hidden, and null while the tab is visible. A second hidden event keeps the first timestamp. The output marker therefore clears as soon as you come back, and the chip dots from xpo-f2d4c8 take over from there.

### Favicon: `src/favicon.ts`
`faviconFor(attention)` builds a `data:image/svg+xml` URL from the real logo, imported `?raw` from `public/favicon.svg`. That keeps one source for the logo. A `<circle>` is added before `</svg>`:
- Center (264, 86), radius 73, a 26-unit ring. At 16px that is about 8px across, and it touches the top and right edges.
- The ring color comes from a `prefers-color-scheme` media query, the same way the logo's own colors do. The dot then stands apart from the logo in both schemes.
- The colors are hex literals, because a favicon can't read CSS variables.

The two badged URLs are built once at module load.

### Wiring: `TabStatus` and `App.tsx`
`App` renders `AgentChatProvider` itself, so it can't read the agent context. A small `TabStatus` component rendered inside the provider does it. It sets `document.title` from the `title` prop, and sets the `<link rel="icon">` href to `faviconFor(attention)`. App's old title effect moved into it, and App now only builds the title string.

### Background polling
The chat list polls every 2s while a chat runs. React Query's default `refetchIntervalInBackground: false` paused that poll in hidden tabs, so a background tab never learned that a turn ended. `AgentChatProvider` now sets it to `true`. This was required for the output marker to work at all.

## Decisions
- **Favicon only, no title prefix.** `(!)` and `(•)` prefixes were built first, then dropped in review because they looked odd. The favicon dot is the indicator.
- **Top-right dot, a bit larger.** Moved from the bottom right and grown about 1px during review.
- **Output only while hidden.** This follows the issue. A tab you are on doesn't need a marker; the chips show unseen replies.
- **No running marker.** Running is normal and not a call to action.

## Limits
- `hiddenSince` is client time and `updated_at` is server time. They agree on one machine. A skewed remote server could miss or misflag a reply.
- With no chat running, the list doesn't poll. A chat started elsewhere (another tab, the CLI) only shows up after the next refetch.
- Desktop notifications are the follow-up, xpo-a3f62b.

## Acceptance criteria
- [x] A tab with any chat waiting on approval shows an amber favicon dot, whatever view is open and whether or not the tab is visible. Evidence: the `tabAttention` tests "flags an approval whatever the visibility", and the `faviconFor` test for the amber fill; checked by the user.
- [x] A hidden tab that gets new agent output shows a blue favicon dot. It clears when the tab becomes visible. Evidence: `tabAttention` "flags a turn that ended while the tab was hidden" and "ignores output while visible…", and `useHiddenSince` resets to null on visible; checked by the user.
- [x] Approval wins over new output. Evidence: `tabAttention` "ranks an approval above new output".
- [x] The title is always the normal `Project ❯ …`; when nothing needs attention the favicon is `/favicon.svg`. Evidence: `TabStatus` sets the title unchanged; `faviconFor('none')` test.
- [x] The dot sits in the top-right corner of the favicon. Evidence: the `favicon.test.ts` "puts the dot in the top-right corner"; checked by the user.
- [x] The chat list keeps polling while the tab is hidden and a chat runs. Evidence: `refetchIntervalInBackground: true` on the chats query in `AgentChatProvider`.
- [x] Unit tests cover `tabAttention` and `faviconFor`. Evidence: `agent-state-utils.test.ts` "tab attention" and `favicon.test.ts`.
- [x] `make test` passes. Evidence: 68 test files and 919 frontend tests, plus the Go tests, exit 0.
