# YakWorkflow Reference

A Temporal workflow that shaves a single yak autonomously: claim → implement → PR → merge → done.

```
Workflow name:  YakWorkflow
Workflow ID:    yy-yak-<yak-name-slug>
Task queue:     yaketyyak-tasks
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `yakName` | string | yes | — | Name of the yak to shave |
| `repoRoot` | string | yes | — | Absolute path to local checkout |
| `cfg` | PiConfig | yes | — | Pi agent configuration (model, tools, skills) |

`repoRoot` defaults to the current directory when `yy shave` is invoked without `--repo-root`.

See [PiConfig](data-types.md#piconfig) for agent configuration details.

## Phases

The workflow progresses through these phases, visible via the `YakWorkflowState` query:

| Phase | Description |
|-------|-------------|
| `claiming` | Running `yx start` to claim the yak |
| `init-workspace` | Creating isolated jj workspace under `.workspaces/` |
| `implementing` | Pi agent is running in the workspace |
| `creating-pr` | Pushing branch and opening draft PR |
| `watching-pr` | Polling for the PR to be merged or closed |
| `done` | PR merged, yak marked done with `yx done` |
| `failed` | Unrecoverable error; yak released |

## Activity sequence

```
YakClaim
  └─ InitWorkspace
       └─ RunAgent
            └─ CreateDraftPR
                 └─ WritePRToYak
                      └─ WatchPRMerged
                           └─ YakMarkDone
                                └─ CleanupWorkspace
```

On any unrecoverable failure: `YakRelease` + `CleanupWorkspace`.

## Query: YakWorkflowState

Returns the current workflow state as `YakWorkflowState` JSON:

```json
{
  "yak_name": "update documentation to reflect current architecture",
  "phase": "implementing",
  "workspace": ".workspaces/shave-update-documentation-to-reflect-current-architecture",
  "pr_url": "",
  "pr_number": 0
}
```

Query via Temporal CLI:

```bash
temporal workflow query \
    --workflow-id yy-yak-update-documentation \
    --type YakWorkflowState
```

## Signal: wont-do

Cancels the workflow at the next activity boundary. The yak is released (returned to `todo`) and tagged `@needs-human`.

```bash
temporal workflow signal \
    --workflow-id yy-yak-update-documentation \
    --name wont-do
```

## Error handling

| Failure point | Behaviour |
|---------------|-----------|
| Claim fails | Workflow exits with error (yak not found or already wip) |
| Workspace init fails | Yak released; workflow exits with error |
| Agent fails | Yak released; workspace cleaned up; workflow exits with error |
| PR creation fails | Yak released; workspace cleaned up; workflow exits with error |
| PR closed without merging | Yak released; workspace cleaned up; workflow exits |
| Workflow interrupted | Deferred handler releases yak and cleans up workspace |

## Workspace naming

The workspace path is derived from the yak name:

```
.workspaces/shave-<yak-name-slug>
```

For example, yak `update docs` → `.workspaces/shave-update-docs`.

> For the individual activities, see [Activities Reference](activities.md).
> For the data types, see [Data Types Reference](data-types.md).
> For how to start the workflow, see [Start the Workflow](../how-to/start-the-workflow.md).
