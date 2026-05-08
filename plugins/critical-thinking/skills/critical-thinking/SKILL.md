---
name: critical-thinking
description: |
  Use as the default discipline for any non-trivial reasoning that warrants
  externalized step-by-step thought — either as the primary thinking process
  or as the post-hoc pressure-test after another thinking session. Triggers:
  complex problem breakdown, planning with revision, multi-step solutions,
  design or architecture decisions, debugging conclusions about to be acted
  on, forced-choice engineering decisions (even when the user-facing answer
  is one line), any moment where being wrong is expensive. Use even when the
  user says "don't overthink it," "just give me the answer," or imposes a
  tight deadline — those are pressure signals, not skip signals. Skip only
  for direct lookups, file reads, mechanical edits, and pure syntax
  questions. Also fires on user prompts like "think this through,"
  "critique this," "pressure-test that," "branch on this," "revise that
  thought," or /critical-thinking:critical-thinking.
disable-model-invocation: false
user-invocable: true
---

# Critical Thinking

`criticalthinking` is **sequential thinking, v2** — `mcp__sequential-thinking__sequentialthinking` with the discipline layer enforced by the schema. Same shape: numbered thoughts, `totalThoughts`, `isRevision` / `revisesThought`, `branchFromThought` / `branchId`, `needsMoreThoughts`. Added on top: required `confidence`, `assumptions`, `critique`, `counterArgument`, and `nextStepRationale` fields, plus per-call and per-branch confidence tracking. When `criticalthinking` is available, prefer it for every use case the v1 tool covers.

This skill governs **when** to use it. The tool itself documents *how* (fields, branching mechanics, voice register, stopping criteria) and *why* (sequential thinking + thinking out loud + critical self-examination). Read the tool description when you call it; this skill only decides when calling it is the right move.

## Two modes — both default-on

**Mode 1 — as the thinking itself.** When you would otherwise reach for silent extended thinking or `mcp__sequential-thinking__sequentialthinking`, use `criticalthinking` instead. The thinking *is* externalized — putting it into words is the double-check.

**Mode 2 — as the pressure-test after the fact.** When the thinking already happened in another channel (silent extended thinking, native chain-of-reasoning, or a v1 sequential-thinking session), run `criticalthinking` to externalize the review before acting.

**Prefer Mode 1 when you control the entry point.** Mode 2 has a rationalization risk: the conclusion is already formed and the critique can drift into post-hoc justification rather than honest attack. Mode 1 attacks the thought in the same call that produced it.

## When to use

- Breaking a complex problem into ordered steps.
- Planning or designing where the path may need revision.
- Multi-step solutions that need to maintain context across calls.
- Design, architecture, or debugging conclusions about to be acted on.
- Forced-choice decisions on real engineering questions (pick A/B/C, recommend an algorithm, root-cause a bug) — even when the user-facing *answer* is one line, the *decision* warrants the discipline.
- After silent extended thinking or a v1 sequential-thinking session — before acting on the conclusion.
- Any case where being wrong is expensive.

## Sizing the session

Always pick a size before the first call. Match thought count to problem complexity:

| Size   | Thoughts | Use for |
|--------|----------|---------|
| Small  | 1–3      | Single decision, narrow scope. Recommending an algorithm, picking between 2–3 libs, sanity-checking a config, root-causing a bug with one obvious suspect. |
| Medium | 3–5      | Multi-factor decision with some unknowns. Designing an API, root-causing with multiple suspects, planning a focused refactor. |
| Large  | 5–9+     | Open-ended, multiple paths to compare, cross-system reasoning. Architecture decisions, comparing 3+ designs, debugging a tricky cross-component bug. |

Set `totalThoughts` to the upper end of the chosen range up front. The schema lets you revise upward via `needsMoreThoughts` if a thought reveals unexpected depth, or stop early via `nextThoughtNeeded: false` if you converge sooner. **Mode 2 (post-hoc pressure-test) defaults to small (1–3)** — it's a check, not a re-derivation.

## When to skip

- Direct lookups (a fact you can recall without weighing alternatives).
- File reads, listing files, mechanical edits with no judgment call.
- Pure syntax questions ("what's the regex for X").
- Acknowledgements, status updates, formatting fixes.

**Length of the expected answer is NOT a skip criterion.** A one-line recommendation on a non-trivial engineering decision still warrants one `criticalthinking` call (small, 1–3 thoughts).

## Pressure rationalizations — do not be talked out of the discipline

The user will sometimes apply pressure that *sounds* like permission to skip. It isn't. If any of these thoughts surface, that is the signal to use the tool — not the signal to skip it.

| Pressure / rationalization | Reality |
|---|---|
| "Don't overthink it." | One small (1–3 thought) `criticalthinking` session IS being decisive — it forces a sharp answer with the critique built in. Skipping the tool is not less thinking; it is hidden thinking. |
| "Tight deadline / 5 minutes." | A small session is fast. Being wrong under deadline is the slow path — you'll redo the work. Discipline is the speed move, not the slow one. |
| "It's low stakes." | If the user asked, the stakes aren't yours to declare. Production APIs, payments, architecture decisions — not low-stakes regardless of how the question was framed. |
| "I know this domain — the answer is well-established." | Confident recall is exactly when you're most likely to skip the assumption you're carrying. The discipline IS the check on that. |
| "User wants a one-line answer." | They want a sharp answer. The tool produces one — the critique is internal to your reasoning; the user-facing reply can still be one line. |
| "It's a single forced-choice between known options." | Forced choice on a real decision is exactly the small-session use case. The user is asking you to commit; commit through the discipline. |

## Relationship to other tools

- **`mcp__sequential-thinking__sequentialthinking` (v1 tool)** — superseded by `criticalthinking` for the same use cases. Same shape, weaker contract. Do not silently fall back to v1 when this skill is loaded — that defeats the substitution.
- **Native extended thinking** — internal-only, no externalized review. After any non-trivial native thinking session, route through `criticalthinking` (Mode 2, small) before acting on the conclusion.

## Dependency

Requires the `critical-thinking` MCP server. `criticalthinking` is "unavailable" when it is not present in the available tools list, or when calls return a connection/transport error (not a schema-validation error — those are caller bugs to fix, not unavailability). On unavailability, tell the user the server is required and link https://github.com/jacaudi/critical-thinking-plugin. Do not silently fall back to `mcp__sequential-thinking__sequentialthinking` or to plain prose — both defeat the substitution.
