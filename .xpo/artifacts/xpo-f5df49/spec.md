## What
`Store.readEvents` reads a chat's log with no limit on line length. The board cuts long string arguments in the stored input of an `approval_requested` event. One large event no longer makes the chat unreadable.

## Why
`readEvents` uses a `bufio.Scanner` capped at 4 MiB per line. One longer event stops the scan with `bufio.ErrTooLong`. `Store.Get` then fails, `GET /api/local/chats/{id}` answers 500, and the replay of `chat:<id>` fails. The line stays in the log, so the failure is permanent. `Append` also calls `readEvents` on its first write after a board restart, so the chat cannot take new events either.

The agent's stdout reader (`proc.go`) accepts lines up to 16 MiB. So the board can store events that its own log reader then rejects.

The realistic large event is `approval_requested`: it stores the full tool input, for example the content of a large `Write`. The approval card (`approvalArgs`) shows every argument in full, so a multi-megabyte value only bloats the log, the replay and the page. The agent answers from its own copy of the input (`approval.go`), so the stored copy is display-only.

## How
1. In `readEvents`, replace the `bufio.Scanner` with a `bufio.Reader` and `ReadBytes('\n')`.
2. Keep the current rule: skip a line that does not parse as an `Event`.
3. Parse an unterminated final line too, as the scanner did.
4. Return read errors other than `io.EOF`.
5. Add `capInput(input string) string` in `internal/chat`. It parses the input as JSON and walks it.
6. In `capInput`, cut each string value longer than 64 KiB to 64 KiB, on a UTF-8 boundary.
7. Append a marker to each cut string: `… [cut: N bytes total]`.
8. Re-marshal the JSON only when `capInput` cut a value. Otherwise return the input unchanged.
9. When the input is not valid JSON and is longer than 64 KiB, cut it as one string with the same marker.
10. In `runner.go`, store `capInput(req.Input)` in the `approval_requested` event.

## Decisions
- **Cut string values, not the JSON text.** A cut JSON text does not parse, and the card then shows no arguments. Cutting values keeps every key on the card.
- **64 KiB per string value.** That is more than a card can usefully show, and far below the old 4 MiB line limit.
- **The marker is in the text.** The card shows it with no frontend change.
- **No cap on `user_message` or `assistant_text`.** A user message is what the agent received, and editing a message re-sends the stored text. Cutting it would change the conversation. The reader without a line limit handles both, and the 16 MiB stdout limit bounds `assistant_text`.
- **Skip, no placeholder event.** `readEvents` already skips unparseable lines. A reader without a line limit no longer hits the large-line case.

## Acceptance criteria
- [ ] Test: a chat whose log has a 5 MiB event opens with `Store.Get`. The test fails before the fix.
- [ ] The test checks that every event, the large one included, comes back intact and in order.
- [ ] Test: `Append` after a restart (new `Store`) on that log continues the sequence numbers.
- [ ] Test: `capInput` cuts a long string value, keeps short values and keys, and returns valid JSON with the marker.
- [ ] Test: `capInput` returns small input unchanged, byte for byte.
- [ ] Test: `capInput` cuts long invalid JSON as one string.
- [ ] Test: the runner stores a capped `input` for an approval with a long argument. The agent still gets the full input.
- [ ] `make test` passes.
