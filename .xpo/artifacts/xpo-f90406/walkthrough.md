## What changed

An agent's AskUserQuestion `question` can span several lines, for example a `•` bullet list before the actual question. The chat rendered that text with normal HTML whitespace rules, so the newlines collapsed and the bullets ran together into one paragraph.

The three elements that show question text now carry `whitespace-pre-line`:

- `QuestionCard.tsx`: the question heading on the answer step (the `<span id={labelId}>` that also labels the radio group).
- `QuestionCard.tsx` `QuestionReview`: the question above each answer on the Review step.
- `ChatThread.tsx` `AnsweredQuestion`: the answered question left in the thread.

## Why `pre-line`

`pre-line` keeps `\n` as line breaks but still collapses runs of spaces and trims them at line ends. Single-line questions render exactly as before, and stray double spaces from the model don't show up as gaps. `pre-wrap` (used for user messages) would keep those spaces too, which we don't want for model-written text.

Option labels, option descriptions and answers keep the old rendering. The issue only covered the question text.

## Tests

Each place has a render test with the question `"Two groups:\n• A\n• B\nWhere?"`. The test checks that the element has the `whitespace-pre-line` class and contains the raw newlines. The tests were written first and failed before the fix.

## Acceptance Criteria

- [x] A multi-line question shows each line on its own line in the question card heading. Test: `QuestionCard with one question > keeps the line breaks in a multi-line question`.
- [x] The same holds on the Review step and in the answered question in the thread. Tests: `QuestionReview > keeps the line breaks…` and `ChatThread answered question > keeps the line breaks…`.
- [x] Single-line questions render unchanged. All existing QuestionCard and ChatThread tests still pass. `pre-line` has no effect on text without newlines.
- [x] Render tests cover a multi-line question in all three places, written first. They failed before the fix and pass after it.
- [x] `make test` passes: exit 0, 1440 frontend tests, plus Go tests.
