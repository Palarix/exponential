# Agent chat: browser tab title and favicon show agent activity

## What
The browser tab shows when an agent chat needs you, without switching to it. The favicon gets a colored dot. The title is unchanged.

## Why
xpo-f2d4c8 added state dots to the status-bar chips, but you only see them on the tab. With several xpo tabs open, or the tab in the background, you can't tell an agent is waiting for an approval or has finished a reply.

## Behavior

Tab attention, most urgent first:

| Attention | Condition | Favicon |
|---|---|---|
| approval | any chat has `meta.waiting`, whatever the visibility or view | favicon + amber dot (same hue as `--color-warning`) |
| new output | the tab is hidden, and some chat that is not running has an `updated_at` later than the moment the tab became hidden | favicon + blue dot (same hue as `--color-info`, matching the chip "New reply" dot) |
| none | otherwise | normal `/favicon.svg` |

- Every chat counts, not only the chat on screen.
- Running alone changes nothing: no dot.
- The title keeps the normal `Project ❯ View/Issue` format in every state. Text prefixes such as `(!)` were tried and dropped as visually odd; the favicon dot is the indicator.
- New output clears on `visibilitychange` once the tab is visible again. It does not wait for the chat to be seen; the chip dots from xpo-f2d4c8 take over from there.
- An interrupted turn that ends while the tab is hidden counts as new output.
- Approval clears as soon as no chat waits.

### Background polling
The chat list polls every 2s only while a chat runs (`chatListPollMs`). React Query's default `refetchIntervalInBackground: false` pauses that poll in a hidden tab. Set it to `true` on the chat list query so a hidden tab sees turns end and approvals arrive.

## How
- `agent-state-utils.ts` (pure, unit-tested):
  - `tabAttention(chats, hiddenSince: string | null): "approval" | "output" | "none"`
- `favicon.ts`: `faviconFor(attention)` returns `/favicon.svg` for none. Otherwise it returns a `data:image/svg+xml` URL: the favicon SVG (imported `?raw` from `public/`) plus a dot in the top-right corner (radius 73 of the 350 viewBox, about 8px across with its ring at 16px, up from 7px; it touches the top and right edges). The dot has a ring in the logo's contrast color, so it stays legible in both color schemes. The dot colors are hex literals, since a favicon can't read CSS variables.
- `hooks/useHiddenSince.ts`: an ISO timestamp set when `visibilitychange` reports hidden, null while visible.
- `TabStatus` component (`components/Agent/TabStatus.tsx`), rendered inside `AgentChatProvider` in `App`. It reads `chats` from the agent context, then sets `document.title` to the given title and the `<link rel="icon">` href to `faviconFor(attention)`.
- `App.tsx`: builds the base `Project ❯ View/Issue` title as before and passes it to `TabStatus`. The old title effect is removed.
- `AgentChatProvider`: the chat list query sets `refetchIntervalInBackground: true`.

## Edge cases
- If the agent is unavailable, the chat list is empty and attention is none.
- Clock skew: `hiddenSince` is client time, `updated_at` is server time. On one machine (the normal xpo setup) they agree. A skewed remote server could miss a reply or flag it wrongly. Acceptable for a convenience marker.
- No poll runs when no chat runs. A chat started outside this tab (another tab, the CLI) is not picked up until something refetches. Out of scope.

## Out of scope
- Desktop notifications: xpo-a3f62b.
- A running marker.

## Acceptance criteria
- [ ] A tab with any chat waiting on approval shows an amber favicon dot, whatever view is open and whether or not the tab is visible.
- [ ] A hidden tab that gets new agent output shows a blue favicon dot. It clears when the tab becomes visible.
- [ ] Approval wins over new output.
- [ ] The title is always the normal `Project ❯ …`; when nothing needs attention the favicon is `/favicon.svg`.
- [ ] The dot sits in the top-right corner of the favicon.
- [ ] The chat list keeps polling while the tab is hidden and a chat runs.
- [ ] Unit tests cover `tabAttention` and `faviconFor`.
- [ ] `make test` passes.
