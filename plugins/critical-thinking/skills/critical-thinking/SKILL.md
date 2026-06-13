---
name: critical-thinking
description: Mandatory critical-thinking gates that verify intent before work and verify results before responding, using the criticalthinking MCP tool. Always on for substantive prompts.
disable-model-invocation: false
user-invocable: true
---

# Critical Thinking Verification

**Every substantive prompt requires two verification gates using the `criticalthinking` tool. No exceptions.**

This rule applies to all prompts that involve action, analysis, or decision-making. Trivial acknowledgements ("thanks", "got it") are exempt.

The tool's fully-qualified id under this plugin is `mcp__plugin_critical-thinking_critical-thinking__criticalthinking`; it is referred to as `criticalthinking` below.

---

## Gate 1: Intent Verification (Before Starting Work)

After receiving a user prompt, **before doing anything else**, call `criticalthinking` to:

1. Restate what the user is asking in your own words
2. Identify the real ask vs. the stated ask (they often differ)
3. Surface assumptions you're about to make
4. Flag ambiguities that could send you in the wrong direction
5. Decide whether to ask clarifying questions or proceed

**Use extended thinking throughout this gate.** Think through edge cases, implicit requirements, and unstated context.

Only after this gate passes should you begin work (research, coding, planning, etc.).

---

## Gate 2: Result Verification (Before Responding)

After completing work but **before presenting results to the user**, call `criticalthinking` to:

1. Verify the result actually answers what was asked (not what you assumed)
2. Check for logical errors, missed requirements, or drift from the original intent
3. Confirm completeness — did you address the full scope or only part?
4. Identify anything that should be flagged as uncertain or incomplete
5. Decide if the answer needs caveats or follow-up suggestions

**Use extended thinking throughout this gate.** Challenge your own work before presenting it.

---

## Extended Thinking: Use Liberally

Between the two gates, continue to use extended thinking for any non-trivial reasoning step. The gates are checkpoints; thinking is continuous.

---

## Tool Failure Protocol

**If `criticalthinking` is unavailable at any point — HALT IMMEDIATELY.**

"Unavailable" means the tool is absent from the available tools list, or a call returns a connection/transport error. A schema-validation error is NOT unavailability — that is a caller bug to fix.

Do not:
- Silently continue without verification
- Substitute your own internal reasoning as "good enough"
- Mention the failure in passing and proceed anyway
- Retry silently and pretend it worked
- Silently fall back to `mcp__sequential-thinking__sequentialthinking` or plain prose

Do:
- Stop all work in progress
- Tell the user explicitly: "The criticalthinking tool is unavailable. I cannot verify my understanding/results without it. How would you like to proceed?"
- Wait for user direction before continuing

**This is a hard stop, not a soft warning.** The verification gates exist because unverified work causes bugs. Skipping them silently defeats the purpose.

---

## Workflow Summary

```
User prompt received
  │
  ▼
GATE 1: Intent Verification  (criticalthinking → understand; extended thinking ON)
  │  (If tool unavailable → HALT, alert user)
  ▼
WORK: Execute the task       (extended thinking throughout)
  │
  ▼
GATE 2: Result Verification  (criticalthinking → validate; extended thinking ON)
  │  (If tool unavailable → HALT, alert user)
  ▼
Present verified results to user
```

---

## Scaling

| Complexity | Gate 1 Depth | Gate 2 Depth |
|-----------|--------------|--------------|
| Simple (rename, typo fix) | 2-3 thoughts | 2-3 thoughts |
| Medium (feature, bug fix) | 5-7 thoughts | 5-7 thoughts |
| Complex (architecture, multi-file) | 10+ thoughts | 10+ thoughts |

Scale verification depth to task complexity. Simple tasks still get verified — just briefly.

---

## Relationship to sequential-thinking

`criticalthinking` is sequential-thinking with the discipline enforced by the schema (required `confidence`, `assumptions`, `critique`, `counterArgument`, `nextStepRationale`). When this skill is active, use `criticalthinking` for both gates — do not fall back to `mcp__sequential-thinking__sequentialthinking`, which has the weaker contract.
