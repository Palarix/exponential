# Walkthrough: paste attachments and images into agent chat

## What changed

The chat composer takes files: paste, drop, or the paperclip next to Send. The board saves them with the chat and hands them to the agent. Images go in as real image input; every file is also listed by its absolute path in the prompt. The transcript shows each message's attachments.

## How the pieces fit

### Storage (`internal/chat/attachment.go`)

`Store.SaveAttachments` writes each file to `.xpo/chats/<chat-id>/attachments/<uuid><ext>`.

- **The file name** is a new UUID plus the original name's extension, when it matches `^\.[a-z0-9]{1,10}$`. The agent sees `.log` or `.pdf`, and nothing else from the browser's name reaches the file system.
- **The `.xpo/chats/` directory** is already private (0700/0600) and gitignored, so attachments need no new setup. `Store.List` only reads `<uuid>.json`, so the per-chat directory does not get in its way.
- **`Store.Delete`** removes the directory with the chat. Archiving keeps it.

We store attachments with the chat, not in a temp directory as other harnesses do. A temp directory gets cleaned by the OS, and the transcript would lose its images after a reload.

**Image detection sniffs the content** (`http.DetectContentType`) and does not trust the browser's MIME type. Only PNG, JPEG, GIF and WebP count as images. A file that claims `image/png` but is not one is saved as a plain file. This matters because Claude rejects a malformed image block, and that would fail the whole turn.

`Store.Attachment(chatID, attachmentID)` looks a file up by UUID and sniffs it again. Both IDs must be UUIDs, so a path can never escape the directory.

`ValidateAttachments` checks the limits: 10 per message, 5 MB per image (Anthropic's per-image limit), 10 MB per other file, and no empty files. It runs in `Runner.Send` before anything is written, so a rejected message leaves no trace.

### Getting attachments to the agents

`Session.Send` now takes a `Prompt{Text, Images}` instead of a string. `Images` holds the absolute path and the sniffed MIME type of each image.

- **Claude** (`claudeUserMessage`): with images, the stream-json `content` is an array: one base64 `image` block per image, then the text. With no images it stays a plain string, as before.
- **Codex** (`codexUserInput`): the `turn/start` input is the text item, then a `localImage` item with the path for each image. Codex reads the file itself.

`buildPrompt` adds an `<attachments>` block after the user's text. It lists every attachment with its absolute path, and marks images `(image, attached)`. Images are listed too, so the agent can copy a screenshot into the repo, for example. Both agents can read under the project root, which includes `.xpo/chats/`.

`Runner.Send` saves the files after the turn is registered and the meta is checked, then records them on the `user_message` event as `attachments: [{id, name, mime, size}]`. A chat with no text takes its placeholder title from the first attachment's name.

### Server (`internal/server/chat.go`)

- **`POST /api/local/chats/{id}/messages`** takes `attachments: [{name, mime, data}]`, with `data` in base64; Go's JSON decoding turns it into `[]byte`. Text may be empty when there are attachments. `http.MaxBytesReader` caps the body at the worst case: 10 files at 10 MB each, as base64, plus 1 MB. `ErrAttachment` maps to 400.
- **`GET /api/local/chats/{id}/attachments/{aid}`** serves the file with `nosniff` and a long private cache. Only the four sniffed image types are served as images. Everything else is `application/octet-stream` with `Content-Disposition: attachment`, so a pasted HTML or SVG file can never run in the board's origin.

### Frontend

- **`attachment-utils.ts`** holds the pure logic: the limits (mirroring the server), `addAttachments`, which returns what fits and an error for what doesn't, base64 encoding (in chunks, so large files don't overflow `String.fromCharCode`), `formatSize` and `attachmentUrl`.
- **`ChatComposer`** keeps pending files as `{key, file, preview}`. `preview` is an object URL, for images only, freed on remove, on send and on unmount.
  - **Paste:** only files on the clipboard become attachments; plain text still pastes as text.
  - **Drop and paperclip:** drop on the composer box, or use the paperclip, which opens a hidden `<input type="file" multiple>`.
  - **Errors:** a limit error shows inline, above the input.
  - **Tooltip:** the paperclip uses the app's `Tooltip`. The browser's `title` tooltip appeared away from the button.
- **`useChatSession.send`** takes the files, encodes them, and sends them with the message.
- **`ChatThread`**: the reducer keeps `attachments` on the turn's user message. `MessageAttachments` shows them above the bubble. Image thumbnails link to the full image in a new tab; file chips download under their original name. A message with only attachments shows no empty bubble. Auth is cookie-based, so plain `<img>` tags work.

## Decisions worth knowing

- **Inline upload, not a separate endpoint.** One request per message, and no orphaned uploads when a chip is removed or a message is never sent. The cost is a larger JSON body, which the cap bounds.
- **Content sniffing decides what an image is**, both when saving and when serving.
- **The frontend limit check uses the browser's MIME type**, for speed. A fake 6 MB "PNG" is refused by the frontend, although the server would take it as a file. This edge case was accepted.

## Acceptance criteria

- [x] Pasting an image into the composer adds an image chip with a thumbnail; pasting text still inserts text. *Evidence: tested in the board; the screenshot reached the agent.*
- [x] Dropping files on the composer adds them as chips. *Evidence: `onDrop` uses the same `addFiles` path as paste.*
- [x] The paperclip button next to Send opens a file picker; the picked files are added as chips. *Evidence: `ChatComposer attachments` tests; seen in the board.*
- [x] × on a chip removes it before sending. *Evidence: the `ComposerAttachments` test renders the remove buttons.*
- [x] A message with attachments and no text can be sent. *Evidence: `TestRunner_ClaudeGetsAttachments`, `TestChat_SendWithAttachments`.*
- [x] Over 10 attachments, or a file over its size limit, shows an error and is not added; the server rejects it too (400). *Evidence: `addAttachments` tests, `TestValidateAttachments`, `TestChat_SendAttachmentValidation`.*
- [x] Sent attachments are saved under `.xpo/chats/<id>/attachments/` and recorded on the `user_message` event. *Evidence: `TestStore_SaveAttachments`, `TestRunner_ClaudeGetsAttachments`.*
- [x] Claude gets images as image content blocks; Codex gets them as `localImage` items. *Evidence: `TestRunner_ClaudeGetsAttachments`, `TestCodexDriver_Attachments`; confirmed with a real Claude chat.*
- [x] Non-image files are listed with their absolute paths in the prompt, for both agents. *Evidence: `TestBuildPrompt_Attachments`; the `<attachments>` block arrived in the real chat.*
- [x] The transcript shows a message's attachments after a reload; image thumbnails open the full image. *Evidence: `ChatThread attachments` tests.*
- [x] The serve endpoint rejects IDs that are not UUIDs and serves non-images as downloads. *Evidence: `TestChat_SendWithAttachments`, `TestStore_AttachmentRejectsBadIDs`.*
- [x] Deleting a chat deletes its attachments directory. *Evidence: `TestStore_DeleteRemovesAttachments`.*
- [x] `make test` passes.

## Things to know

- Codex image input is covered by `TestCodexDriver_Attachments` against a fake Codex, but has not been tried with a real Codex chat yet.
- PDFs go in as paths, not as native Claude document blocks.
