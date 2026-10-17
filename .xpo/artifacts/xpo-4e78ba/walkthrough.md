# Walkthrough: keep each chat's composer draft

## What was built
The composer's text and pending attachments now live in a draft store, one draft per chat, instead of in `ChatComposer`'s local state. This fixes two bugs:
- The popup renders one composer for whichever chat is open. Its local state did not change with `chatId`, so chat A's draft showed up in chat B.
- The popup and the Agent view are separate composer instances, so a draft was lost on Maximize or Dock.

## How the pieces fit
- **`composer-drafts.ts`** is a small external store: a `Map` from draft key to `{ text, attachments, prefilled? }`, plus a set of listeners. Its operations are `get`, `setText`, `addFiles`, `removeAttachments`, `sent`, `move`, `prefill`, `prefillUsed` and `subscribe`. An empty draft is deleted, so `get` returns the shared `EMPTY_DRAFT` and keeps returning the same object until the draft changes. `useSyncExternalStore` needs that stability.
- **Draft keys.** An existing chat's key is its ID, which the popup and the Agent view share, so its draft follows it both ways. A new chat has two keys, `NEW_CHAT_POPUP` and `NEW_CHAT_VIEW`. Both start with a NUL character, so they can never clash with a real chat ID.
- **`AgentChatProvider`** creates the store once (`useState(createComposerDrafts)`) and puts it on the context as `drafts`. It is a stable reference, so typing does not re-render the context's consumers. Only the composer, through `useComposerDraft`, re-renders on a keystroke.
- **`ChatComposer`** takes `drafts` and `draftKey` props. The popup passes `chatId ?? NEW_CHAT_POPUP`, and the Agent view passes `chatId ?? NEW_CHAT_VIEW`. Send captures the key before it awaits, because a new chat's first send changes `chatId` in the middle of the send. `sent` then clears that key's text and only the attachments that were sent, so a file added during the send stays.
- **Maximize** on the popup's new chat calls `drafts.move(NEW_CHAT_POPUP, NEW_CHAT_VIEW)`. The popup's draft replaces the view's, and the popup's new chat is left empty. An empty popup draft moves nothing. The flow is one way only: the Agent view's new chat has no Dock button.
- **Start Work / Continue** used to go through the one-shot `draftFor` / `clearDraft` / `draft` / `onDraftUsed`. Now `openChat(id, draft)` calls `drafts.prefill`, which sets the text and a `prefilled` flag. The composer's effect focuses the input on that flag, puts the caret at the end, and calls `prefillUsed`.

## Key decisions
- **The store owns the previews' object URLs.** The URLs are made when an image is added, and revoked when it is removed, sent or replaced by `move`. If each composer made its own URLs, unmounting the popup could revoke a URL that the Agent view still shows. An abandoned draft keeps its URLs until a reload, like the draft itself.
- **The new chat has a separate draft in the popup and the full view** (changed during review). After moving a long new-chat draft to the full view, the user can open the popup's new chat for a side question without touching the main draft.
- **Maximize replaces the view's new-chat draft.** This was the user's choice over appending or keeping the view's draft.
- **Drafts are in memory only.** They do not survive a reload, and attachments could not be stored anyway.
- **Accepted gap.** If a new chat's first send fails after the server has created the chat, the text stays in the new-chat draft, and the created chat shows an empty composer.

## Non-obvious details
- The attach error is not part of the draft. It resets when `draftKey` changes, through the "store the previous prop in state" pattern, not an effect.
- A popup chat switch keeps the same textarea, so focus stays where it was. An effect on `draftKey` moves the caret to the end of the new draft.

## Acceptance criteria
- [x] Typing in popup chat A, then switching to chat B, shows B's own draft or an empty composer. Switching back shows A's draft. *Evidence:* the "keeps a draft per chat" store test, the "shows the open chat's own draft" and "shows an empty input for a chat without a draft" composer tests, and the user's tophat.
- [x] Pending attachments are scoped per chat in the same way. *Evidence:* the "scopes attachments per chat" store test, the "shows the open chat's pending attachments only" composer test, and the tophat.
- [x] An existing chat keeps its draft text and attachments on Maximize and on Dock. *Evidence:* both views use the chat ID as the key; checked in the tophat.
- [x] Maximize on the popup's new chat moves its draft to the full view's new chat, replacing what was there, and the popup's new chat is then empty. *Evidence:* the "moves a draft, replacing the target's and freeing its previews" store test, and the tophat.
- [x] Typing in the popup's new chat leaves the full view's new-chat draft alone, and the other way round. *Evidence:* the "keeps the popup's and the full view's new chat drafts apart" store test, and the tophat.
- [x] The caret lands at the end of the draft, and the composer keeps focus on a popup chat switch. *Evidence:* the tophat.
- [x] Sending clears only that draft, and a failed send keeps it. *Evidence:* the "clears only the sent text and attachments of one chat" store test. `sent` runs only when `onSend` resolves true.
- [x] Start Work / Continue still prefill the composer and focus it. *Evidence:* the "prefills text, flagged until the composer takes it" store test; Start Work was checked in the tophat once the demo project was a git repo.
- [x] Unit tests cover the store and the composer's per-draft behaviour, and `make test` passes. *Evidence:* `make test` exit 0, 1217 frontend tests passing.
