package thinking

// ToolDescription is the verbatim description registered on the
// criticalthinking MCP tool. Every agent calling the tool reads it — it is the
// prompt-engineering contract for narrated, critically examined, sequential
// thinking. Treat changes here as protocol changes: add a migration note.
const ToolDescription = `A tool for critical, narrated, sequential problem-solving: one thought per
call, said out loud in first person, and interrogated in the same call.

Three disciplines, all required:
  1. Sequential — number your thoughts and build each on the ones before.
     Revise an earlier one (isRevision + revisesThought) when a critique shows
     it was wrong; branch (branchFromThought + branchId) when the path forks.
  2. Out loud — write the thought in exploratory, first-person voice. Hedges,
     false starts, and self-corrections belong in it: putting half-formed
     reasoning into words is itself the check on it. Not polished prose.
  3. Critical — pair every thought with confidence, assumptions, critique, and
     a counter-argument, then let them change the next thought.

Required fields — ` + requiredFieldsChecklist + `

Stateless: this tool keeps nothing between calls. It validates and narrates
the one thought you send; it has never seen your earlier ones.
Your own context is the record, so open each thought after the first by
restating what the previous one concluded and what its critique changed, and
set confidence by comparing it with the confidences you already assigned: if
they all sit in
0.8–0.9, the field is telling you nothing.

Fields:
  - thought: one thought, first person, exploratory.
  - thoughtNumber / totalThoughts (both ≥ 1): where you are and your current
    estimate of the whole. Adjust totalThoughts as understanding evolves; if
    thoughtNumber exceeds it, the server raises totalThoughts to match.
  - nextThoughtNeeded: false only when this line of thinking is genuinely done.
  - nextStepRationale (required when nextThoughtNeeded=true): why THIS is the
    next thought — what this one ruled out, opened up, or exposed.
  - confidence (0.0–1.0): how sure you are, honestly. 0.5 is a coin flip;
    above 0.8 needs evidence, not enthusiasm.
  - assumptions (string[]): what you are taking for granted, one per entry,
    none blank. Send [] only if you genuinely claim there are none.
  - critique: what is weak, suspect, or under-examined in this thought.
    "Looks good" is not a critique.
  - counterArgument: the strongest case against this thought. If you cannot
    find one, your confidence is wrong.
  - isRevision + revisesThought (always together): this thought corrects an
    earlier one; say in the critique why the earlier one was wrong.
  - branchFromThought + branchId (always together): this thought explores an
    alternative from an earlier point; keep one branchId per alternative.

Anti-patterns:
  - Thoughts that ignore prior steps. Sequential means each thought uses what
    came before, including your own earlier critiques.
  - Boilerplate critique ("could be improved"). Be specific.
  - Confidence inflation.
  - Claiming there is no counter-argument. There always is one.
  - Treating critique and counterArgument as paperwork. They exist to change
    your next thought, not to satisfy the schema.
  - Opening a thought after the first without saying what the previous one
    concluded and what its critique changed.

Returns the narrated transcript of this thought plus its routing fields
(thoughtNumber, totalThoughts, nextThoughtNeeded, confidence) echoed back.

Use this when being wrong is expensive and the problem deserves examined,
multi-step thinking. When a host protocol asks for it on every prompt, follow
the protocol and keep the sequence as short as the question honestly allows.`
