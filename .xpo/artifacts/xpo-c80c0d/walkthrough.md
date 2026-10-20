# Walkthrough: Preferences page

## What was built

A `#/preferences` page, opened from the sidebar avatar or `g ,`. It has three sections built from cards:

- **My Profile:** whoami, from `useUser()`.
- **Agent Chat:** Default model and Desktop notifications.
- **Preferences:** Motion and Diff view.

The issue asked only for Motion. The other cards came out of review. The agent chat cards and the diff view already existed as per-browser settings, each set only where it applies, and the page gives them one home. Two of them also changed meaning along the way: the default model and the diff view used to be "whatever was last picked anywhere", and are now real defaults with per-place overrides.

## How the pieces fit

### `src/preferences.ts`: one store for page-owned settings

- A single localStorage key, `exponential-preferences`, holds `{ motion, diffView }`.
- `parsePreferences` validates each field on its own, so adding a setting needs no migration.
- The store follows the same external-store pattern as elsewhere: `getPreferences`, `setPreference`, `subscribe`, plus `usePreferences()` through `useSyncExternalStore`.
- `subscribe` also listens to the `storage` event, so other tabs stay in sync.
- The hooks pass a server snapshot, because MergeView is rendered with `renderToString` in tests.

The agent settings are not in this store. They belong to `AgentChatProvider`, which already owned them (`xpo.agent.prefs`, `xpo.agent.notify`) together with the permission flow. Copying them into a second store would have meant two sources of truth.

### Motion: three tiers, decided in CSS

The model comes from WCAG 2.3.3 and platform guidance. Reduced motion means removing movement, which can trigger vestibular issues, and keeping status feedback.

| Tier | Utilities | full | reduced | off |
|---|---|---|---|---|
| Movement | step-in, pop-in, slide-up, scale-in, smooth scroll | play | fade instead | none |
| Fade | fade-in | play | play | none |
| Status | spin, pulse, `.text-shimmer` | play | play | play |

How the tiers are applied:

- `resolveMotion(pref, systemReduced)` maps the preference and the OS setting to `full`, `reduced` or `off`.
- `startMotionSync()` runs in `main.tsx` before the first render. It writes `<html data-motion>` and keeps it updated as the preference and the media query change.
- In `index.css`, two unlayered rules keyed on that attribute swap movement utilities to the `fade-in` keyframes (`reduced`), or set `animation: none` (`off`). Unlayered rules beat Tailwind's utilities layer whatever their specificity.
- Components therefore use plain `animate-pop-in`, with no `motion-safe:` prefix. The tier lives next to the keyframes, in one place.
- `motion-safe:` and `motion-reduce:` are redefined with `@custom-variant` to read the attribute, so future uses of the variants follow the preference too.
- JS motion goes through `motionAllowed()` and `scrollBehavior()`. The three existing smooth scrolls use them, and xpo-50b9d9's chat scroll and fade will too.
- Status animations are deliberately never overridden. Earlier decisions in xpo-855e28 (shimmer) and xpo-ec0480 (spinners) already kept them running under reduced motion, because they are the only sign the app is working. Off keeps that.

### Default model: a preference plus per-chat overrides

