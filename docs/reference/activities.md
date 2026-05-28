# Activities Reference

All Temporal activities used by `YakWorkflow`.

## YakClaim

Claims the yak using `yx start` and syncs state with `yx sync`.

- Input: `yakName string`
- Timeout: 30s
- Retry: 3 attempts

## YakRelease

Releases a claimed yak (marks it back to `todo`) if the workflow cannot proceed. Uses `yx sync` after release.

- Input: `yakName string`
- Timeout: 30s
- Retry: 3 attempts

## YakMarkDone

Marks the yak as done using `yx done` and syncs with `yx sync`.

- Input: `yakName string`
- Timeout: 30s
- Retry: 3 attempts

## WritePRToYak

Writes the PR URL and number into the yak's context field so it is visible in `yx show`.

- Input: `yakName string`, `prURL string`, `prNumber int`
- Timeout: 30s
- Retry: 3 attempts

## InitWorkspace

Creates an isolated jj workspace for the yak under `.workspaces/`.

Runs:
1. `jj git fetch` — pull latest from remote
2. `jj workspace add --name shave-<slug> .workspaces/shave-<slug>`

- Input: `repoRoot string`, `yakName string`
- Returns: workspace path string (e.g. `.workspaces/shave-update-docs`)
- Timeout: 60s
- Retry: 2 attempts

## CleanupWorkspace

Removes the isolated jj workspace with `jj workspace forget`.

- Input: `repoRoot string`, `workspacePath string`
- Timeout: 30s
- Retry: 2 attempts

## RunAgent

Dispatches Pi via LiteLLM to implement the yak in the workspace. Pi receives the yak context as its prompt.

- Input: `PiConfig`, `yakName string`, `workspacePath string`
- Timeout: 2h
- Retry: 1 attempt (no automatic retry — agent failures are surfaced to the workflow)

See [PiConfig](data-types.md#piconfig) for agent configuration.

## CreateDraftPR

Pushes the workspace branch to GitHub and opens a draft PR using `gh pr create --draft`.

- Input: `repoRoot string`, `workspacePath string`
- Returns: [`PRResult`](data-types.md#prresult)
- Timeout: 5m
- Retry: 2 attempts

## WatchPRMerged

Polls the GitHub API until the PR is merged or closed. Returns `"merged"` or `"closed"`.

- Input: `prNumber int`
- Poll interval: 30s
- Timeout: 168h (7 days)
- Retry: 1 attempt

> For the workflow that orchestrates these activities, see [YakWorkflow Reference](yak-workflow.md).
> For signal definitions, see [Signals Reference](signals.md).
