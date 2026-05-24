# Architecture

yaketyyak orchestrates autonomous yak shaving using Temporal.io as a durable execution backbone.

## Why Temporal?

Temporal provides four properties that map directly to the problem of CI-triggered agent loops:

**Durability.** If the worker process crashes, the workflow resumes from the last completed event. No state is lost. This matters when a PR review arrives three hours after CI passed.

**Signals.** CI completion, PR feedback, and manual commands are all Temporal signals. The workflow processes them as they arrive, in order, without polling.

**Long-running workflows.** A single workflow can live for days or weeks — however long it takes from yak claim to PR merge. Temporal handles this natively with event history and continue-as-new.

**Retry policies.** Every side effect (yx CLI, GitHub API, agent subprocess) has a configurable retry policy. Network blips and rate limits are handled automatically.

## System context

```
┌─────────────┐
│   yx CLI    │ ── yak state (git refs)
└──────┬──────┘
       │ yx sync / yx ls / yx start / yx done
       ▼
┌──────────────────────────────────────────┐
│          BarberWorkflow                       │
│                                          │
│  ┌──────────┐   ┌──────────────────┐     │
│  │ Signals  │──▶│ Main Loop        │     │
│  │ (ci, g2g,│   │                  │     │
│  │  pr, etc)│   │  triage → claim  │     │
│  └──────────┘   │  → implement     │     │
│                  │  → watch CI      │     │
│  ┌──────────┐   │  → feedback loop │     │
│  │ Queries  │──▶│  → merge         │     │
│  │ (status) │   │  → mark done     │     │
│  └──────────┘   └────────┬─────────┘     │
└──────────────────────────┼───────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │   gh CLI │ │ Agent CLI│ │  GitHub  │
        │ (pr, ci) │ │(pi/codex/│ │  API     │
        │          │ │ claude)  │ │(webhook) │
        └──────────┘ └──────────┘ └──────────┘
```

## Flow details

### Trigger sources

| Source | Mechanism | Latency |
|--------|-----------|---------|
| GitHub CI completion | `workflow_run` webhook → `ci_signal` | Seconds |
| `@g2g` tag on yak | `g2g_signal` from CLI or script | Instant |
| Periodic scan | Idle timeout in main loop | Configurable (default 60m) |
| PR review comment | `pr_feedback` signal from CLI | Manual |

### Yak lifecycle within the workflow

```
todo → claimed → wip → PR created → CI running → merged → done
                         │               │
                         │ (rework)      │ (failure)
                         ▼               ▼
                     PR updated      agent re-dispatched
```

The workflow never exits the yak lifecycle until the PR is merged or the yak is explicitly failed.

## Design decisions

**Why activities instead of direct subprocess calls?** Activities get retry policies, timeouts, and heartbeats. A long-running agent dispatch can heartbeat to avoid timeout. A failing `gh pr merge` retries automatically.

**Why a single workflow per repo?** Idempotency. If two instances run, they'd conflict on yak claims. The fixed workflow ID ensures exactly one barber per repo. Signal-With-Start would create on demand if needed.

**Why g2g-only mode?** Safety. By default, the workflow only processes yaks explicitly tagged `@g2g`. This prevents it from claiming every `todo` yak and making unwanted changes. The tag is the human's commit signal.

> For alternatives considered, see [Why Temporal](why-temporal.md).
> For the single-yak shave loop workflow with adversarial review, see [Barber and Shave Workflows](barber-and-shave-workflows.md).
> For how the workflow was designed to integrate with the existing shave-yaks ecosystem, see the [nix repo](https://github.com/funkymonkeymonk/nix).
