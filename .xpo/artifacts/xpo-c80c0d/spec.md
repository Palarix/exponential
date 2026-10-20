# Preferences page: profile, Motion and Animations, diff view, and agent chat defaults

## What

Clicking the user block at the bottom left of the sidebar opens a full page at `#/preferences`. The `g ,` shortcut opens it too. The page is built from cards, in sections, in this order:

- **My Profile:** a card with avatar, name and email, from `GET /api/user` (`useUser()`). Read-only.
- **Agent Chat:** shown only when agent chat is available.
  - **Default model:** the provider, model and effort every new chat starts with. A chat can pick another model; that pick applies to that chat only.
  - **Desktop notifications:** on or off.
- **Preferences:** one card per setting.
  - **Motion**, with three choices:
    - **System default** (the default): follows the OS `prefers-reduced-motion` setting.
    - **On:** all animations play.
    - **Off:** no movement and no fades; changes are instant. Loading and progress indicators still animate.
  - **Diff view:** Unified (default) or Split: how the Changes tab's diffs open. The tab's own toggle still switches one view temporarily.

## Why

Today the only motion control is the OS reduced-motion setting, through Tailwind's `motion-safe:` variant, and it is applied unevenly: some movement (`animate-scale-in`, `animate-slide-up`) plays under reduced motion, and there is no way to opt in or out in the app. A user may also find instant jumps in the agent chat more disruptive than a gentle ease (xpo-50b9d9), and want motion On despite the OS setting.

The diff view, the default agent model and desktop notifications are already per-browser settings, but each one is set only in the place it applies to. The Preferences page gives them one home. The Issues view's story points toggle stays where it is: it is a quick view toggle, not a preference.

The default model and the diff view were "the last one picked anywhere", so one experiment in one chat or one merge view changed every later one. As preferences they become stable defaults, and a pick in a chat or a merge view is an override for that one place.

## Animation tiers

Every animation in the app belongs to one of three tiers. The tiers follow current accessibility guidance: WCAG 2.3.3 (Animation from Interactions) asks that motion can be turned off unless essential, and defines motion as changes to perceived size, shape or position, not opacity or color. WCAG 2.2.2 and platform guidance (Apple Reduce Motion, Material) keep progress indicators, because without them the app looks stuck.

| Tier | Animations | On | System, OS reduced motion | Off |
|---|---|---|---|---|
| **Movement** | `animate-step-in-forward`/`-back`, `animate-pop-in`, `animate-slide-up`, `animate-scale-in`, smooth scroll | play | replaced by a plain fade (scroll jumps) | none (scroll jumps) |
| **Fade** | `animate-fade-in` | play | play | none |
| **Status** | `animate-spin`, `animate-pulse`, `.text-shimmer` | play | play | play |

System default without OS reduced motion behaves like On.

## How

### Preferences store: `src/preferences.ts`

- One localStorage key, `exponential-preferences`, holds a JSON object: `{ "motion": "system" | "on" | "off", "diffView": "unified" | "split" }`.
- `parsePreferences(raw)` fills in a default for every missing or invalid field, so later settings can be added without a migration.
- A small external store: `getPreferences()`, `setPreference(key, value)`, `subscribe()`. It listens to the `storage` event, so a change in one tab applies to the others.
- `usePreferences()` reads it through `useSyncExternalStore`.
- If localStorage is blocked, the defaults are used and a change lasts for the session.

### Motion resolution

- `resolveMotion(pref, systemReduced)` returns `"full"`, `"reduced"` or `"off"`: `on` → `full`; `off` → `off`; `system` → `reduced` when the OS asks for reduced motion, else `full`.
- `startMotionSync()` runs in `main.tsx` before the first render. It sets `<html data-motion="full|reduced|off">` and updates it when the preference or the `(prefers-reduced-motion: reduce)` media query changes, so the first paint already uses the right tier.
- `useMotion()` returns the resolved value. `motionAllowed()` is a non-hook helper, true only for `full`; `scrollBehavior()` returns `smooth` or `auto` from it. JS-driven motion uses these (smooth scroll here, the chat scroll and fade in xpo-50b9d9).

### CSS: tiers live in `index.css`

- Each animation utility is tagged with its tier in `index.css`. Unlayered rules keyed on the root attribute override them:
  - `[data-motion="reduced"]` swaps movement utilities to the `fade-in` keyframes.
  - `[data-motion="off"]` sets `animation: none` on movement and fade utilities.
  - Status animations have no override.
- Components use the plain utility (`animate-pop-in`), no `motion-safe:` prefix: the tier decides. The `motion-safe:` prefixes in QuestionCard and `stepInClass` are removed.
- Tailwind's `motion-safe:` / `motion-reduce:` variants are redefined to read the root attribute (`full` / not `full`), so any future use follows the preference too.

### JS smooth scrolls

The three `behavior: "smooth"` scrolls use `scrollBehavior()`: `IssueDetail.tsx`, `MergeView.tsx`, `ChatThread.tsx`.

### Diff view

- The Preferences card sets `diffView`, the default for every diff. It replaces the old `exponential-diff-mode` key, which is not read: a split-view choice made before this change resets to Unified once.
- The Changes tab's Unified/Split toggle (`MergeView.tsx`) is a temporary override: it switches the diff it sits on, for as long as that view stays open, and does not change the preference. Opening another merge view starts from the preference again. Without an override, an open diff follows the preference live.

### Agent chat: default model and per-chat overrides

The default model stays in its existing key (`xpo.agent.prefs`), owned by the agent chat context (`AgentChatProvider`). What changes is who writes it:

