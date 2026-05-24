# How to Send Signals to the Workflow

yaketyyak uses Temporal signals to communicate with the running workflow. Each signal triggers a specific behavior.

## CI signal

Triggered when a CI pipeline completes:

```bash
yyx ci \
    --conclusion failure \
    --branch feat/foo \
    --sha abc1234
```

The workflow syncs yaks, triages g2g yaks, and starts implementing.

## g2g scan signal

Triggers an immediate check for `@g2g`-tagged yaks:

```bash
yyx g2g-scan
```

Use this after tagging a new yak as ready.

## PR feedback signal

Send review comments back into the rework loop:

```bash
yyx pr-feedback \
    --pr-number 42 \
    --comment "Fix the lint warning in payment.py" \
    --author reviewer
```

The workflow re-dispatches the agent to address feedback, then re-watches CI.

## Pause and resume

Pause processing:

```bash
yyx pause
```

Resume:

```bash
yyx resume
```

When paused, the workflow ignores all signals. They queue up and are processed on resume.

## Check status

```bash
yyx status
```

Returns the current workflow phase, active yak, and completion counts.

> For full signal definitions, see [Signals Reference](../reference/signals.md).
> For the CI-driven tutorial, see [CI-Driven Loop](../tutorials/ci-driven-loop.md).
> To shave a single yak with the shave loop from the TUI, see [Shave a Yak](shave-a-yak.md).
