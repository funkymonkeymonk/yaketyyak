> **This feature is not yet implemented.** This document describes planned behaviour.

# How to Integrate with GitHub Actions

Automatic CI-triggered yak shaving is not yet implemented. The `ci_signal` and the CLI commands that send it (`yyx ci`) do not exist in the current codebase.

When this feature is implemented, it will allow a GitHub Actions workflow to signal `YakWorkflow` instances on CI completion, enabling a fully automated CI → implement → PR → merge loop.

> For the current way to start a shave, see [Start the Workflow](start-the-workflow.md).
> For implemented signal definitions, see [Signals Reference](../reference/signals.md).
