# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes and ensure clean coordination.
Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution, clean context, and deliberate delegation over speed.
For trivial tasks, use judgment.

## 0. Context Hygiene & Delegation

**Keep the main session conversation clean. The main agent coordinates; sub-agents execute.**

- The main agent should **never** directly perform exploration, implementation, debugging, or large-scale refactoring inside the main conversation.
- Instead, delegate these tasks to **sub-agents** via tool calls:
  - **Exploration / Research / Simple lookups** → delegate to a **Haiku** sub-agent (fast, cheap).
  - **Standard implementation, bug fixes, moderate refactoring** → delegate to a **Sonnet** sub-agent (default workhorse).
  - **Complex design, architecture planning, multi-step reasoning, or high-stakes decisions** → delegate to an **Opus** sub-agent (thorough, accurate).
- The main agent’s role is:
  - Clarifying goals and defining success criteria.
  - Deciding which sub-agent to spawn for each task.
  - Verifying sub-agent outputs and integrating results.
  - Committing completed work.
- Never dump exploratory logs, debugging dumps, or speculative code directly into the main conversation. Sub-agents do the messy work in isolation; only the final, reviewed result is presented.

**Git rhythm:** Commit after **each completed task** (not mid-task). One logical change = one commit with a clear message.

---

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing (in a sub-agent or planning the next step):
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

---

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

---

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

---

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let sub-agents loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** the main conversation stays clean, sub-agents produce verified results with minimal back-and-forth, commits are small and logical, and clarifying questions come before implementation rather than after mistakes.