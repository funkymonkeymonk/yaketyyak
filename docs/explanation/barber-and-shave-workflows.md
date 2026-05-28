> **This feature is not yet implemented.** This document describes planned behaviour.

# Barber and Shave Workflows

The `BarberWorkflow` and `ShaveWorkflow` described here do not exist in the current codebase. The only implemented workflow is `YakWorkflow` — a single-yak workflow invoked with `yyx shave`.

This document describes the planned design for a future continuous barber that monitors CI, processes `@g2g` yaks, and runs adversarial review before opening PRs.

> For the real workflow, see [YakWorkflow Reference](../reference/yak-workflow.md).
> For the design rationale behind Temporal in general, see [Why Temporal](why-temporal.md).
