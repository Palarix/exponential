# Walkthrough: Activity avatars render with placeholder style

## What was wrong
`Avatar` (`web/src/components/ui/Avatar.tsx`) decides how to render from the `<email>` inside its `name` prop. With an email, it tries a gravatar and falls back to filled initials. Without one, it renders the dashed-border "unknown person" placeholder.

`ActivityTimeline` built each entry's author with `displayActor(created_by, on_behalf_of).principal`, which is `shortName(created_by)`, i.e. the name with the email removed. That short name was used for both the label and the avatar, so every avatar in the Activity section got no email and always showed the placeholder. The issue title said "missing name prop", but the prop was passed; it just had no email in it.

## What changed
- **`web/src/utils/format.ts`:** new `activityActor(createdBy, onBehalfOf?)`, which returns:
  - `identity`: the raw `created_by` (`Name <email>`), passed to `Avatar`
  - `name`: the short name, for the visible label
  - `tooltip`: `"on behalf of <short name>"` when `on_behalf_of` is set, otherwise `""`
- **`ActivityTimeline.tsx`:** each `ActivityEntry` now carries `actor`, `author` and `tooltip` (replacing `via`). For both system events and comments, the avatar and name are wrapped in one `<span>` inside a single `Tooltip`, so hovering either shows the delegation. `Tooltip` already renders its children unwrapped when `content` is empty, so entries with no delegation get no tooltip.

## Decisions
- **Avatar shows who did it (`created_by`), with "on behalf of" in a tooltip.** The user chose this during spec review. Before, the tooltip covered only the name and showed the bare delegator name; now it covers avatar + name and reads as a sentence.
- **New helper instead of changing `displayActor`.** `displayActor` also feeds the global Timeline (`Timeline.tsx`, `timeline-utils.ts`) and the Dashboard `ActivityFeed`, which build their own sentences from `principal`/`via`. Changing its return shape would have affected those views and their tests for no benefit.
- **Logic in a pure util.** The web package's tests are vitest unit tests on pure functions, with no DOM or testing-library setup. Putting the identity/tooltip logic in `format.ts` makes it testable without adding a component-testing stack.

## Acceptance Criteria
- [x] Activity system-event and comment avatars receive the full `Name <email>` identity of `created_by`. Evidence: `ActivityTimeline` passes `entry.actor` (`activityActor().identity`); `format.test.ts` asserts `identity` keeps the email.
- [x] Avatars show a gravatar or filled initials, not the dashed placeholder. Evidence: user checked it in the Vite dev server (5173 → API on 8080) and approved ("lgtm").
- [x] Visible author label stays the short name. Evidence: label renders `entry.author` = `activityActor().name`; tested.
- [x] When `on_behalf_of` is set, hovering the avatar or the name shows "on behalf of <name>"; otherwise no tooltip. Evidence: `tooltip` asserted in both `format.test.ts` cases; one `Tooltip` wraps avatar + name; checked in the browser.
- [x] Unit tests cover entry building and tooltip text. Evidence: `describe("activityActor")` in `web/src/utils/format.test.ts`. Rendering is not covered, since there is no DOM test setup (see Decisions).
- [x] `make test` passes. Evidence: exit 0; 24 files / 377 frontend tests, lint and Go tests green.
