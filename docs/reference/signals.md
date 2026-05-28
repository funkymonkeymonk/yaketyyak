# Signals Reference

Signals that can be sent to a running `YakWorkflow`.

## wont-do

Cancels the workflow and releases the claimed yak. The yak is returned to `todo` state.

```bash
temporal workflow signal \
    --workflow-id yyx-yak-<yak-name-slug> \
    --name wont-do
```

Use this when a running shave should be abandoned — for example, if the yak turned out to be out of scope, or Pi is clearly going in the wrong direction.

## Query: yak_status

Returns the current workflow state as a `YakWorkflowState` JSON object. Not a signal — a Temporal query (read-only).

```bash
temporal workflow query \
    --workflow-id yyx-yak-<yak-name-slug> \
    --type yak_status
```

Output:

```json
{
  "yakName": "update documentation to reflect current architecture",
  "phase": "implementing",
  "workspace": "shave-update-documentation-to-reflect-current-architecture-7yf2",
  "prUrl": "",
  "prNumber": 0
}
```

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
