# Signals Reference

Signals that can be sent to a running `YakWorkflow`.

## wont-do

Cancels the workflow and releases the claimed yak. The yak is returned to `todo` state and tagged `@needs-human`.

```bash
temporal workflow signal \
    --workflow-id yyx-yak-<yak-name-slug> \
    --name wont-do
```

Use this when a running shave should be abandoned — for example, if the yak turned out to be out of scope, or Pi is clearly going in the wrong direction.

---

## Not yet implemented

The following signals are planned but not yet implemented.

| Signal | Planned purpose |
|--------|----------------|
| `ci_signal` | Notify the workflow of a CI pipeline result |
| `pr_feedback` | Send PR review comments back to the agent |
| `pause` | Pause the workflow; queue incoming signals |
| `resume` | Resume a paused workflow |

> For the workflow definition, see [YakWorkflow Reference](yak-workflow.md).
> For activity details, see [Activities Reference](activities.md).
