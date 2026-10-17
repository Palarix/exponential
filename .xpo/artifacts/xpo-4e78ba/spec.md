# Spec: keep each chat's composer draft

## What
Every chat gets its own composer draft: the text and the pending attachments. The draft stays the same when you switch chats in the popup, and when an existing chat moves between the popup and the full Agent view, in both directions. A new chat (one that does not exist yet) has two separate drafts, one in the popup and one in the full view. Maximize moves the popup's draft into the full view, one way only. A draft is cleared only when its message is sent.

## Why
`ChatComposer` kept `text` and `attachments` in local `useState`. The popup renders one composer for whichever chat is open, so a draft typed in chat A showed up in chat B. The popup and the Agent view are separate composer instances, so a draft was lost on Maximize or Dock.

The two new-chat drafts are separate on purpose. A user who moved a long new-chat draft to the full view can then open the popup's new chat to fire off a side question, without touching the main draft.

## How
1. **A draft store** (`composer-drafts.ts`). It is a small external store keyed by a draft key: the chat ID for an existing chat, and `NEW_CHAT_POPUP` or `NEW_CHAT_VIEW` for the new chat in either place. A draft is `{ text, attachments: PendingAttachment[], prefilled? }`. Its operations are `get`, `setText`, `addFiles`, `removeAttachments`, `sent`, `prefill`, `prefillUsed`, `move` and `subscribe`. `AgentChatProvider` creates the store once and passes it through the context as a stable reference. The composer reads it with `useSyncExternalStore`, so a keystroke re-renders only that composer. Drafts last until a reload.
2. **The composer reads and writes the store.** `ChatComposer` takes `drafts` and `draftKey` props in place of its local text and attachment state. The popup passes `chatId ?? NEW_CHAT_POPUP`, and the Agent view passes `chatId ?? NEW_CHAT_VIEW`. A successful send clears the text and the sent attachments of the draft it started from.
3. **Maximize moves a new chat's draft.** Maximize on the popup's new chat calls `move(NEW_CHAT_POPUP, NEW_CHAT_VIEW)`. The popup's draft replaces the full view's draft, and the popup's new chat is left empty. The previews of the replaced attachments are revoked. An empty popup draft moves nothing, so Maximize on an empty new chat keeps the full view's draft. The full view's new chat has no Dock button, so nothing moves back. An existing chat keeps one draft under its ID, so it carries over both ways.
4. **The store owns the previews.** It makes an object URL for an image when the image is added, and revokes it when the attachment is removed, sent or replaced. The popup and the full view show the same URL.
5. **Start Work's prefill goes into the store.** `openChat(id, draft)` calls `prefill`, which replaces that chat's text and flags the draft. The composer focuses on the flag, puts the caret at the end, and calls `prefillUsed`. The one-shot `draftFor`, `clearDraft`, `draft` and `onDraftUsed` are removed.
6. **Cursor and focus.** When the draft key changes or the composer mounts, the caret goes to the end of the text. A popup chat switch keeps the same textarea, so focus stays where it was.
7. **The attach error is not part of the draft.** It resets when the draft key changes.

## Acceptance criteria
- [ ] Typing in popup chat A, then switching to chat B, shows B's own draft or an empty composer. Switching back shows A's draft.
- [ ] Pending attachments are scoped per chat in the same way.
- [ ] An existing chat keeps its draft text and attachments on Maximize and on Dock.
- [ ] Maximize on the popup's new chat moves its draft to the full view's new chat, replacing what was there. The popup's new chat is then empty.
- [ ] Typing in the popup's new chat while the full view's new chat has a draft leaves the full view's draft alone, and the other way round.
- [ ] The caret lands at the end of the draft. The composer keeps focus on a popup chat switch if it had it.
- [ ] Sending clears only that draft. A failed send keeps it.
- [ ] Start Work / Continue still prefill the composer and focus it.
- [ ] Unit tests cover the store and the composer's per-draft behaviour. `make test` passes.

## Edge cases
- **A new chat's first send.** A successful send clears the new-chat draft it came from, and the chat then has its own ID with an empty draft. If the send fails after the server has created the chat, the text stays in the new-chat draft and the now-created chat shows an empty composer. This is accepted.
- **Deleted or archived chats.** Their drafts stay in memory until a reload. The archived banner replaces the composer, so they are never shown.

## Decisions
1. Drafts do not survive a page reload, and are not expected to.
2. The popup's new chat and the full view's new chat have separate drafts. Only Maximize moves one into the other, from the popup to the full view.
3. On Maximize, the popup's draft replaces the full view's new-chat draft.
4. The gap for a failed first send of a new chat (see Edge cases) is accepted.
