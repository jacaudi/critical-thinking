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

How a session unfolds: each call produces one thought + its critique. The
next call builds on everything that came before — including, importantly, on
what your own prior critique surfaced. Sequential is not just numbering; it
means each thought *uses* what came before. See "Structural fields" below
for the mechanics (revisions, branches, position, totals).

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

Structural fields (control how this call relates to prior thoughts):

  - thoughtNumber (int, optional): Current position. Omit to let the server
                            auto-assign (trunk: history+1; branch: depth within
                            the branch). Send explicitly when you want to be
                            unambiguous, e.g. on revisions.
  - totalThoughts (int, required on the first trunk thought, optional after):
                            Your current estimate of how many thoughts you'll
                            need. Resend when revised; omit to inherit the prior
                            trunk thought's value. Adjust up if you realize more
                            is needed — the server auto-bumps if your
                            thoughtNumber exceeds totalThoughts.
  - nextThoughtNeeded (bool, required): true if you need another thought, even
                            when you thought you were done. Set false only when
                            you have a satisfactory answer.
  - isRevision (bool) + revisesThought (int): Use together when this thought
                            revises an earlier one. The critique on the revising
                            thought should explain *why* the previous thinking
                            was wrong — not just restate the new view.
  - branchFromThought (int) + branchId (string): Use together to fork into an
                            alternative path from a specific prior thought.
                            Branches accumulate their own running confidence
                            average; a branch averaging low confidence is a
                            signal the path is shaky.
  - needsMoreThoughts (bool, optional): Set true when you reach the end of your
                            estimate but realize more thinking is needed. This
                            is a self-signal — it doesn't itself extend the
                            session; raise totalThoughts to do that.

When to use this tool:
  - Breaking complex problems into ordered, examined steps
  - Multi-step solutions that need to maintain context across calls
  - Analysis that may need course correction as understanding deepens
  - Problems where the full scope isn't clear at the start
  - Decisions where being wrong is expensive
  - Situations where filtering irrelevant context matters

Process guidance:
  1. Start with an initial totalThoughts estimate; be ready to adjust.
  2. Question and revise previous thoughts when your critique surfaces a flaw
     (isRevision + revisesThought).
  3. Add more thoughts past the estimate when needed (set needsMoreThoughts and
     raise totalThoughts).
  4. Express uncertainty honestly via confidence, assumptions, and critique.
  5. Branch when paths diverge (branchFromThought + branchId); don't conflate
     alternatives onto the trunk.
  6. Generate a hypothesis, then verify it across subsequent thoughts.
  7. Filter irrelevant information — not every prior detail matters at every
     step.
  8. Set nextThoughtNeeded=false only when you have a satisfactory answer.

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
