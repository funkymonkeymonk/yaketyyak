# Signals Reference

All signals that can be sent to the `BarberWorkflow`.

## ci_signal

Sent when a CI pipeline completes.

| Parameter | Type | Description |
|-----------|------|-------------|
| `conclusion` | string | `"success"`, `"failure"`, or `"cancelled"` |
| `branch` | string | Git branch that was built |
| `sha` | string | Commit SHA |
| `details` | string | Optional JSON with run IDs, check URLs |

```bash
yyx ci \
    --conclusion failure \
    --branch feat/foo \
    --sha abc1234
```

## g2g_signal

Triggers an immediate scan for yaks tagged `@g2g`.

Parameters: none

```bash
yyx g2g-scan
```

## pr_feedback

Sends a PR review comment into the rework loop.

| Parameter | Type | Description |
|-----------|------|-------------|
| `pr_number` | int | Pull request number |
| `comment` | string | Review body |
| `author` | string | Reviewer handle |

```bash
yyx pr-feedback \
    --pr-number 42 \
    --comment "Fix the lint warning" \
    --author reviewer
```

## pause

Pauses the workflow. All incoming signals are queued and processed on resume.

```bash
yyx pause
```

## resume

Resumes a paused workflow.

```bash
yyx resume
```

## Query: status

Returns the current workflow state. Not a signal — a Temporal query (read-only).

```bash
yyx status
```

Output:

```json
{
  "phase": "implementing",
  "current_yak": {
    "name": "Add unit tests for payment module",
    "state": "wip",
    "g2g": true,
    "pr_number": 42,
    "pr_url": "https://github.com/owner/repo/pull/42"
  },
  "completed_yaks": 1,
  "failed_yaks": 0
}
```

> For how to send signals in CI, see [Integrate with GitHub Actions](../how-to/integrate-github-actions.md).
> For agent dispatch details, see [Activities Reference](activities.md).
