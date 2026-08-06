# Interview Pattern (访谈模式) — A Design Pattern for Commands, Skills & Agents

A reusable design pattern for any unit whose job is to extract information that **only the user has**, through many rounds of live interaction, while **continuously writing every answer through to a target artifact**, until the user confirms a stable shared understanding.

Where [`reconcile-pattern.md`](reconcile-pattern.md) converges a *durable artifact space* toward a describable ideal form, this pattern converges a *shared understanding* between the agent and the user. The information source is the user's head; the terminator is the user's confirmation.

## When to Apply (适用判据)

Apply this pattern when **all** of these hold:

1. The needed information is **tacit** — it lives in the user's head (intent, preference, priority, trade-off acceptance) and is **not derivable** from the repo, the filesystem, the tools, or the web.
2. The decisions **branch**: an answer changes which questions are even askable next, so a fixed questionnaire cannot cover the space.
3. There is a **target artifact** that must absorb every answer and stay current — the session produces a converged document, not a chat transcript.
4. A **live user** is answering. The mode is human-in-the-loop by construction.

Do **not** apply to: facts the agent can look up itself (that is the agent's job — see [Fact vs Decision Split](#fact-vs-decision-split-事实归我决策归你)); a single preview→confirm gate; read-only analysis or reporting; unattended / fire-and-forget runs.

## Boundary: What This Is Not (概念消歧)

This pattern is easily confused with four neighbours that also involve "repeated rounds + LLM judgment + a converging artifact". The root differences are **where the information comes from** and **what ends the loop (信息来源 + 终止语义)**:

| Mode | Information source | Terminates when | Human position |
|------|--------------------|-----------------|----------------|
| **Interview** (this pattern) | The **user's tacit knowledge** | The frontier is empty **and** the user confirms the understanding is stable | Answers every round; the loop cannot advance without them |
| **Reconcile** ([`reconcile-pattern.md`](reconcile-pattern.md)) | A durable **artifact space** with a describable ideal form | Current state matches desired state within the tolerance band | Present per invocation, gating destructive actions |
| **Iteration loop** ([`operating-loops.md`](../../skills/create-team/references/operating-loops.md)) | The **agents' own output** scored against quality dimensions | A quality threshold is reached, or `max_iterations` is hit | Sets the threshold, then largely observes |
| **Continuous loop** (same reference) | An unbounded **stream of incoming work** | Never — it operates on a cadence | May be absent at L3 (kill-switch/budget instead) |
| **Confirmation gate** | Already-computed agent output | One preview→confirm exchange | Approves or rejects once |

**判定口诀 (rule of thumb)**: 信息只在用户脑子里、且答案会长出新问题 → interview；有可描述的制品理想形态要反复收敛 → reconcile；靠打分逼近阈值 → iteration；源源不断的工作流 → continuous；只需一次点头 → confirmation gate。

**Nesting, not competitors (嵌套关系，不是竞品)**: an interview **feeds** downstream work — its converged artifact becomes the input to a reconcile run, a plan, or a team's execution. It never replaces them, and they never replace it: no amount of reconciling or scoring can recover a decision the user was never asked about.

**Near-synonyms not to conflate (容易张冠李戴的近义概念)**:
- 前沿 frontier (interview: which questions are askable *now*) ≠ 容忍带 tolerance band (reconcile: which diffs are worth converging)
- 用户确认稳定 (interview: the exit gate) ≠ 质量阈值 (iteration: a score) ≠ 一次性预览确认 (confirmation gate)
- 访谈台账 interview ledger (which branches were decided, by whom, when) ≠ audit log (reconcile: what was written) ≠ STATE.md (continuous: cross-run memory)

## Core Model (核心模型)

- **Design tree (决策树)** — the decision space as a tree: every decision branches into the decisions that hang off it. Answering "use a queue" creates "which queue", "what retry policy", "what dead-letter behavior"; those questions did not exist before the answer.
- **Frontier (前沿)** — every decision whose prerequisites are **already settled**: the questions that can be asked *now* without guessing at answers not yet heard. A question that depends on another question still open belongs to a **later round**, not this one.
- **Round (轮次)** — ask the **whole frontier at once**, numbered, each with a recommended answer; then **wait**. One question per round when the frontier is wide wastes the user's time; asking a blocked question forces the agent to guess.
- **Write-through (即答即落)** — every round's answers land in the target artifact before the next round is asked. The artifact, not the conversation, is the deliverable.

## The Interview Loop (访谈环 I0–I5)

```
- [ ] I0 Resolve target + ledger; declare conventions once; resume if a ledger exists
- [ ] I1 Seed / refresh the design tree            → artifact: design tree
- [ ] I2 Compute the frontier
- [ ] I3 Dispatch fact-finding probes (non-blocking)
- [ ] I4 Ask the whole frontier in one round, then WAIT
- [ ] I5 Write through to the artifact + ledger    → artifacts: target artifact, ledger row
        └─ frontier non-empty → back to I2
- [ ] Exit gate: present the consolidated understanding → explicit user confirmation
```

- **I0 — resolve & resume**: identify the **target artifact** (what converges) and the **ledger location** (see [State & Resumability](#state--resumability-状态与续跑)). Declare the session's conventions **once** — granularity, how decisions are recorded, the done-criterion for a branch — so they are not renegotiated every round. If a ledger already exists, read its header conventions (**never renegotiate them**) and resume from the first unsettled branch.
- **I1 — seed the tree**: build the initial decision tree from the target artifact's current content, a scan of the surrounding repo/context, and this run's user input. Mark what is already settled — never re-ask a decision the artifact already records.
- **I2 — compute the frontier**: select every decision whose prerequisites are settled. Defer the rest explicitly; a deferred question is *scheduled*, not dropped.
- **I3 — dispatch probes (non-blocking)**: when a frontier question needs an **environment fact**, dispatch a subagent to find it instead of asking the user. Do **not** block the round: a running probe is an unsettled prerequisite, so only the questions downstream of it wait — ask the rest of the frontier now.
- **I4 — ask one round, then wait**: present the whole frontier in the [question format](#question-format-提问格式契约) below. Then **stop and wait**. Do not answer on the user's behalf; do not proceed on silence.
- **I5 — write through**: record each decision into the target artifact **overwrite-style — the latest round wins**, so the artifact never accumulates contradictory rounds. Update the ledger row. Then recompute the frontier (`I2`): each answer reshapes the tree, pushing the frontier outward and unblocking what depended on it.
- **Exit gate**: the session is done when the frontier is empty — every branch visited, nothing silently assumed. Present the consolidated understanding and obtain **explicit user confirmation**. **Never act on the design before that confirmation**; an empty frontier is the agent's opinion, the confirmation is the user's.

### Question Format (提问格式契约)

Every question carries a number, a title, and a **recommended answer**. The recommendation is mandatory — it converts an interrogation into a review, which is faster and higher-fidelity:

```
❓ **Q1** — **<question title>**: <question body; may be several paragraphs and may enumerate options>

➡️ <your recommended answer, with the reason it is recommended>
```

Each question must be answerable as a multiple choice (2–5 options) or a short answer. When the host environment offers a structured question tool, use it and keep this format as the fallback.

### Fact vs Decision Split (事实归我，决策归你)

The load-bearing rule of the pattern:

- **Facts are the agent's job.** Anything discoverable — file contents, existing conventions, dependency versions, whether an endpoint exists, what the docs say — is looked up, never asked. Asking the user to be a lookup service is the fastest way to lose their attention.
- **Decisions are the user's.** Intent, priority, trade-off acceptance, naming preference, scope boundaries. Put each one to them and wait. Inferring a decision because it "seems obvious" is fabrication with extra steps.

When a question mixes both, split it: look the fact up, then ask only the decision that remains.

## State & Resumability (状态与续跑)

The **ledger** is the durable state that lets one interview span sessions, days, or agents. Place it nearest to the artifact it serves:

| Context | Ledger location |
|---------|-----------------|
| Inside a feature / spec workflow | `.specify/specs/<feature>/interview-log.md` |
| No feature context | `.specify/memory/session/` (per the memory-as-files layer) |
| The host defines its own ledger | Use it — never open a second parallel ledger |

Rows are **structured columns**, not free text, so residuals can be counted and queried at closing:

```markdown
# Interview Ledger: <topic>

- **Started**: <date>
- **Target artifact**: <path>
- **Granularity**: <one branch per round | …>
- **Recording**: overwrite-style; latest round wins
- **Done criteria**: <what makes a branch settled>
- **Status legend**: ⬜ open / 🔄 asked, awaiting answer / ✅ settled / ⏭ deferred-out-of-scope

| Round | Branch | Question | Status | Decision | Artifact touched |
|-------|--------|----------|--------|----------|------------------|
| 1 | <branch> | <Q title> | ⬜ | | |
```

Operating rules:

- **Checkpoint at round boundaries only.** When turns or time run out, stop after a completed round — never mid-round with answers unrecorded. End every session with the ledger fully written; start every session by re-reading it.
- **Unseen branches stay ⬜.** Skipping ahead and inventing the user's answer is the one unrecoverable failure mode of this pattern: the artifact then looks converged while encoding decisions nobody made.
- **Deferred is explicit.** A branch ruled out of scope is ⏭ with the reason, not silently dropped.

## Embedding Contract (宿主接入契约)

A unit claiming this mode must declare four things: the **target artifact**, the **ledger location**, the **exit gate**, and its **round budget** (unbounded, or a cap with an escalation path). Per host type:

| Host | Declares | Additional obligation |
|------|----------|-----------------------|
| **Command** | An `## Interview Mode` section naming the pattern, the target artifact, the ledger, and the exit gate | If it caps rounds/questions (like a bounded clarification pass), name the escalation path to an unbounded interview |
| **Skill** | The same four items in its `SKILL.md` | A **non-interactive escape hatch**: when no live user is available, degrade to a declared default and say so — never fabricate answers |
| **Agent / team member** | The mode in its workflow section, plus the human-in-the-loop constraint | Which team pattern it runs under, and what a fire-and-forget subagent invocation is allowed to do (process the requested batch against the durable ledger, then return) |

A host may **narrow** the pattern (fewer rounds, a fixed taxonomy of questions) but may not drop the write-through rule, the fact/decision split, or the user-confirmation exit gate.

## Adoption Map (接入图谱)

Where the pattern already lives in this project, and where it fits:

| Unit | Relationship |
|------|--------------|
| [`interview-walkthrough.md`](../workflow/interview-walkthrough.md) | **Instance** — the requirements domain: interview units, decision cards, verification levels, delivery-loop coupling |
| `/speckit.interview` | **Instance** — the standalone, general-purpose entry for running the pattern on any target artifact |
| `/speckit.clarify` | **Bounded instance** — a capped ambiguity pass over the current phase artifact; escalates to `/speckit.interview` when the space branches |
| `/speckit.goal` (`create`) | **Bounded instance** — a three-item elicitation (objective / criteria / identity) using this question protocol |
| `/speckit.plan` | **Opt-in escalation** — when design decisions are underdetermined, interview instead of fabricating a design |
| `create-team` | **Opt-in escalation** — eliciting a missing goal, and deriving roster/pattern when the goal underdetermines them |
| `summarize-project`, `create-skills`, `study-project` | **Pre-existing narrower loops** — gate-by-gate confirmation, one-question-per-round authoring, ≤3-per-round scoping. Conformant as-is; cite the pattern when they are next revised |
| `/speckit.analyze`, `/speckit.review`, `/speckit.implement`, `/speckit.history`, `/speckit.feature` | **Bad fit — do not adopt.** Read-only diagnosis, mechanical execution, or automated distillation: there is no tacit information to elicit |

## Anti-patterns (反模式)

- **Interrogating for discoverable facts**: asking the user what the code, config, or docs already say.
- **Drip-feeding**: one question per round while the frontier holds five askable questions.
- **Asking blocked questions**: putting a question whose prerequisite is still open, then guessing the prerequisite to interpret the answer.
- **Accumulating without writing through**: banking several rounds of answers "to integrate later" — the artifact drifts from the conversation and the session becomes unresumable.
- **Append-only recording**: stacking every round into the artifact so it holds mutually contradictory decisions. Overwrite; latest round wins.
- **Self-certified completion**: declaring shared understanding and acting on it without the user's explicit confirmation.
- **Fabricated answers**: filling in unseen branches to make the artifact look finished.
- **Unattended interviewing**: running the mode with no live user instead of degrading to the declared non-interactive default.
