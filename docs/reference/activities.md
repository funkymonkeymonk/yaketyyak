# Activities Reference

All Temporal activities used by `YakWorkflow`.

## YakClaim

Claims the yak using `yx start` and syncs state with `yx sync`.

- Input: `yakName: string`
- Timeout: 30s
- Retry: 3 attempts

## YakRelease

Releases a claimed yak (marks it back to `todo`) if the workflow cannot proceed. Runs `yx sync` after release.

- Input: `yakName: string`, `reason: string`
- Timeout: 60s
- Retry: 1 attempt

## YakMarkDone

Marks the yak as done using `yx done` and syncs with `yx sync`.

- Input: `yakName: string`
- Timeout: 30s
- Retry: 3 attempts

## WritePRToYak

Writes the PR URL into the yak's custom field so it is visible in `yx show`.

- Input: `yakName: string`, `prUrl: string`
- Timeout: 30s
- Retry: 3 attempts

## InitWorkspace

Clones the repository into an isolated workspace directory under `.workspaces/` and creates a new branch for the shave.

Runs:
1. `rmSync` — remove any stale workspace from a previous run
2. `git clone --depth 1 <authenticated-url> .workspaces/shave-<slug>`
3. `git checkout -b shave/<slug>`
4. Sets `user.email` and `user.name` for commits in the workspace

- Input: `repoUrl: string`, `yakName: string`
- Returns: workspace name string (e.g. `"shave-update-docs"`)
- Timeout: 60s
- Retry: 1 attempt

## CleanupWorkspace

Removes the isolated workspace directory with `rmSync`.

- Input: `workspaceName: string`
- Timeout: 60s
- Retry: 1 attempt

## RunAgent

Dispatches Pi via LiteLLM to implement the yak in the workspace. Reads the yak context with `yx show --format json`, writes it to `.yak-context.md` inside the workspace, and prompts Pi to implement it and commit.

- Input: `yakName: string`, `workspaceName: string`, `cfg: PiConfig`
- Timeout: 4h
- Heartbeat timeout: 2 minutes
- Retry: 1 attempt (no automatic retry — agent failures surface to the workflow)

See [PiConfig](data-types.md#piconfig) for agent configuration.

## CreateDraftPR

Pushes the workspace branch to GitHub and opens a draft PR using the Octokit REST API.

- Input: `repoUrl: string`, `workspaceName: string`, `yakName: string`
- Returns: [`PRResult`](data-types.md#prresult)
- Timeout: 5m
- Retry: 1 attempt

## WatchPRMerged

Polls the GitHub API every 60 seconds until the PR is merged or closed without merging. Returns `true` if merged, `false` if closed without merge.

- Input: `prNumber: number`, `repoUrl: string`
- Returns: `boolean`
- Poll interval: 60s
- Timeout: 168h (7 days)
- Heartbeat timeout: 2 minutes
- Retry: 1 attempt

> For the workflow that orchestrates these activities, see [YakWorkflow Reference](yak-workflow.md).
> For signal definitions, see [Signals Reference](signals.md).
