# Activity avatars render with placeholder style

## What
Avatars in the issue detail Activity section (system events and comments) render with the dashed-border placeholder instead of a gravatar or filled initials.

## Why (root cause)
`ActivityTimeline` builds each entry's `author` from `displayActor(evt.created_by, evt.on_behalf_of).principal`, which is `shortName(created_by)`: the display name with the `<email>` removed. That stripped string is then passed to `<Avatar name={entry.author} />`. `Avatar` pulls the email out of the `Name <email>` form, and when there is no email it skips the gravatar and applies `border-dashed bg-transparent`. So every activity avatar shows the "unknown person" placeholder.

## How
- Add an `actor: string` field (the raw `evt.created_by`) to both `ActivityEntry` variants.
- Render `<Avatar name={entry.actor} />`. Keep using `entry.author` (short name) for the visible label.
- The avatar and name show the person who did the action (`created_by`).
- When `on_behalf_of` is set, the avatar and name together are wrapped in a single tooltip that reads `on behalf of <via short name>`. With no `on_behalf_of`, there is no tooltip.
- Extract the entry-building logic into a pure exported helper (`buildActivityEntries`) so it can be unit tested.

## Decisions
- Avatar identity = `created_by` (who did it), confirmed by the user.
- Tooltip text changes from the bare via name to "on behalf of X" and covers avatar + name, confirmed by the user.

## Acceptance Criteria
- [ ] Activity system-event and comment avatars receive the full `Name <email>` identity of `created_by`
- [ ] Avatars show a gravatar or filled initials, not the dashed placeholder, when `created_by` contains an email
- [ ] Visible author label stays the short name
- [ ] When `on_behalf_of` is set, hovering the avatar or the name shows "on behalf of <name>"; otherwise no tooltip
- [ ] Unit tests cover entry building (full identity, via) and tooltip rendering
- [ ] `make test` passes
