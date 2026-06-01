> **This feature is not yet implemented.** This document describes planned behaviour.

# How to Send Signals to the Workflow

Most workflow signals beyond `wont-do` are not yet implemented.

## wont-do (implemented)

To cancel a running shave:

```bash
temporal workflow signal \
    --workflow-id yy-yak-<yak-name-slug> \
    --name wont-do
```

See [Signals Reference](../reference/signals.md) for details.

---

## Not yet implemented

The following signals are planned but not yet implemented:

| Signal | CLI command | Planned purpose |
|--------|-------------|----------------|
| `ci_signal` | `yy ci` | Notify the workflow of a CI pipeline result |
| `g2g_signal` | `yy g2g-scan` | Trigger an immediate scan for `@g2g`-tagged yaks |
| `pr_feedback` | `yy pr-feedback` | Send PR review comments back to the agent |
| `pause` | `yy pause` | Pause the workflow |
| `resume` | `yy resume` | Resume a paused workflow |

> For all implemented signal definitions, see [Signals Reference](../reference/signals.md).
