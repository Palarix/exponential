## What

Keep the line breaks in an agent's AskUserQuestion `question` text wherever the chat shows it.

## Why

Agents put bullet lists or context lines before the actual question. Today the card runs them into one paragraph with inline `•` bullets, which is hard to read.

## How

Add `whitespace-pre-line` to the three elements that render the question text. It keeps `\n` as line breaks but still collapses runs of spaces, so normal single-line questions look the same.

- `web/src/components/Agent/QuestionCard.tsx`: the question heading (`<span id={labelId}>`) on the answer step.
- `web/src/components/Agent/QuestionCard.tsx`: the question on the Review step.
- `web/src/components/Agent/ChatThread.tsx` `AnsweredQuestion`: the answered question in the thread.

Out of scope: option labels/descriptions and the answer text.

## Acceptance Criteria

- [ ] A multi-line question shows each line on its own line in the question card heading.
- [ ] The same holds on the Review step and in the answered question in the thread.
- [ ] Single-line questions render unchanged.
- [ ] Render tests cover a multi-line question in all three places (written first).
- [ ] `make test` passes.
