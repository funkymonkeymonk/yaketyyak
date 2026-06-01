> **This document is superseded.** The workflow described here (`ShaveWorkflow`) does not exist in the current codebase. See [YakWorkflow Reference](yak-workflow.md) for the real workflow.

# ShaveWorkflow Reference

A standalone Temporal workflow that shaves a single yak through the shave loop pattern: implement → validate → adversarial review → create PR → merge.

```
Workflow name:     ShaveWorkflow
Workflow ID:       yy-shave-<yak-name-slug>
Task queue:        yaketyyak-tasks
Registration:      temporal/worker.go
Query handler:     shave_status
Signal channel:    shave_cancel
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `yak_name` | string | yes | — | Name of the yak to shave |
| `agent_type` | string | yes | — | Agent CLI: `pi`, `claude-code`, `codex`, `opencode` |
| `repo` | string | yes | — | GitHub repo in `owner/repo` format |
| `repo_root` | string | yes | — | Absolute path to local checkout |
| `max_retries` | int | no | `3` | Maximum implement-fix-review iterations |

## Phases

The workflow cycles through these phases, settable via the `shave_status` query:

| Phase | Description |
|-------|-------------|
| `init` | Workflow starting |
| `syncing` | Running `yx sync` |
| `claiming` | Claiming the yak via `yx start` |
| `init-workspace` | Creating isolated jj workspace |
| `implementing` | Agent writes tests + code |
| `validating` | Running `go vet`, `go test`, `go build` |
| `reviewing` | Adversarial reviewer examines the diff |
| `accepted` | Validation and review both passed |
| `creating-pr` | Pushing branch, opening PR |
| `watching-ci` | Polling GitHub checks |
| `merging` | Merging passing PR |
| `done` | Yak marked done, workflow complete |
| `needs-human` | Max retries exhausted or CI failed, yak tagged `@needs-human` |
| `cancelled` | Workflow cancelled via `shave_cancel` signal |

## Query: shave_status

Returns a `ShaveState` JSON object:

```json
{
  "yak_name": "Add unit tests for payment module",
  "agent_type": "opencode",
  "repo": "owner/repo",
  "repo_root": "/path/to/repo",
  "max_retries": 3,
  "iteration": 2,
  "phase": "reviewing",
  "workspace": "shave-add-unit-tests-for-payment-module",
  "pr_number": 0,
  "pr_url": ""
}
```

Query from the CLI:

```bash
temporal workflow query \
    --workflow-id yy-shave-add-unit-tests \
    --type shave_status
```

## Signal: shave_cancel

Cancels the workflow at the next iteration boundary. Any in-flight activity continues to completion, but the next iteration is skipped.

```bash
temporal workflow signal \
    --workflow-id yy-shave-add-unit-tests \
    --name shave_cancel
```

## Error handling

| Failure point | Behavior |
|---------------|----------|
| Claim fails | Workflow exits with error (yak already claimed or doesn't exist) |
| Workspace init fails | Workflow exits with error |
| All retries exhausted | Yak tagged `@needs-human`, workflow completes with status message |
| PR creation fails | Workflow exits with error |
| CI fails | Yak tagged `@needs-human`, workflow completes with status message |
| Workflow interrupted | Deferred handler marks yak `@needs-human` with phase at time of interruption |

## Deferred cleanup

On completion (success or failure), the workflow:
1. Marks the yak `@needs-human` if interrupted in a non-terminal phase
2. Runs `jj workspace forget` on the isolated workspace

> For the individual activities that power this workflow, see [Shave Activities Reference](shave-activities.md).
> For how to trigger this workflow, see [Shave a Yak](../how-to/shave-a-yak.md).
> For the design rationale, see [Barber and Shave Workflows](../explanation/barber-and-shave-workflows.md).
