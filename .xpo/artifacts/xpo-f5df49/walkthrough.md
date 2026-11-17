## What changed
- `internal/chat/store.go`: `readEvents` reads the chat log with `bufio.Reader.ReadBytes('\n')`. It has no line limit. It skips a line that does not parse, and it parses an unterminated last line. It returns read errors other than `io.EOF`.
- `internal/chat/capinput.go` (new): `capInput` cuts each JSON string value over `maxInputString` (64 KiB), on a UTF-8 boundary. It appends `… [cut: N bytes total]`. Input of 64 KiB or less, and JSON with no long string, comes back unchanged. Input that is not valid JSON is cut as one string.
- `internal/chat/runner.go`: the `approval_requested` event stores `capInput(req.Input)`. The agent's allow response uses the original request input, so the agent gets the full input.

## Why this design
- **No line limit on the reader.** A `bufio.Scanner` with a 4 MiB cap failed the whole log on one long line. The agent's stdout reader accepts 16 MiB lines, so the board could write events it could not read back.
- **Cut string values, not the JSON text.** The card parses the input as JSON. Cut JSON text does not parse, and the card would show no arguments.
- **Token-stream rewrite.** `capValue` copies the JSON token by token. Key order stays the same, so the card lists the arguments in the agent's order. A `map` re-marshal would sort the keys. Strings are written without HTML escaping.
- **Only approval input is capped.** A stored `user_message` is what the agent received, and an edit re-sends it. The reader without a line limit handles long messages and replies.

## Acceptance criteria
- [x] Test: a chat whose log has a 5 MiB event opens with `Store.Get`. The test fails before the fix. Evidence: `TestStore_GetReadsEventsOverFourMiB` failed with `bufio.Scanner: token too long` against the old reader, and it passes now.
- [x] The test checks that every event, the large one included, comes back intact and in order. Evidence: the same test compares all three events' text and `Seq` 1–3.
- [x] Test: `Append` after a restart (new `Store`) on that log continues the sequence numbers. Evidence: the same test appends with a new `Store` and gets `Seq` 4.
- [x] Test: `capInput` cuts a long string value, keeps short values and keys, and returns valid JSON with the marker. Evidence: `TestCapInput_CutsLongStringValues` covers a top-level string, a string nested in an array, a number and a short value. It also checks valid UTF-8.
- [x] Test: `capInput` returns small input unchanged, byte for byte. Evidence: `TestCapInput_KeepsSmallInputByteForByte`.
- [x] Test: `capInput` cuts long invalid JSON as one string. Evidence: `TestCapInput_CutsLongInvalidJSONAsOneString`.
- [x] Test: the runner stores a capped `input` for an approval with a long argument. The agent still gets the full input. Evidence: `TestRunner_ApprovalStoresCappedInput` checks the live event and the logged event. It also checks that `updatedInput.content` in the recorded response has the full length.
- [x] `make test` passes. Evidence: exit 0, which includes lint, 1723 frontend tests and the Go tests.

Also added: `TestStore_ReadEventsSkipsBadLinesAndKeepsUnterminatedLast` keeps the old reader rules for bad lines and an unterminated last line.
