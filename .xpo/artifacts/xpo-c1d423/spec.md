# Agent chat: paste attachments and images as context

## What

The user can paste or drop files and images into the chat composer. They show as chips above the input and can be removed. On send, the board saves them with the chat and hands them to the agent: images as real image input, other files as paths the agent can read. The transcript shows each message's attachments, with thumbnails for images.

## Why

Screenshots of a UI bug, logs or a design mock are often the fastest way to explain something. Today the user has to save the file and type its path.

## Where attachments live

Other harnesses keep pasted images outside the project. Claude Code caches them under `~/.claude/image-cache/`, and Codex passes a local file path. Neither keeps them with a conversation we control.

xpo stores them **with the chat**: `.xpo/chats/<chat-id>/attachments/<attachment-id>.<ext>`.

- `.xpo/chats/` is already private (0700/0600) and gitignored.
- The files outlive the turn, so the transcript can show them after a reload, and a resumed session can still read the paths.
- Deleting a chat deletes its attachments. Archiving keeps them.
- A temp directory would be cleaned by the OS and break the transcript.

## How the agents get them

| | Images (PNG, JPEG, GIF, WebP) | Other files |
|---|---|---|
| Claude | `image` content blocks (base64) in the stream-json user message | Path listed in the prompt |
| Codex | `localImage` input items with the saved path | Path listed in the prompt |

Every attachment is listed with its absolute path after the user's text. Images are marked, since they also go in as images; the path lets the agent copy a screenshot into the repo, for example.

```
<attachments>
- shot.png (image, attached): /abs/path/.xpo/chats/<id>/attachments/<aid>.png
- design.pdf: /abs/path/.xpo/chats/<id>/attachments/<aid>.pdf
</attachments>
```

Whether a file is an image is decided by **sniffing its content**, not by the type the browser claims. A file that claims `image/png` but is not one goes in as a file, so a bad image never fails the agent's turn.

The agent reads them with its own tools. Both agents can read files in the project, and `.xpo/chats/` is inside it.

## Flow

1. Paste (Cmd/Ctrl+V) with files on the clipboard, dropping files on the composer, or the paperclip button next to Send (a file picker, multiple files) adds them as pending attachments. Pasting plain text still pastes text.
2. Each attachment shows as a chip above the input: a thumbnail for images, an icon for other files, the name, and an × to remove it. A pending attachment over a limit shows the error and is not added.
3. A message can be sent with attachments and no text.
4. `POST /api/local/chats/{id}/messages` gets `attachments: [{name, mime, data}]`, data as base64. The server checks the limits, saves the files and records them on the `user_message` event as `attachments: [{id, name, mime, size}]`.
5. The runner builds the agent input from the text, the context and the attachments.
6. The transcript shows the attachments under the user's message. Images are thumbnails served by `GET /api/local/chats/{id}/attachments/{aid}`; clicking opens the full image in a new tab. Other files show as chips.
7. A chat with no title and no text takes its title from the first attachment's name.

## Limits

- Up to 10 attachments per message.
- Images up to 5 MB each (Anthropic's per-image limit).
- Other files up to 10 MB each.
- Any file type is accepted. Only the four image types go in as images; an SVG or HEIC goes in as a path.

## Decisions

- **Inline upload with the message, not a separate upload endpoint.** One request, no orphaned uploads when the user removes a chip or never sends.
- **Attachment IDs are UUIDs**, and the file name is the ID plus the original name's extension when it is plain (`[a-z0-9]{1,10}`), so the agent sees `.log` or `.pdf`, and a name from the browser never reaches the file system.
- **Limit errors show inline** in the composer, above the input, and clear on the next change.
- **The serve endpoint** sniffs the file again and sends an image type only for the four image types; everything else is `application/octet-stream` with `Content-Disposition: attachment`, so a pasted HTML or SVG file cannot run in the board's origin.

## Out of scope

- PDFs as native Claude document blocks.
- Attachments in the issue description or comments.

## Acceptance Criteria

- [ ] Pasting an image into the composer adds an image chip with a thumbnail; pasting text still inserts text.
- [ ] Dropping files on the composer adds them as chips.
- [ ] The paperclip button next to Send opens a file picker; the picked files are added as chips.
- [ ] × on a chip removes it before sending.
- [ ] A message with attachments and no text can be sent.
- [ ] Over 10 attachments, or a file over its size limit, shows an error and is not added; the server rejects it too (400).
- [ ] Sent attachments are saved under `.xpo/chats/<id>/attachments/` and recorded on the `user_message` event.
- [ ] Claude gets images as image content blocks; Codex gets them as `localImage` items.
- [ ] Non-image files are listed with their absolute paths in the prompt, for both agents.
- [ ] The transcript shows a message's attachments after a reload; image thumbnails open the full image.
- [ ] The serve endpoint rejects IDs that are not UUIDs and serves non-images as downloads.
- [ ] Deleting a chat deletes its attachments directory.
- [ ] `make test` passes.
