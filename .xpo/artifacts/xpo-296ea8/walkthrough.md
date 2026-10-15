## What changed

The empty Agent view used to greet with "What do you want to know about <project>?". That makes the agent sound like a Q&A bot, but it can also plan, spec and build issues in Work Mode. The greeting, the composer placeholder and one suggestion card now say so.

## The pieces

- `web/src/components/Agent/AgentView.tsx`
  - The heading is now `What should we build in {projectName || "this project"}?`.
  - The empty-state placeholder is now a fixed string: "Ask a question, or describe what to build…". The project name was dropped from it because the heading right above already names the project. Once a chat has messages, the placeholder is still "Reply…".
  - `SUGGESTION_ICONS[1]` is now `Hammer` instead of `Workflow`, to match the new card. The `Workflow` import is gone.
- `web/src/components/Agent/agent-state-utils.ts`: in `SUGGESTIONS`, "What's blocking an epic?" is replaced by "Plan and build a feature". Its prompt asks the agent to interview the user, shape the idea into an issue with a spec, and then offer to start work. That follows the normal xpo flow, so the user still approves the spec and the start.

## Decisions

- **Which card goes:** "What's blocking an epic?" was the most niche of the three. "File a bug" and "Summarize my open issues" are the everyday entry points, so they stay.
- **Shared constant:** `ChatPopup` renders the same `SUGGESTIONS`, so the docked popup gets the build card too. That is intended. The popup's own placeholder ("Ask about issues, or the code behind them…") did not change.
- **Icons go by position:** `SUGGESTION_ICONS` is matched to `SUGGESTIONS` by index. If you reorder one, reorder the other.

## Acceptance criteria

- [x] The empty Agent view heading reads "What should we build in <project>?". Covered by the test "greets with building, not just asking, on the new chat home".
- [x] The empty-state composer placeholder reads "Ask a question, or describe what to build…". Covered by the same test.
- [x] The middle suggestion card is "Plan and build a feature" with a hammer icon. Covered by the test "offers a build card between the bug and summary suggestions" (card order plus `lucide-hammer`). Clicking it sends `s.prompt` through the existing click handler, which was not changed.
- [x] A test covers the heading and the build card. Both tests were written first and failed before the change.
- [x] `make test` passes (exit 0: lint, 1117 frontend tests, Go tests).
- [x] The user tried the change and approved it.
