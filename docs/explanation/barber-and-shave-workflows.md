# Barber and Shave Workflows

Why yaketyyak has two shaving workflows, and why one of them reviews code with a second, independent agent.

## Two workflows, two philosophies

yaketyyak ships with two Temporal workflows for shaving yaks:

| | BarberWorkflow (`s`) | ShaveWorkflow (`v`) |
|---|---|---|
| Approach | Fire-and-forget with CI gate | Quality gate before PR |
| Trigger | CI signal, g2g tag | Manual yak selection |
| Review | Human via PR feedback signal | Adversarial agent review |
| Iterations | Unlimited rework loop | Fixed max retries (default 3) |

The **BarberWorkflow** is the continuous barber: it waits for signals and processes yaks as they come. Its quality gate is CI — if CI passes, merge. If CI fails, re-dispatch.

The **ShaveWorkflow** is a focused, single-yak workflow: quality is front-loaded through validation and adversarial review *before* a PR is even opened. This catches issues early, when context is still fresh and no human reviewer time has been spent.

## What is the shave loop?

The shave loop is named after two things: the autonomous agent loop pattern from Open Ralph Wiggum, and the iterative "implement, test, refine" pattern formalized by the shave-specs skill.

```
┌──────────────────────────────────────────────────────┐
│                     ShaveWorkflow                    │
│                                                      │
│  ┌──────────┐   ┌──────────┐   ┌───────────────┐   │
│  │Implement │──▶│ Validate │──▶│   Review      │   │
│  │(agent)   │◀──│ go vet,  │   │ adversarial   │   │
│  │          │   │ go test  │   │ agent review  │   │
│  └──────────┘   └──────────┘   └───────┬───────┘   │
│       ▲                                │             │
│       │         ┌──────────┐          │             │
│       └─────────│  Issues  │◀─────────┘             │
│        (fix)    │  found   │   (fail)               │
│                 └──────────┘                        │
│                       │                             │
│                       │ (pass)                      │
│                       ▼                             │
│                 ┌──────────┐   ┌──────┐   ┌──────┐ │
│                 │Create PR │──▶│Watch │──▶│Merge │ │
│                 │          │   │  CI  │   │      │ │
│                 └──────────┘   └──────┘   └──────┘ │
└──────────────────────────────────────────────────────┘
```

Each iteration is self-contained: the agent receives the yak context (plus failure feedback from the previous iteration), implements, and commits. The workflow validates and reviews. If either fails, the next agent gets a fresh context with the list of issues — it never sees the previous agent's reasoning.

## Why adversarial review?

### The anchoring bias problem

When a coding agent implements a yak, it builds a mental model of the solution. If the same agent (or a human seeing its reasoning) reviews the code, they tend to agree with the approach — anchoring bias. They see what the implementer intended to do, not what the code actually does.

### How adversarial review works

The `ShaveAdversarialReview` activity dispatches a **fresh, independent agent** that receives:

1. The original yak context and acceptance criteria (the "brief")
2. The `jj diff` of all workspace changes

It does **not** receive:
- The implementer's chain of thought
- What the implementer intended
- Any context about the implementation approach or tradeoffs

The reviewer prompt explicitly directs adversarial thinking:

> Be adversarial: assume nothing, verify everything, find every flaw.

This breaks anchoring bias. The reviewer evaluates the code as a user or attacker would — without the implementer's assumptions.

### Concrete example

A yak asks for "input validation on the payment amount field."

The implementer adds a check for `amount > 0`. It sees `999999999.99` as valid.

The adversarial reviewer sees:
- No upper bound? A user could submit `2^63-1` and overflow the database column.
- No type check? The field accepts strings that `parseFloat` coerces to `NaN`, which `> 0` returns `false` for — silently rejecting valid-looking inputs.
- No test for the `NaN` case.

Without adversarial review, the PR passes CI (the existing tests still pass) and merges with a latent bug. With adversarial review, these issues are caught before a PR is opened.

## When to use which workflow

**Use the shave loop (`v`) when:**
- You're implementing a single yak and want confidence before opening a PR
- The yak is complex enough that bugs are likely
- You want to save human reviewer time by catching issues first
- You're working on a yak that will take more than one attempt

**Use the barber (`s`) when:**
- You want continuous, unattended shaving of `@g2g` yaks
- The yaks are simple enough that CI is a sufficient gate
- You have human reviewers providing PR feedback signals
- You want the workflow to handle the entire backlog

The two workflows do not conflict. The barber processes `@g2g` yaks continuously. The shave loop tackles one yak deeply. Both use the same yak claim mechanism (`yx start`), so they cannot claim the same yak simultaneously.

## Design decisions

**Why a separate workflow instead of a barber mode?** The shave loop has fundamentally different semantics: fixed retries, isolated workspace, pre-PR review. Baking these into the barber would complicate a workflow that's intentionally simple (signal → triage → dispatch → CI → merge). A separate workflow keeps each one's invariants clear.

**Why 3 default retries?** Three iterations is enough for most tasks: first attempt implements, second fixes validation/lint, third addresses deeper review concerns. More than 3 usually indicates the yak is underspecified and needs human input — which is why the workflow tags it `@needs-human`.

**Why `jj workspace` instead of a git branch?** Workspaces are fully isolated checkouts. The agent can make breaking changes, run tests, and iterate without affecting the main checkout. If the workspace needs to be abandoned (all retries exhausted), `jj workspace forget` cleans up cleanly.

> For the workflow that implements this pattern, see [ShaveWorkflow Reference](../reference/shave-workflow.md).
> For the barber workflow, see [Architecture](architecture.md).
> For the specification pattern the shave loop implements, see the [shave-specs skill](https://github.com/funkymonkeymonk/yaketyyak) (`shave-specs`).
