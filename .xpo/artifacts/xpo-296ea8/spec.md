## What

Reword the empty Agent view so it says the agent can build things, not just answer questions.

## Why

"What do you want to know about <project>?" makes the agent sound like a Q&A bot. It can also plan, spec and implement issues in Work Mode. The placeholder and all three suggestion cards say the same thing: questions and triage only.

## How

1. `AgentView.tsx:184` heading: `What should we build in {projectName || "this project"}?`
2. `AgentView.tsx:127` placeholder for the empty state: `Ask a question, or describe what to build…` (stays `Reply…` once there are messages).
3. `SUGGESTIONS` in `agent-state-utils.ts`: replace "What's blocking an epic?" with a build card:
   - label: `Plan and build a feature`
   - description: `Shape an idea into a spec'd issue, then build it`
   - prompt: `I want to build something new. Ask me what it is, help me shape it into an issue with a spec, then offer to start work on it.`
   - Card order: File a bug, Plan and build a feature, Summarize my open issues.
4. `SUGGESTION_ICONS[1]`: change `Workflow` to `Hammer` (lucide).

## Decisions

- **Which card goes:** "What's blocking an epic?" is the most niche of the three. Filing a bug and summarizing open issues are the everyday entry points.
- **Shared constant:** the docked `ChatPopup` uses the same `SUGGESTIONS`, so the new card shows up there too. The popup placeholder ("Ask about issues, or the code behind them…") stays as it is. Its compact context is mostly Q&A.

## Acceptance criteria

- [ ] The empty Agent view heading reads "What should we build in <project>?"
- [ ] The empty-state composer placeholder reads "Ask a question, or describe what to build…"
- [ ] The middle suggestion card is "Plan and build a feature" with a hammer icon, and clicking it sends the build prompt
- [ ] A test covers the heading and the build card
- [ ] `make test` passes
