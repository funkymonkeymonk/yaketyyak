# YakWorkflow Reference

A Temporal workflow that shaves a single yak autonomously: claim → implement → PR → merge → done.

```
Workflow name:  YakWorkflow
Workflow ID:    yyx-yak-<yak-name-slug>
Task queue:     yaketyyak-tasks
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `yakName` | string | yes | — | Name of the yak to shave |
| `cfg` | WorkflowConfig | yes | — | Repo URL and Pi agent configuration |

`cfg.repoUrl` is the GitHub repository URL (e.g. `https://github.com/owner/repo`). See [PiConfig](data-types.md#piconfig) for agent configuration details.

## Phases

The workflow progresses through these phases, visible via the `yak_status` query:

| Phase | Description |
|-------|-------------|
| `init` | Workflow just started |
| `claiming` | Running `yx start` to claim the yak |
| `init-workspace` | Cloning repo into isolated workspace under `.workspaces/` |
| `implementing` | Pi agent is running in the workspace |
| `creating-pr` | Pushing branch and opening draft PR |
| `waiting-for-merge` | Polling GitHub API for PR merge |
| `done` | PR merged, yak marked done with `yx done` |

## Activity sequence

```
YakClaim
  └─ InitWorkspace
       └─ RunAgent
            └─ CreateDraftPR
                 └─ WritePRToYak (fire-and-forget)
                      └─ WatchPRMerged
                           └─ YakMarkDone
                                └─ CleanupWorkspace (deferred)
```

On any unrecoverable failure: `YakRelease` + `CleanupWorkspace`.

## Query: yak_status

Returns the current workflow state as `YakWorkflowState` JSON:

```json
{
  "yakName": "update documentation to reflect current architecture",
  "phase": "implementing",
  "workspace": "shave-update-documentation-to-reflect-current-architecture-7yf2",
  "prUrl": "",
  "prNumber": 0
}
```

Query via Temporal CLI:

```bash
temporal workflow query \
    --workflow-id yyx-yak-update-documentation \
    --type yak_status
```

## Signal: wont-do

Cancels the workflow at the next activity boundary. The yak is released (returned to `todo`).

```bash
temporal workflow signal \
    --workflow-id yyx-yak-update-documentation \
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

The workspace name is derived from the yak name by lower-casing and replacing non-alphanumeric characters with hyphens:

```
shave-<yak-name-slug>
```

For example, yak `update docs` → `shave-update-docs`.

The full path on disk is `.workspaces/shave-<slug>` relative to the worker's working directory.

## Workflow ID

The workflow ID is deterministic, derived from the yak name:

```
yyx-yak-<sanitized-yak-name>
```

For example, yak `update docs` → `yyx-yak-update-docs`.

This means only one `YakWorkflow` can run per yak at a time — attempting to start a second workflow for the same yak will fail with a workflow already exists error.

> For the individual activities, see [Activities Reference](activities.md).
> For the data types, see [Data Types Reference](data-types.md).
> For how to start the workflow, see [Start the Workflow](../how-to/start-the-workflow.md).