- `AgentChatProvider` now separates `defaultPrefs` and `setDefaultPrefs` (the preference, set only by the Preferences card) from `prefsFor` and `setPrefsFor` (a chat's composer).
- Composer picks go into the session-only `picked` map. The new, unsent chat uses the existing `NEW_CHAT` (`""`) key, the same one the consumed-context records use.
- When sending creates the chat, `useChatSession` calls `adoptNewChatPrefs(id)`. The pick moves to the real ID, so the composer doesn't flicker back to the default before the chat's meta arrives, and the next new chat starts from the default again.
- After the first message, the chat's meta records its model, and `chatPrefs` reads it from there as before.
- The card reuses the composer's menus: `ModelPickers` was extracted from `ComposerPickers`.
- Full model names and context windows ("Claude Opus 5.5 · 1M") only show once a turn has used the model. That is how the provider list learns them, so a fresh board shows short labels in both places.

### Diff view: same pattern

- The card sets `diffView`. MergeView's toolbar toggle is a local `override` state, so a view you haven't toggled follows the preference live, and a toggle lasts while that view is open.
- The old `exponential-diff-mode` key isn't read. Per the project's no-backward-compat rule, a Split choice resets once.

### Route and sidebar

- `views.ts` gets a `preferences` view, with goKey `,` and a new `section: "none"` so it stays out of the nav.
- The sidebar user block is now a button, highlighted while the page is open.
- `Avatar` gains an `xl` size (40px), and `Toggle` an `ariaLabel` prop.

## Things a future reader should know

- **Ungated `animate-in fade-in-0 zoom-in-95`:** the tooltips and chat popovers carry these classes, but no `tw-animate-css` plugin is installed, so they do nothing. They were left alone.
- **Out of scope for Motion:** `transition-*` classes (hover colors, the sidebar width, chevron rotation).
- **A new animation utility:** add its class to the matching tier rule at the end of the ANIMATIONS section in `index.css`. `motion-css.test.ts` pins the tier lists.

## Acceptance criteria

- [x] Clicking the sidebar user block opens `#/preferences`, which is bookmarkable and works with the back button. Evidence: `views.test.ts` covers parse and format for `#/preferences` and the round-trip; `Sidebar.tsx` calls `onViewChange("preferences")`; checked by the user on the demo board.
- [x] `g ,` opens the page. Evidence: the `VIEW_BY_GO_KEY` test includes `",": "preferences"`, and the chord comes from that map in `App.tsx`.
- [x] Sections are in order: My Profile, Agent Chat (only when available), Preferences. Evidence: `Preferences.tsx`; the user's review on the demo board, including the requested reorder.
- [x] Motion and Diff view default to System and Unified and persist in `exponential-preferences`. Evidence: `preferences.test.ts` covers defaults, field-by-field fallback, and bad JSON.
- [x] On: everything plays, even under OS reduced motion. Evidence: the `resolveMotion("on", true) === "full"` test; no tier rule for `full`.
- [x] System with OS reduced motion: movement turns into a fade and smooth scrolls jump; fades and status animations play. Evidence: `resolveMotion` tests; the `motion-css.test.ts` "swaps movement for a fade" test; `scrollBehavior()` returns `auto` unless the tier is `full`.
- [x] Off: no movement or fades, and smooth scrolls jump; spinners, pulse and shimmer still animate. Evidence: `motion-css.test.ts` "stops movement and fades" and "never stops status animations".
- [x] System default follows live OS changes. Evidence: `startMotionSync` and `useMotion` subscribe to the media query's `change` event.
- [x] The diff view preference is how every Changes tab opens, and the tab's toggle switches only that view. Evidence: the MergeView `override ?? diffView`; the user's review.
- [x] The Default model on the page is what new chats start with, and it persists. Evidence: `setDefaultPrefs` saves to `xpo.agent.prefs`; `chatPrefs` falls back to it.
- [x] A composer pick changes only that chat. Evidence: `setPrefsFor` no longer saves; `adoptNewChatPick` tests; the user's review.
- [x] The notifications switch matches the bell, asks for permission, and explains blocked or unsupported. Evidence: it uses the context's `notify` and `toggleNotify`; `notifyCopy` tests.
- [x] `useMotion()` and `motionAllowed()` are exported for xpo-50b9d9. Evidence: `preferences.ts`.
- [x] Unit tests cover the store, motion resolution, routes and registry, CSS tiers, card hints, and pick adoption. Evidence: `preferences.test.ts`, `views.test.ts`, `motion-css.test.ts`, `preferences-utils.test.ts`, `agent-state-utils.test.ts`.
- [x] `make test` passes. Evidence: 94 files and 1422 tests green, lint clean.
