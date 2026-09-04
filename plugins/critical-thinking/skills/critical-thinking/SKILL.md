---
name: critical-thinking
description: Mandatory critical-thinking gates that verify intent before acting and verify results before responding, using the criticalthinking MCP tool. Always on for substantive prompts.
disable-model-invocation: false
user-invocable: true
---

# Critical Thinking Verification

**Every substantive prompt gets two verification gates through the `criticalthinking` tool. No exceptions.**

A substantive prompt involves action, analysis, or a decision. Trivial acknowledgements ("thanks", "got it") are exempt.

`criticalthinking` means the tool whose leaf name is `criticalthinking`, whatever server prefix your host adds. Under this plugin's own stdio registration it appears as `mcp__plugin_critical-thinking_critical-thinking__criticalthinking`; behind a gateway it carries a different prefix. Match the leaf name.

## Gate 1 — intent, before acting

After reading the prompt and whatever orientation the task needs (files, docs, recent history), and **before any edit, state-changing command, or answer**, run a `criticalthinking` sequence that:

1. Restates the ask in your own words
2. Separates the real ask from the stated ask (they often differ)
3. Names the assumptions you are about to make
4. Flags ambiguities that could send you the wrong way
5. Decides whether to ask a clarifying question or proceed

Only then begin the work.

## Gate 2 — result, before responding

After the work is done — or, when Gate 1 decided to ask a clarifying question, once that question is drafted — and **before presenting anything**, run a `criticalthinking` sequence that:

1. Checks the result answers what was asked, not what you assumed
2. Looks for logic errors, missed requirements, and drift from the original intent
3. Confirms completeness — the full scope, not only the easy part
4. Names anything uncertain or unfinished
5. Decides what caveats or follow-ups the answer needs

Use extended thinking throughout both gates and the work between them. The gates are checkpoints; the thinking is continuous.

## The tool is stateless

`criticalthinking` validates and narrates the one thought you send and remembers nothing: no history, no running confidence, nothing to isolate. Nothing you send is carried into another call. (This describes the server this plugin installs; if your host points the tool at an older server, follow the tool description you are actually served.) Your own context is the record:

- Reread your earlier gate thoughts before writing the next one, and build on your own critiques.
- Keep `thoughtNumber` sequential yourself. Send `isRevision` and `revisesThought` together, and `branchFromThought` and `branchId` together, pointing only at thoughts you actually wrote.
- Judge your calibration across your own thoughts: if every confidence sits in 0.8–0.9, the field is telling you nothing.

## Tool failure protocol

**If `criticalthinking` is unavailable at any point — HALT IMMEDIATELY.**

"Unavailable" means no tool with the leaf name `criticalthinking` is in your tool list, or a call returns a connection or transport error. A schema-validation error is NOT unavailability; that is a caller bug to fix and retry.

Do not: continue without verification; treat your own reasoning as "good enough"; mention the failure in passing and proceed; retry silently and pretend it worked; fall back to `mcp__sequential-thinking__sequentialthinking` or plain prose.

Do: stop all work in progress; tell the user "The criticalthinking tool is unavailable. I cannot verify my understanding/results without it. How would you like to proceed?" — offering to proceed without it; wait for direction. If the user then tells you to proceed without the tool, do so: that waiver holds for the rest of the conversation unless they revoke it, and every answer you give ungated says so plainly.

This is a hard stop, not a soft warning. Unverified work causes bugs.

## Scaling the gates

| Prompt | Each gate |
|---|---|
| Simple (rename, typo fix, factual answer, short edit) | 2–3 thoughts |
| Medium (feature, bug fix, a document or analysis with several parts) | 5–7 thoughts |
| Complex (architecture, multi-file change, research with competing options) | 10+ thoughts |

These are guides, not quotas — count thoughts, not questions: one thought may answer several of a gate's five questions. End a gate as soon as they are honestly answered: set `nextThoughtNeeded=false` and move on. Simple prompts still get both gates, briefly — including a bare approval or continuation ("yes", "go ahead"), where Gate 1 restates what is being approved.

## Relationship to sequential-thinking

`criticalthinking` keeps sequential-thinking's step structure (numbered thoughts, revisions, branches) and adds the discipline the tool enforces: `confidence`, `assumptions`, `critique`, and `counterArgument` on every thought, plus `nextStepRationale` whenever `nextThoughtNeeded=true`. Unlike sequential-thinking, it keeps no history of its own. When this skill is active, use `criticalthinking` for both gates; do not fall back to `mcp__sequential-thinking__sequentialthinking`, which has the weaker contract.