| Action | Before | After |
|---|---|---|
| Pick on the Preferences card | n/a | sets the default |
| Pick in an existing chat's composer | that chat + the default | that chat only |
| Pick in a new, unsent chat's composer | the default | that new chat only; it carries over to the chat once created |

- Context API:
  - `defaultPrefs` (normalized against the provider list) and `setDefaultPrefs(prefs)`: the preference. Only the Preferences card calls the setter.
  - `prefsFor(chatId)` / `setPrefsFor(chatId, prefs)`: a chat's composer. Picks live in the session-only `picked` map; the new, unsent chat's pick lives under the key `""` (`NEW_CHAT`, shared with the consumed-context records).
  - `adoptNewChatPrefs(id)`: when sending creates the chat (`useChatSession`), its `""` pick moves to the new ID, so the composer does not flicker back to the default before the chat's meta arrives, and the next new chat starts from the default again.
- `chatPrefs(meta, picked, defaults)` is unchanged: a pick in the chat, else what the chat's last message used, else the default.
- The card reuses the composer's model and effort menus: `ComposerPickers.tsx` exports them as `ModelPickers`, and the composer uses that same component.
- **Model names:** the menus show what the provider list knows. The full name ("Claude Opus 5.5") and context window ("1M") are learned once a turn has used a model; before that, both the composer and the card show the short label ("Opus").

### Desktop notifications

`notify` and `toggleNotify()` from the agent chat context. Turning them on asks the browser for permission, as the bell does today. When the browser has blocked notifications, or has no Notification API, the card shows why instead of a switch. The Agent Chat section is hidden when agent chat is not available (`status.available` is false).

### Route, shortcut and sidebar

- `views.ts` gets a `preferences` view: route `preferences`, title "Preferences", goKey `,`. The section type gets a third value, `"none"`, so it is not listed in the sidebar nav.
- The sidebar user block becomes a button, titled "Preferences", that opens the page and is highlighted while it is open. Collapsed, only the avatar shows.
- `ViewRouter` lazy-loads `components/Preferences/Preferences.tsx`.

### Page layout

`ViewContainer` + `TopBar` titled "Preferences". A centered column (`max-w-2xl`) of sections: My Profile, Agent Chat, Preferences. Each section has a small heading above its cards; cards use the `Card` primitive.

- **My Profile:** one card with a 40px avatar (`Avatar` gets an `xl` size), the name, and the email in muted text.
- A setting card has its title and a one-line description on the left, its control on the right, and an optional hint line below.
- **Default model:** "New chats start with this model and effort." The model and effort menus. No hint.
- **Desktop notifications:** "Notify me when a chat waits for my input while this tab is in the background." A switch. In place of the switch: "Blocked by the browser" (hint: "Allow notifications for this site in your browser's settings.") or "Not supported".
- **Motion:** "How much the interface moves." A segmented control (System / On / Off), built from native radio inputs so arrow keys work. The hint explains the current choice: for System default it says what the OS asks for now ("Your system asks for reduced motion, so movement becomes a fade."); for Off, "Loading and progress indicators still animate."
- **Diff view:** "How the Changes tab shows diffs." A segmented control (Unified / Split).

## Edge cases

- **Invalid stored value:** an unknown `motion` or `diffView` value, bad JSON, or a non-object falls back to the defaults, field by field.
- **OS setting changes while the app is open:** under System default, the app follows immediately, and the hint updates.
- **No user (`/api/user` fails):** the profile card says "Not signed in."
- **Saved model no longer offered:** `defaultPrefs` is normalized, so the card shows the model new chats will really use.
- **Overrides are session-only:** a pick in an unsent new chat is dropped on reload; a sent chat keeps its model in its meta. A merge view's diff toggle lasts while that view is open.
- **Out of scope:** color, opacity and width `transition-*` classes (hover states, sidebar collapse, chevron rotation). The `animate-in fade-in-0 zoom-in-95` classes on tooltips and chat popovers do nothing today (no `tw-animate-css` plugin), so they are left alone.

## Acceptance criteria

- [ ] Clicking the sidebar user block opens `#/preferences`; the URL is bookmarkable and the back button works.
- [ ] `g ,` opens the Preferences page.
- [ ] The page shows, in order: a My Profile card; when agent chat is available, an Agent Chat section with Default model and Desktop notifications cards; a Preferences section with Motion and Diff view cards.
- [ ] Motion and Diff view default to System default and Unified, and persist across reloads in `exponential-preferences`.
- [ ] **On:** movement, fades and status animations play, even with OS reduced motion on.
- [ ] **System default with OS reduced motion:** movement becomes a fade and smooth scrolls jump; fades and status animations play.
- [ ] **Off:** no movement or fade animations, smooth scrolls jump; spinners, pulse and shimmer still animate.
- [ ] **System default** follows live changes to `prefers-reduced-motion`.
- [ ] Diff view set on the page is how every Changes tab opens; the Changes tab's toggle switches only that open view and leaves the preference alone.
- [ ] The Default model set on the page is what every new chat starts with, and persists across reloads.
- [ ] A pick in a chat's composer (new or existing) changes only that chat; the Preferences card and the next new chat keep the default.
- [ ] The notifications switch matches the chat's bell, asks for permission when turned on, and explains when blocked or unsupported.
- [ ] `useMotion()` and `motionAllowed()` are exported for xpo-50b9d9.
- [ ] Unit tests cover `parsePreferences`, `resolveMotion`, the route parse/format for `preferences`, the views registry, the motion tiers in `index.css`, the cards' hints, and adopting a new chat's pick.
- [ ] `make test` passes.
