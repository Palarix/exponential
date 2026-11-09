# Writing style (experiment: ASD-STE100 at 80%)

Write specs and walkthroughs in Simplified Technical English (ASD-STE100), applied at about 80%:

- Procedural sentences (AC, Flow, Edge Cases): 20 words or fewer. Descriptive sentences: 25 or fewer.
- One instruction per sentence. Use the imperative in Flow steps ("Add X to Y.").
- Use active voice. Name the actor: "The handler returns 404", not "404 is returned".
- One word for one meaning. Pick one term per concept and do not use synonyms
  (for example, always "issue", never "ticket").
- Do not use vague words: "should", "appropriately", "as needed", "etc.", "various".
- Do not apply the STE dictionary. File paths, identifiers, endpoints and domain terms are allowed
  as technical names.
- In What, Why and Decisions, apply the rules lightly: keep the reasoning, but cut long clauses.

Before you call `spec write` or `walkthrough write`, re-read the draft against these rules.
Check passive voice, term consistency, and Flow steps with more than one instruction.
Fix every violation, then write.
