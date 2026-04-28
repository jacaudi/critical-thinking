package thinking

// ToolDescription is the verbatim description string registered on the
// criticalthinking MCP tool. Every agent calling the tool reads this — it is
// the prompt-engineering contract for rubber-duck narration + critical self-
// examination on top of sequential thinking.
//
// Treat changes here as protocol changes: bump the package version and
// document in the README migration notes.
const ToolDescription = `A tool for *critical*, *narrated*, *sequential* problem-solving — a rubber duck
you talk to while you think one step at a time. This tool fuses three disciplines:

  1. Sequential thinking — break the problem into ordered, numbered steps. Each
     thought builds on the previous ones; you can revise earlier steps and branch
     into alternatives when the path forks.
  2. Rubber-duck narration — explain each thought out loud, in first-person,
     to an imagined listener. The act of explaining surfaces the assumptions
     you didn't know you were making.
  3. Critical self-examination — every thought is paired with confidence,
     assumptions, critique, and a counter-argument. You produce the thought,
     then you interrogate it, in the same call.

Sequential thinking is the *spine*. Rubber-ducking is the *voice*. Critical
self-examination is the *check*. Skipping any one of them is a misuse.

How a session unfolds:

  - You start at thoughtNumber=1 with an estimated totalThoughts. Each call
    produces one thought + its critique. The next call builds on what came
    before — including, importantly, on what your own critique surfaced.
  - thoughtNumber and totalThoughts are OPTIONAL after the first call: omit
    thoughtNumber to let the server auto-assign the next sequential position;
    omit totalThoughts to inherit the value from the prior thought. Send them
    only when you need to override (revisions, branches, or a changed
    estimate). The first thought of a session must include totalThoughts.
  - When a critique reveals that an earlier thought was wrong, use isRevision
    + revisesThought to revisit it. The new critique should explain *why*
    the old one was wrong.
  - When the path forks and you want to explore an alternative, use
    branchFromThought + branchId together. Branches accumulate their own
    running confidence average — if a branch is averaging 0.4, that's a
    signal the path is shaky.
  - Adjust totalThoughts as understanding evolves (resend it when changed).
    Set nextThoughtNeeded=false only when the work is genuinely done.

For each call, you write ONE thought as if explaining it to a patient listener.
Then, in the same call, you provide:

  - confidence (0.0–1.0): How sure are you, *honestly*? 0.5 means a coin flip.
                          High confidence (>0.8) requires evidence, not enthusiasm.
  - assumptions (string[]): What are you taking for granted? List them as
                            bullets — each one ≤ 200 chars, one fact per entry.
                            Send [] only if you've genuinely accounted for none.
  - critique (string, required, non-empty, ≤ 280 chars):
                            What is weak, suspect, or under-examined about the
                            thought you just produced? One tight sentence naming
                            a SPECIFIC weakness. "Looks good" is not a critique;
                            neither is a paragraph of generic self-doubt.
  - counterArgument (string, required, non-empty, ≤ 280 chars):
                            The strongest case AGAINST this thought, in one
                            sentence. Steelman the opposition. If you can't
                            think of one, your confidence is wrong.
  - nextStepRationale (string, ≤ 200 chars, REQUIRED when nextThoughtNeeded=true,
                            OMIT when nextThoughtNeeded=false):
                            Why is *this* the next thought, not some other one?
                            One sentence: what this thought ruled out, opened up,
                            or exposed.

Brevity discipline: 'thought' is your narration; the critical fields are
bullets, not paragraphs. One tight sentence each, enforced server-side. A
specific weakness in 20 words beats vague self-doubt in 200.

Voice and register for the thought field:
  - First-person, narrative, exploratory. "I think... but wait... actually..."
  - Include hedges, false starts, and self-corrections — that's the rubber-duck
    register, not noise.
  - Address an imagined listener. The discipline of explaining out loud is what
    surfaces hidden assumptions.
  - This is NOT polished prose. Polished prose hides uncertainty. Be messy and
    honest.

Anti-patterns to avoid:
  - Producing thoughts in isolation, not building on prior steps. Sequential
    means each thought *uses* what came before — including your own prior
    critiques.
  - Boilerplate critique ("could be improved"). Be specific.
  - Confidence inflation. If everything is 0.9, the field is uninformative.
  - Skipping counterArgument by claiming there is none. There always is one.
  - Treating critique/counterArgument as paperwork. They exist to change your
    next thought, not to satisfy the schema.

What you get back:
  - A narrated transcript of this thought (rendered in rubber-duck voice).
  - Running session confidence (mean of trunk thoughts).
  - Per-branch confidence (if any branches exist).
  - thoughtHistoryLength and the list of branch ids.

The response deliberately does NOT echo thoughtNumber, totalThoughts, or
nextThoughtNeeded — you sent those, you already have them. Track session
position locally; the server is the source of truth via the
thinking://current resource.

Use this when the problem deserves slow, examined, multi-step thinking. Don't
use it for trivia or one-step lookups — the ceremony will get in the way. Use
it when being wrong is expensive.`
