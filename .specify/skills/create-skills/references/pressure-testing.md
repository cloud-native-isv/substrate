# Pressure Testing Skills (RED-GREEN)

Method adapted from superpowers' `writing-skills` TDD discipline (skill creation treated as RED-GREEN-REFACTOR) to Spec Kit's subagent dispatch model. Applies to `create-skills` (before Register) and to `improve-skills` (for behavior-changing edits).

## Why

A skill that reads well can still fail under pressure: agents rationalize around rules ("just this once", "the user is in a hurry", "this case is different"). Static validation (frontmatter, size, paths) cannot catch this — only observing an agent attempt the task can.

## Method

### 1. RED — run the scenario WITHOUT the skill

- Construct one realistic **pressure scenario**: a task the skill should govern, plus a stressor (time pressure, a plausible shortcut, an instruction that tempts rule-bending).
- Dispatch a fresh subagent (no access to the skill body) with the scenario.
- **Record verbatim** what it does wrong and every rationalization it produces. These are the failure modes the skill must close.
- If the subagent already behaves correctly without the skill, the skill may be unnecessary — surface this to the user before continuing.

### 2. GREEN — run the same scenario WITH the skill

- Dispatch a fresh subagent whose prompt includes the skill body (or instructs it to load the skill).
- Verify it now complies: the RED failure modes do not recur.
- A pass requires observed compliance, not a plausibility argument.

### 3. REFACTOR — close loopholes

- For each RED rationalization the GREEN run did not fully suppress, add an explicit counter to the skill (a MUST/MUST NOT line, a red-flag phrase list, or a rationalization table entry).
- Re-run GREEN until clean or the remaining gap is explicitly accepted by the user.

## Dispatch template

```
Subagent prompt (RED):  <scenario task + stressor>. Do NOT load any skill.
Subagent prompt (GREEN): <same scenario>. First read and follow: <skill body or path>.
```

Use a read-only/worktree-isolated subagent when the scenario would otherwise write files.

## Scope & waiver

- MANDATORY for new discipline/workflow skills (skills that constrain agent behavior).
- OPTIONAL for pure utility skills (wrappers around deterministic scripts) — static validation plus a smoke invocation suffices; state which case applies in the report.
- The user may waive the pressure test explicitly; record the waiver in the creation report.

## Record

Append to the skill creation/improvement report: scenario used, RED failures (verbatim), GREEN outcome, loopholes closed. One scenario well-observed beats five imagined.
